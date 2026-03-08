package voice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/idgen"
	"nova-echoes/api/internal/modules/patients/preferences"
	"nova-echoes/api/internal/modules/voicecatalog"
)

var (
	errClientDisconnected   = errors.New("client disconnected")
	errClientRequestedClose = errors.New("client requested close")
)

const defaultStartCallPrompt = "Start the call now. Greet the person warmly, then ask your first short question."

type Service struct {
	cfg              config.Config
	voiceCatalog     voicecatalog.Catalog
	repo             Repository
	preferencesStore preferences.Store
	liveStarter      LiveSessionStarter
	artifactExporter ArtifactExporter
	sessions         *SessionManager
	now              func() time.Time
}

func NewService(cfg config.Config, voiceCatalog voicecatalog.Catalog, repo Repository, preferencesStore preferences.Store, liveStarter LiveSessionStarter, artifactExporter ArtifactExporter, sessions *SessionManager) *Service {
	if artifactExporter == nil {
		artifactExporter = NewNoopArtifactExporter()
	}

	return &Service{
		cfg:              cfg,
		voiceCatalog:     voiceCatalog,
		repo:             repo,
		preferencesStore: preferencesStore,
		liveStarter:      liveStarter,
		artifactExporter: artifactExporter,
		sessions:         sessions,
		now:              func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) ListVoices() []voicecatalog.Voice {
	return s.voiceCatalog.Allowed()
}

func (s *Service) CreateSession(ctx context.Context, input CreateSessionRequest) (SessionDescriptor, error) {
	patientID := strings.TrimSpace(input.PatientID)
	if patientID == "" {
		return SessionDescriptor{}, ErrPatientIDRequired
	}

	systemPrompt := strings.TrimSpace(input.SystemPrompt)
	if len(systemPrompt) > maxSystemPromptBytes {
		return SessionDescriptor{}, ErrSystemPromptTooLarge
	}

	voiceID, err := s.resolveVoiceID(ctx, patientID, strings.TrimSpace(input.VoiceID))
	if err != nil {
		return SessionDescriptor{}, err
	}

	sessionID, err := idgen.New()
	if err != nil {
		return SessionDescriptor{}, err
	}

	streamToken, streamTokenHash, err := newStreamToken()
	if err != nil {
		return SessionDescriptor{}, err
	}

	now := s.now()
	expiresAt := now.Add(streamTokenTTL)
	record := SessionRecord{
		ID:                     sessionID,
		PatientID:              patientID,
		Status:                 StatusAwaitingStream,
		VoiceID:                voiceID,
		SystemPrompt:           systemPrompt,
		InputSampleRateHz:      s.cfg.NovaInputSampleRate,
		OutputSampleRateHz:     s.cfg.NovaOutputSampleRate,
		EndpointingSensitivity: s.cfg.NovaEndpointingSensitivity,
		ModelID:                s.cfg.NovaVoiceModelID,
		AWSRegion:              s.cfg.AWSRegion,
		BedrockRegion:          s.cfg.BedrockRegion,
		StreamTokenHash:        streamTokenHash,
		StreamTokenExpiresAt:   expiresAt,
		LastActivityAt:         now,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := s.repo.CreateSession(ctx, record); err != nil {
		return SessionDescriptor{}, err
	}

	return SessionDescriptor{
		ID:                   sessionID,
		VoiceID:              voiceID,
		WebSocketPath:        fmt.Sprintf("/api/v1/voice/sessions/%s/stream", sessionID),
		StreamToken:          streamToken,
		StreamTokenExpiresAt: expiresAt,
		AudioInput: AudioConfig{
			Encoding:     "pcm_s16le",
			SampleRateHz: s.cfg.NovaInputSampleRate,
			Channels:     1,
		},
		AudioOutput: AudioConfig{
			Encoding:     "pcm_s16le",
			SampleRateHz: s.cfg.NovaOutputSampleRate,
			Channels:     1,
		},
		DrainSeconds:      drainSeconds,
		MaxSessionSeconds: maxSessionSeconds,
	}, nil
}

func (s *Service) Attach(ctx context.Context, sessionID, token string, conn *websocket.Conn) error {
	record, err := s.repo.ConsumeAttachToken(ctx, sessionID, hashToken(token), s.now())
	if err != nil {
		return err
	}

	promptName, err := idgen.New()
	if err != nil {
		_ = s.repo.MarkSessionEnded(ctx, sessionID, StatusFailed, "", "id_generation_failed", err.Error(), s.now())
		return err
	}

	audioContentName, err := idgen.New()
	if err != nil {
		_ = s.repo.MarkSessionEnded(ctx, sessionID, StatusFailed, "", "id_generation_failed", err.Error(), s.now())
		return err
	}

	liveSession, err := s.liveStarter.StartSession(ctx, StartLiveSessionInput{
		ModelID:                record.ModelID,
		VoiceID:                record.VoiceID,
		SystemPrompt:           record.SystemPrompt,
		PromptName:             promptName,
		AudioContentName:       audioContentName,
		InputSampleRateHz:      record.InputSampleRateHz,
		OutputSampleRateHz:     record.OutputSampleRateHz,
		EndpointingSensitivity: record.EndpointingSensitivity,
	})
	if err != nil {
		_ = s.repo.MarkSessionEnded(ctx, sessionID, StatusFailed, "", "bedrock_start_failed", err.Error(), s.now())
		return err
	}

	streamStartedAt := s.now()
	sessionExpiresAt := streamStartedAt.Add(maxSessionSeconds * time.Second)
	if err := s.repo.MarkSessionStreaming(ctx, sessionID, promptName, sessionExpiresAt, streamStartedAt); err != nil {
		_ = liveSession.Close()
		_ = s.repo.MarkSessionEnded(ctx, sessionID, StatusFailed, "", "session_mark_failed", err.Error(), s.now())
		return err
	}

	runtime := &runtimeSession{
		repo:             s.repo,
		live:             liveSession,
		conn:             conn,
		sessionID:        sessionID,
		record:           record,
		promptName:       promptName,
		sessionExpiresAt: sessionExpiresAt,
		contents:         make(map[string]*outputContentState),
		artifactExporter: s.artifactExporter,
		lastTouchAt:      streamStartedAt,
		now:              s.now,
	}

	if !s.sessions.Add(sessionID, runtime) {
		_ = liveSession.Close()
		_ = s.repo.MarkSessionEnded(ctx, sessionID, StatusFailed, "", "duplicate_attach", "voice session already active", s.now())
		return ErrStreamConsumed
	}
	defer s.sessions.Delete(sessionID)

	if err := runtime.writeJSON(map[string]any{
		"type":             wsMessageSessionReady,
		"voiceSessionId":   sessionID,
		"sessionExpiresAt": sessionExpiresAt,
	}); err != nil {
		_ = s.repo.MarkSessionEnded(ctx, sessionID, StatusFailed, "", "websocket_write_failed", err.Error(), s.now())
		_ = runtime.Close()
		return err
	}

	return runtime.run(ctx)
}

type runtimeSession struct {
	repo             Repository
	live             LiveSession
	conn             *websocket.Conn
	sessionID        string
	record           SessionRecord
	promptName       string
	sessionExpiresAt time.Time
	artifactExporter ArtifactExporter
	now              func() time.Time

	sendMu      sync.Mutex
	sequenceMu  sync.Mutex
	stateMu     sync.Mutex
	activityMu  sync.Mutex
	closeOnce   sync.Once
	startOnce   sync.Once
	contents    map[string]*outputContentState
	turnSeq     int
	usageSeq    int
	lastStop    string
	lastTouchAt time.Time
	clientAlive atomic.Bool
}

func (r *runtimeSession) run(ctx context.Context) error {
	r.clientAlive.Store(true)
	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer r.Close()

	readErrCh := make(chan error, 1)
	eventErrCh := make(chan error, 1)

	go func() {
		readErrCh <- r.readClientMessages(sessionCtx)
	}()

	go func() {
		eventErrCh <- r.handleModelEvents(sessionCtx)
	}()

	select {
	case err := <-readErrCh:
		isDisconnect := errors.Is(err, errClientDisconnected)
		if isDisconnect {
			r.clientAlive.Store(false)
		}
		now := r.now()
		if isDisconnect {
			graceUntil := now.Add(drainSeconds * time.Second)
			_ = r.repo.MarkDisconnectGrace(sessionCtx, r.sessionID, now, graceUntil)
		}
		_ = r.live.EndConversation(sessionCtx)

		timer := time.NewTimer(drainSeconds * time.Second)
		defer timer.Stop()

		var modelErr error
		select {
		case modelErr = <-eventErrCh:
		case <-timer.C:
			_ = r.live.Close()
			modelErr = <-eventErrCh
		}

		return r.finalize(sessionCtx, combineRuntimeErrors(err, modelErr))
	case err := <-eventErrCh:
		cancel()
		return r.finalize(sessionCtx, err)
	}
}

func (r *runtimeSession) Close() error {
	var closeErr error
	r.closeOnce.Do(func() {
		if err := r.live.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
		if err := r.conn.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	})
	return closeErr
}

func (r *runtimeSession) readClientMessages(ctx context.Context) error {
	r.conn.SetReadLimit(64 * 1024)

	for {
		messageType, payload, err := r.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("%w: %v", errClientDisconnected, err)
		}

		switch messageType {
		case websocket.BinaryMessage:
			if err := r.live.SendAudio(ctx, payload); err != nil {
				return err
			}
			r.touchSession(ctx, r.now())
		case websocket.TextMessage:
			var message clientMessage
			if err := json.Unmarshal(payload, &message); err != nil {
				return fmt.Errorf("decode client message: %w", err)
			}

			switch message.Type {
			case wsMessageStartCall:
				if err := r.startCall(ctx); err != nil {
					return err
				}
			case wsMessageTextInput:
				text := strings.TrimSpace(message.Text)
				if text == "" {
					continue
				}

				if err := r.live.SendText(ctx, text); err != nil {
					return err
				}

				if err := r.persistClientTextTurn(ctx, text); err != nil {
					return err
				}
			case wsMessageClientClose:
				return errClientRequestedClose
			default:
				return fmt.Errorf("unsupported websocket message type %q", message.Type)
			}
		}
	}
}

func (r *runtimeSession) handleModelEvents(ctx context.Context) error {
	for event := range r.live.Events() {
		if event.Err != nil {
			return event.Err
		}

		if len(event.Payload) == 0 {
			continue
		}

		if err := r.processModelPayload(ctx, event.Payload); err != nil {
			return err
		}
	}

	return nil
}

func (r *runtimeSession) startCall(ctx context.Context) error {
	var startErr error
	r.startOnce.Do(func() {
		startErr = r.live.SendText(ctx, defaultStartCallPrompt)
		if startErr == nil {
			r.touchSession(ctx, r.now())
		}
	})

	return startErr
}

func (r *runtimeSession) processModelPayload(ctx context.Context, payload []byte) error {
	var envelope outputEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decode bedrock payload: %w", err)
	}

	now := r.now()
	r.touchSession(ctx, now)

	switch {
	case envelope.Event.CompletionStart != nil:
		event := envelope.Event.CompletionStart
		if event.SessionID != "" {
			r.record.BedrockSessionID = event.SessionID
			if event.PromptName != "" {
				r.record.PromptName = event.PromptName
			}
			_ = r.repo.UpdateSessionMetadata(ctx, r.sessionID, event.SessionID, event.PromptName, nil, now)
		}
	case envelope.Event.ContentStart != nil:
		event := envelope.Event.ContentStart
		stage := parseGenerationStage(event.AdditionalModelFields)
		r.stateMu.Lock()
		r.contents[event.ContentID] = &outputContentState{
			Direction:        normalizeDirection(event.Role),
			Modality:         normalizeModality(event.Type),
			GenerationStage:  stage,
			BedrockSessionID: event.SessionID,
			PromptName:       event.PromptName,
			CompletionID:     event.CompletionID,
			ContentID:        event.ContentID,
			OccurredAt:       now,
		}
		r.stateMu.Unlock()

		if event.SessionID != "" {
			r.record.BedrockSessionID = event.SessionID
			if event.PromptName != "" {
				r.record.PromptName = event.PromptName
			}
			_ = r.repo.UpdateSessionMetadata(ctx, r.sessionID, event.SessionID, event.PromptName, nil, now)
		}
	case envelope.Event.TextOutput != nil:
		event := envelope.Event.TextOutput
		state := r.appendContentText(event.ContentID, event.Content)
		if state != nil && r.clientAlive.Load() {
			if err := r.writeJSON(map[string]any{
				"type":            wsMessageTranscriptPartial,
				"direction":       state.Direction,
				"text":            state.Text,
				"promptName":      state.PromptName,
				"completionId":    state.CompletionID,
				"contentId":       state.ContentID,
				"generationStage": state.GenerationStage,
			}); err != nil {
				return err
			}
		}
	case envelope.Event.AudioOutput != nil:
		if !r.clientAlive.Load() {
			return nil
		}

		audioBytes, err := base64.StdEncoding.DecodeString(envelope.Event.AudioOutput.Content)
		if err != nil {
			return fmt.Errorf("decode audio output: %w", err)
		}
		if err := r.writeBinary(audioBytes); err != nil {
			return err
		}
	case envelope.Event.ContentEnd != nil:
		event := envelope.Event.ContentEnd
		state := r.finishContent(event.ContentID, event.StopReason)
		if state == nil {
			return nil
		}

		if strings.EqualFold(state.GenerationStage, "FINAL") && strings.TrimSpace(state.Text) != "" {
			sequenceNo := r.nextTurnSequence()
			turn := TranscriptTurn{
				VoiceSessionID:   r.sessionID,
				SequenceNo:       sequenceNo,
				Direction:        state.Direction,
				Modality:         state.Modality,
				TranscriptText:   state.Text,
				BedrockSessionID: state.BedrockSessionID,
				PromptName:       state.PromptName,
				CompletionID:     state.CompletionID,
				ContentID:        state.ContentID,
				GenerationStage:  state.GenerationStage,
				StopReason:       event.StopReason,
				OccurredAt:       state.OccurredAt,
			}
			if err := r.repo.SaveTranscriptTurn(ctx, turn); err != nil {
				return err
			}

			if r.clientAlive.Load() {
				if err := r.writeJSON(map[string]any{
					"type":            wsMessageTranscriptFinal,
					"sequenceNo":      sequenceNo,
					"direction":       turn.Direction,
					"modality":        turn.Modality,
					"text":            turn.TranscriptText,
					"promptName":      turn.PromptName,
					"completionId":    turn.CompletionID,
					"contentId":       turn.ContentID,
					"generationStage": turn.GenerationStage,
					"stopReason":      turn.StopReason,
					"occurredAt":      turn.OccurredAt,
				}); err != nil {
					return err
				}
			}
		}

		if r.clientAlive.Load() && isInterruptionStopReason(event.StopReason) {
			if err := r.writeJSON(map[string]any{
				"type":         wsMessageInterrupted,
				"completionId": state.CompletionID,
				"contentId":    state.ContentID,
				"stopReason":   event.StopReason,
			}); err != nil {
				return err
			}
		}

		r.lastStop = event.StopReason
	case envelope.Event.UsageEvent != nil:
		sequenceNo := r.nextUsageSequence()
		usage := envelope.Event.UsageEvent
		payloadCopy := append(json.RawMessage(nil), payload...)
		record := UsageEvent{
			VoiceSessionID:          r.sessionID,
			SequenceNo:              sequenceNo,
			BedrockSessionID:        usage.SessionID,
			PromptName:              usage.PromptName,
			CompletionID:            usage.CompletionID,
			InputSpeechTokensDelta:  usage.Details.Delta.Input.SpeechTokens,
			InputTextTokensDelta:    usage.Details.Delta.Input.TextTokens,
			OutputSpeechTokensDelta: usage.Details.Delta.Output.SpeechTokens,
			OutputTextTokensDelta:   usage.Details.Delta.Output.TextTokens,
			TotalInputSpeechTokens:  usage.Details.Total.Input.SpeechTokens,
			TotalInputTextTokens:    usage.Details.Total.Input.TextTokens,
			TotalOutputSpeechTokens: usage.Details.Total.Output.SpeechTokens,
			TotalOutputTextTokens:   usage.Details.Total.Output.TextTokens,
			TotalInputTokens:        usage.Details.Total.Input.TotalTokens,
			TotalOutputTokens:       usage.Details.Total.Output.TotalTokens,
			TotalTokens:             usage.Details.Total.TotalTokens,
			Payload:                 payloadCopy,
			EmittedAt:               now,
		}
		if err := r.repo.SaveUsageEvent(ctx, record); err != nil {
			return err
		}

		if r.clientAlive.Load() {
			if err := r.writeJSON(map[string]any{
				"type":         wsMessageUsage,
				"sequenceNo":   sequenceNo,
				"promptName":   record.PromptName,
				"completionId": record.CompletionID,
				"deltas": map[string]any{
					"inputSpeechTokens":  record.InputSpeechTokensDelta,
					"inputTextTokens":    record.InputTextTokensDelta,
					"outputSpeechTokens": record.OutputSpeechTokensDelta,
					"outputTextTokens":   record.OutputTextTokensDelta,
				},
				"totals": map[string]any{
					"inputSpeechTokens":  record.TotalInputSpeechTokens,
					"inputTextTokens":    record.TotalInputTextTokens,
					"outputSpeechTokens": record.TotalOutputSpeechTokens,
					"outputTextTokens":   record.TotalOutputTextTokens,
					"inputTokens":        record.TotalInputTokens,
					"outputTokens":       record.TotalOutputTokens,
					"tokens":             record.TotalTokens,
				},
			}); err != nil {
				return err
			}
		}
	case envelope.Event.CompletionEnd != nil:
		r.lastStop = envelope.Event.CompletionEnd.StopReason
	}

	return nil
}

