package voice

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"nova-echoes/api/internal/httpserver/respond"
)

type Handler struct {
	service  *Service
	upgrader websocket.Upgrader
}

func NewHandler(service *Service, allowedOrigins []string) Handler {
	normalizedOrigins := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		normalizedOrigins = append(normalizedOrigins, origin)
	}

	return Handler{
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := strings.TrimSpace(r.Header.Get("Origin"))
				return slices.Contains(normalizedOrigins, "*") || slices.Contains(normalizedOrigins, origin)
			},
		},
	}
}

func (h Handler) ListVoices(w http.ResponseWriter, _ *http.Request) {
	respond.JSON(w, http.StatusOK, h.service.ListVoices(), nil)
}

func (h Handler) ListLabConversations(w http.ResponseWriter, r *http.Request) {
	limit, err := parseConversationLimit(r.URL.Query().Get("limit"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	conversations, err := h.service.ListLabConversations(r.Context(), limit)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "voice_lab_history_error", "Could not load saved voice lab conversations.")
		return
	}

	respond.JSON(w, http.StatusOK, conversations, map[string]int{
		"count": len(conversations),
		"limit": limit,
	})
}

func (h Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var input CreateSessionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		respond.Error(w, http.StatusBadRequest, "invalid_json", "Request body must contain a single JSON object.")
		return
	}

	session, err := h.service.CreateSession(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrPatientIDRequired), errors.Is(err, ErrVoiceNotAllowed), errors.Is(err, ErrSystemPromptTooLarge), errors.Is(err, ErrCallRunNotFound), errors.Is(err, ErrCallRunPatientMismatch), errors.Is(err, ErrCallRunLinkInvalid):
			respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, ErrCallRunAlreadyLinked):
			respond.Error(w, http.StatusConflict, "conflict", err.Error())
		default:
			respond.Error(w, http.StatusInternalServerError, "voice_session_error", "Could not create voice session.")
		}
		return
	}

	respond.JSON(w, http.StatusCreated, session, nil)
}

func (h Handler) Stream(w http.ResponseWriter, r *http.Request) {
	if !h.upgrader.CheckOrigin(r) {
		respond.Error(w, http.StatusForbidden, "origin_not_allowed", "WebSocket origin is not allowed.")
		return
	}

	sessionID := strings.TrimSpace(r.PathValue("id"))
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if sessionID == "" || token == "" {
		respond.Error(w, http.StatusBadRequest, "validation_error", "session id and token are required")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	if err := h.service.Attach(r.Context(), sessionID, token, conn); err != nil {
		closeCode := websocket.ClosePolicyViolation
		message := err.Error()
		if !errors.Is(err, ErrInvalidStreamToken) && !errors.Is(err, ErrTokenExpired) && !errors.Is(err, ErrStreamConsumed) && !errors.Is(err, ErrSessionNotFound) {
			closeCode = websocket.CloseInternalServerErr
			message = "voice session failed"
			if h.service.cfg.AppEnv != "production" {
				message = err.Error()
			}
		}

		log.Printf("voice stream attach failed for session %s: %v", sessionID, err)
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, message), time.Now().Add(time.Second))
		_ = conn.Close()
	}
}