func (r *runtimeSession) persistClientTextTurn(ctx context.Context, text string) error {
	sequenceNo := r.nextTurnSequence()
	turn := TranscriptTurn{
		VoiceSessionID:  r.sessionID,
		SequenceNo:      sequenceNo,
		Direction:       "user",
		Modality:        "text",
		TranscriptText:  text,
		PromptName:      r.promptName,
		GenerationStage: "FINAL",
		OccurredAt:      r.now(),
	}

	if err := r.repo.SaveTranscriptTurn(ctx, turn); err != nil {
		return err
	}
	return nil
}

func (r *runtimeSession) writeJSON(payload any) error {
	r.sendMu.Lock()
	defer r.sendMu.Unlock()
	return r.conn.WriteJSON(payload)
}

func (r *runtimeSession) writeBinary(payload []byte) error {
	r.sendMu.Lock()
	defer r.sendMu.Unlock()
	return r.conn.WriteMessage(websocket.BinaryMessage, payload)
}

func (r *runtimeSession) appendContentText(contentID, fragment string) *outputContentState {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	state, ok := r.contents[contentID]
	if !ok {
		return nil
	}

	state.Text += fragment
	copyState := *state
	return &copyState
}

func (r *runtimeSession) finishContent(contentID, stopReason string) *outputContentState {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	state, ok := r.contents[contentID]
	if !ok {
		return nil
	}

	copyState := *state
	delete(r.contents, contentID)
	return &copyState
}

func (r *runtimeSession) nextTurnSequence() int {
	r.sequenceMu.Lock()
	defer r.sequenceMu.Unlock()

	r.turnSeq++
	return r.turnSeq
}

func (r *runtimeSession) nextUsageSequence() int {
	r.sequenceMu.Lock()
	defer r.sequenceMu.Unlock()

	r.usageSeq++
	return r.usageSeq
}

func (r *runtimeSession) finalize(ctx context.Context, err error) error {
	status := StatusCompleted
	failureCode := ""
	failureMessage := ""

	switch {
	case err == nil:
	case errors.Is(err, errClientRequestedClose):
		err = nil
	case errors.Is(err, errClientDisconnected):
		status = StatusFailed
		failureCode = "client_disconnected"
		failureMessage = err.Error()
	case r.now().After(r.sessionExpiresAt):
		status = StatusExpired
	default:
		status = StatusFailed
		failureCode = "stream_error"
		failureMessage = err.Error()
	}

	endedAt := r.now()
	stopReason := r.lastStop
	if status == StatusExpired && stopReason == "" {
		stopReason = "MAX_DURATION_REACHED"
	}

	r.record.Status = status
	r.record.StopReason = stopReason
	r.record.FailureCode = failureCode
	r.record.FailureMessage = failureMessage
	r.record.PromptName = r.promptName
	r.record.SessionExpiresAt = &r.sessionExpiresAt
	r.record.EndedAt = &endedAt

	if markErr := r.repo.MarkSessionEnded(ctx, r.sessionID, status, stopReason, failureCode, failureMessage, endedAt); markErr != nil && err == nil {
		err = markErr
	}

	var artifactPaths ArtifactPaths
	if r.artifactExporter != nil {
		transcripts, usageEvents, loadErr := r.loadPersistedArtifacts(ctx)
		if loadErr != nil {
			if err == nil {
				err = loadErr
			}
			if r.clientAlive.Load() {
				_ = r.writeJSON(map[string]any{
					"type":      wsMessageError,
					"code":      "artifact_load_failed",
					"message":   loadErr.Error(),
					"retryable": false,
				})
			}
		}

		exported, exportErr := r.artifactExporter.Export(ctx, SessionArtifact{
			Session:     r.record,
			Status:      status,
			StopReason:  stopReason,
			EndedAt:     endedAt,
			Transcripts: transcripts,
			UsageEvents: usageEvents,
		})
		if exportErr != nil {
			if r.clientAlive.Load() {
				_ = r.writeJSON(map[string]any{
					"type":      wsMessageError,
					"code":      "artifact_export_failed",
					"message":   exportErr.Error(),
					"retryable": false,
				})
			}
		} else {
			artifactPaths = exported
		}
	}

	if r.clientAlive.Load() {
		if status == StatusFailed {
			_ = r.writeJSON(map[string]any{
				"type":      wsMessageError,
				"code":      failureCode,
				"message":   failureMessage,
				"retryable": false,
			})
		}

		_ = r.writeJSON(map[string]any{
			"type":       wsMessageSessionEnded,
			"status":     status,
			"stopReason": stopReason,
			"endedAt":    endedAt,
			"artifacts":  artifactPaths,
		})
	}

	return err
}

func (r *runtimeSession) touchSession(ctx context.Context, now time.Time) {
	r.activityMu.Lock()
	if !r.lastTouchAt.IsZero() && now.Sub(r.lastTouchAt) < sessionTouchTTL {
		r.activityMu.Unlock()
		return
	}
	r.lastTouchAt = now
	r.activityMu.Unlock()

	_ = r.repo.TouchSession(ctx, r.sessionID, now)
}

func (r *runtimeSession) loadPersistedArtifacts(ctx context.Context) ([]TranscriptTurn, []UsageEvent, error) {
	transcripts, err := r.repo.ListTranscriptTurns(ctx, r.sessionID)
	if err != nil {
		return nil, nil, err
	}

	usageEvents, err := r.repo.ListUsageEvents(ctx, r.sessionID)
	if err != nil {
		return nil, nil, err
	}

	return transcripts, usageEvents, nil
}

func (s *Service) resolveVoiceID(ctx context.Context, patientID, requestedVoiceID string) (string, error) {
	if requestedVoiceID != "" {
		if !s.voiceCatalog.IsAllowed(requestedVoiceID) {
			return "", ErrVoiceNotAllowed
		}

		return requestedVoiceID, nil
	}

	if defaultVoiceID, ok, err := s.preferencesStore.GetDefaultVoiceID(ctx, patientID); err != nil {
		return "", err
	} else if ok {
		if !s.voiceCatalog.IsAllowed(defaultVoiceID) {
			return "", ErrVoiceNotAllowed
		}

		return defaultVoiceID, nil
	}

	return s.voiceCatalog.DefaultVoiceID(), nil
}

func newStreamToken() (string, []byte, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", nil, fmt.Errorf("generate stream token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(randomBytes)
	return token, hashToken(token), nil
}

func hashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func combineRuntimeErrors(primary, secondary error) error {
	switch {
	case primary == nil:
		return secondary
	case secondary == nil:
		return primary
	case errors.Is(primary, errClientRequestedClose):
		return secondary
	default:
		return primary
	}
}

func normalizeDirection(role string) string {
	switch strings.ToUpper(role) {
	case "ASSISTANT":
		return "assistant"
	default:
		return "user"
	}
}

func normalizeModality(contentType string) string {
	switch strings.ToUpper(contentType) {
	case "TEXT":
		return "text"
	default:
		return "audio"
	}
}

func parseGenerationStage(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	var fields struct {
		GenerationStage string `json:"generationStage"`
	}
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return ""
	}

	return fields.GenerationStage
}

func isInterruptionStopReason(reason string) bool {
	switch strings.ToUpper(reason) {
	case "INTERRUPTED", "PARTIAL_TURN":
		return true
	default:
		return false
	}
}

type outputEnvelope struct {
	Event outputEvent `json:"event"`
}

type outputEvent struct {
	CompletionStart *completionStartEvent `json:"completionStart,omitempty"`
	ContentStart    *contentStartEvent    `json:"contentStart,omitempty"`
	TextOutput      *textOutputEvent      `json:"textOutput,omitempty"`
	AudioOutput     *audioOutputEvent     `json:"audioOutput,omitempty"`
	ContentEnd      *contentEndEvent      `json:"contentEnd,omitempty"`
	UsageEvent      *usageEventPayload    `json:"usageEvent,omitempty"`
	CompletionEnd   *completionEndEvent   `json:"completionEnd,omitempty"`
}

type completionStartEvent struct {
	SessionID    string `json:"sessionId"`
	PromptName   string `json:"promptName"`
	CompletionID string `json:"completionId"`
}

type contentStartEvent struct {
	SessionID             string `json:"sessionId"`
	PromptName            string `json:"promptName"`
	CompletionID          string `json:"completionId"`
	ContentID             string `json:"contentId"`
	Type                  string `json:"type"`
	Role                  string `json:"role"`
	AdditionalModelFields string `json:"additionalModelFields"`
}

type textOutputEvent struct {
	SessionID    string `json:"sessionId"`
	PromptName   string `json:"promptName"`
	CompletionID string `json:"completionId"`
	ContentID    string `json:"contentId"`
	Content      string `json:"content"`
}

type audioOutputEvent struct {
	SessionID    string `json:"sessionId"`
	PromptName   string `json:"promptName"`
	CompletionID string `json:"completionId"`
	ContentID    string `json:"contentId"`
	Content      string `json:"content"`
}

type contentEndEvent struct {
	SessionID    string `json:"sessionId"`
	PromptName   string `json:"promptName"`
	CompletionID string `json:"completionId"`
	ContentID    string `json:"contentId"`
	StopReason   string `json:"stopReason"`
}

type completionEndEvent struct {
	SessionID    string `json:"sessionId"`
	PromptName   string `json:"promptName"`
	CompletionID string `json:"completionId"`
	StopReason   string `json:"stopReason"`
}

type usageEventPayload struct {
	SessionID    string            `json:"sessionId"`
	PromptName   string            `json:"promptName"`
	CompletionID string            `json:"completionId"`
	Details      usageEventDetails `json:"details"`
}

type usageEventDetails struct {
	Delta usageTokenBreakdown `json:"delta"`
	Total usageTokenBreakdown `json:"total"`
}

type usageTokenBreakdown struct {
	Input       usageTokenCounts `json:"input"`
	Output      usageTokenCounts `json:"output"`
	TotalTokens int              `json:"totalTokens"`
}

type usageTokenCounts struct {
	SpeechTokens int `json:"speechTokens"`
	TextTokens   int `json:"textTokens"`
	TotalTokens  int `json:"totalTokens"`
}
