package admin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"nova-echoes/api/internal/idgen"
)

func (s *PostgresStore) GetCallPromptContext(ctx context.Context, patientID string) (CallPromptContext, error) {
	patient, ok, err := s.GetPatient(ctx, patientID)
	if err != nil {
		return CallPromptContext{}, err
	}
	if !ok {
		return CallPromptContext{}, ErrPatientNotFound
	}

	safePeople, err := s.listPatientPeopleWithCondition(ctx, patient.ID, "safe_to_suggest_call = true")
	if err != nil {
		return CallPromptContext{}, err
	}
	avoidPeople, err := s.listPatientPeopleWithCondition(ctx, patient.ID, "safe_to_suggest_call = false")
	if err != nil {
		return CallPromptContext{}, err
	}
	recentEntries, err := s.listMemoryBankEntries(ctx, patient.ID, 5)
	if err != nil {
		return CallPromptContext{}, err
	}

	return CallPromptContext{
		Patient:                 patient,
		SafePeopleForCallAnchor: safePeople,
		PeopleToAvoidNaming:     avoidPeople,
		RecentMemoryBankEntries: recentEntries,
	}, nil
}

func (s *PostgresStore) ListPatientPeople(ctx context.Context, patientID string) ([]PatientPerson, error) {
	return s.listPatientPeopleWithCondition(ctx, patientID, "1 = 1")
}

func (s *PostgresStore) UpdatePatientPerson(ctx context.Context, patientID, personID string, input UpdatePatientPersonRequest) (PatientPerson, error) {
	row := s.db.QueryRowContext(ctx, `
		update patient_people
		set name = $3,
		    relationship = nullif($4, ''),
		    status = $5,
		    relationship_quality = $6,
		    notes = nullif($7, ''),
		    updated_at = now()
		where patient_id = $1
		  and id = $2
		returning
			id,
			patient_id,
			name,
			coalesce(relationship, ''),
			status,
			relationship_quality,
			safe_to_suggest_call,
			first_mentioned_at,
			coalesce(first_mentioned_call_run_id, ''),
			last_mentioned_at,
			coalesce(last_mentioned_call_run_id, ''),
			coalesce(context, ''),
			coalesce(notes, ''),
			created_at,
			updated_at
	`, strings.TrimSpace(patientID), strings.TrimSpace(personID), strings.TrimSpace(input.Name), strings.TrimSpace(input.Relationship), strings.TrimSpace(input.Status), strings.TrimSpace(input.RelationshipQuality), strings.TrimSpace(input.Notes))

	person, err := scanPatientPerson(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return PatientPerson{}, ErrPatientPersonNotFound
		}
		return PatientPerson{}, fmt.Errorf("update patient person: %w", err)
	}

	return person, nil
}

func (s *PostgresStore) ListMemoryBankEntries(ctx context.Context, patientID string) ([]MemoryBankEntry, error) {
	return s.listMemoryBankEntries(ctx, patientID, 0)
}

func (s *PostgresStore) ListPatientReminders(ctx context.Context, patientID string) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx, `
		select
			pr.id,
			pr.patient_id,
			coalesce(pr.source_call_run_id, ''),
			coalesce(pr.source_analysis_result_id, ''),
			pr.kind,
			pr.status,
			pr.title,
			coalesce(pr.detail, ''),
			coalesce(pr.person_id, ''),
			pr.caregiver_follow_up_recommended,
			pr.suggested_for,
			pr.created_by,
			pr.created_at,
			pr.updated_at,
			pp.id,
			pp.patient_id,
			pp.name,
			coalesce(pp.relationship, ''),
			coalesce(pp.status, ''),
			coalesce(pp.relationship_quality, ''),
			coalesce(pp.safe_to_suggest_call, false),
			pp.first_mentioned_at,
			coalesce(pp.first_mentioned_call_run_id, ''),
			pp.last_mentioned_at,
			coalesce(pp.last_mentioned_call_run_id, ''),
			coalesce(pp.context, ''),
			coalesce(pp.notes, ''),
			pp.created_at,
			pp.updated_at
		from patient_reminders pr
		left join patient_people pp on pp.id = pr.person_id
		where pr.patient_id = $1
		order by pr.created_at desc, pr.id desc
	`, strings.TrimSpace(patientID))
	if err != nil {
		return nil, fmt.Errorf("list patient reminders: %w", err)
	}
	defer rows.Close()

	reminders := make([]Reminder, 0)
	for rows.Next() {
		reminder, scanErr := scanReminder(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan patient reminder: %w", scanErr)
		}
		reminders = append(reminders, reminder)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patient reminders: %w", err)
	}

	return reminders, nil
}

func (s *PostgresStore) listPatientPeopleWithCondition(ctx context.Context, patientID string, condition string) ([]PatientPerson, error) {
	query := `
		select
			id,
			patient_id,
			name,
			coalesce(relationship, ''),
			status,
			relationship_quality,
			safe_to_suggest_call,
			first_mentioned_at,
			coalesce(first_mentioned_call_run_id, ''),
			last_mentioned_at,
			coalesce(last_mentioned_call_run_id, ''),
			coalesce(context, ''),
			coalesce(notes, ''),
			created_at,
			updated_at
		from patient_people
		where patient_id = $1
		  and ` + condition + `
		order by last_mentioned_at desc, created_at desc, id desc
	`

	rows, err := s.db.QueryContext(ctx, query, strings.TrimSpace(patientID))
	if err != nil {
		return nil, fmt.Errorf("list patient people: %w", err)
	}
	defer rows.Close()

	people := make([]PatientPerson, 0)
	for rows.Next() {
		person, scanErr := scanPatientPerson(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan patient person: %w", scanErr)
		}
		people = append(people, person)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patient people: %w", err)
	}

	return people, nil
}

func (s *PostgresStore) listMemoryBankEntries(ctx context.Context, patientID string, limit int) ([]MemoryBankEntry, error) {
	query := `
		select
			id,
			patient_id,
			source_call_run_id,
			source_analysis_result_id,
			topic,
			summary,
			coalesce(emotional_tone, ''),
			responded_well_to,
			anchor_offered,
			anchor_type,
			anchor_accepted,
			coalesce(anchor_detail, ''),
			coalesce(suggested_followup, ''),
			occurred_at,
			created_at,
			updated_at
		from memory_bank_entries
		where patient_id = $1
		order by occurred_at desc, created_at desc, id desc
	`
	args := []any{strings.TrimSpace(patientID)}
	if limit > 0 {
		query += ` limit $2`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list memory bank entries: %w", err)
	}
	defer rows.Close()

	entryOrder := make([]string, 0)
	entryMap := make(map[string]*MemoryBankEntry)
	for rows.Next() {
		entry, scanErr := scanMemoryBankEntry(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan memory bank entry: %w", scanErr)
		}
		entry.People = []PatientPerson{}
		entryCopy := entry
		entryOrder = append(entryOrder, entry.ID)
		entryMap[entry.ID] = &entryCopy
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory bank entries: %w", err)
	}
	if len(entryOrder) == 0 {
		return []MemoryBankEntry{}, nil
	}

	peopleQuery, peopleArgs := buildINQuery(`
		select
			mep.memory_bank_entry_id,
			pp.id,
			pp.patient_id,
			pp.name,
			coalesce(pp.relationship, ''),
			pp.status,
			pp.relationship_quality,
			pp.safe_to_suggest_call,
			pp.first_mentioned_at,
			coalesce(pp.first_mentioned_call_run_id, ''),
			pp.last_mentioned_at,
			coalesce(pp.last_mentioned_call_run_id, ''),
			coalesce(pp.context, ''),
			coalesce(pp.notes, ''),
			pp.created_at,
			pp.updated_at
		from memory_bank_entry_people mep
		join patient_people pp on pp.id = mep.patient_person_id
		where mep.memory_bank_entry_id in (`, `)
		order by pp.last_mentioned_at desc, pp.created_at desc, pp.id desc
	`, entryOrder)
	peopleRows, err := s.db.QueryContext(ctx, peopleQuery, peopleArgs...)
	if err != nil {
		return nil, fmt.Errorf("list memory bank entry people: %w", err)
	}
	defer peopleRows.Close()

	for peopleRows.Next() {
		var entryID string
		person, scanErr := scanPatientPersonWithPrefix(peopleRows, &entryID)
		if scanErr != nil {
			return nil, fmt.Errorf("scan memory bank entry person: %w", scanErr)
		}
		entry := entryMap[entryID]
		if entry == nil {
			continue
		}
		entry.People = append(entry.People, person)
	}
	if err := peopleRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory bank entry people: %w", err)
	}

	entries := make([]MemoryBankEntry, 0, len(entryOrder))
	for _, entryID := range entryOrder {
		entry := entryMap[entryID]
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries, nil
}

func (s *PostgresStore) materializeAnalysisSideEffectsTx(ctx context.Context, tx *sql.Tx, input SaveAnalysisResultInput, analysisID string) error {
	switch input.CallType {
	case CallTypeCheckIn:
		if input.Result.CheckIn == nil {
			return nil
		}
		return s.materializeCheckInTx(ctx, tx, input, analysisID)
	case CallTypeReminiscence:
		if input.Result.Reminiscence == nil {
			return nil
		}
		return s.materializeReminiscenceTx(ctx, tx, input, analysisID)
	default:
		return nil
	}
}

func (s *PostgresStore) materializeCheckInTx(ctx context.Context, tx *sql.Tx, input SaveAnalysisResultInput, analysisID string) error {
	checkIn := input.Result.CheckIn
	people, err := s.upsertMentionedPeopleTx(ctx, tx, input.PatientID, input.CallRunID, input.GeneratedAt, checkIn.MentionedPeople)
	if err != nil {
		return err
	}

	for _, reminder := range checkIn.RemindersNoted {
		title := strings.TrimSpace(reminder.Title)
		detail := strings.TrimSpace(reminder.Detail)
		if title == "" && detail == "" {
			continue
		}

		kind := ReminderKindGeneral
		if containsAnySubstring(title+" "+detail, "appointment", "doctor", "visit") {
			kind = ReminderKindAppointment
		}
		if _, err := insertPatientReminderTx(ctx, tx, createReminderParams{
			PatientID:                    input.PatientID,
			SourceCallRunID:              input.CallRunID,
			SourceAnalysisResultID:       analysisID,
			Kind:                         kind,
			Status:                       ReminderStatusPending,
			Title:                        chooseString(title, "Reminder"),
			Detail:                       detail,
			PersonID:                     matchMentionedPersonInText(title+" "+detail, people),
			CaregiverFollowUpRecommended: false,
			CreatedBy:                    ReminderCreatedByAnalysisWorker,
			CreatedAt:                    input.GeneratedAt,
		}); err != nil {
			return err
		}
	}

	if checkIn.ReminderDeclined {
		title := chooseString(strings.TrimSpace(checkIn.ReminderDeclinedTopic), "Declined reminder")
		if _, err := insertPatientReminderTx(ctx, tx, createReminderParams{
			PatientID:                    input.PatientID,
			SourceCallRunID:              input.CallRunID,
			SourceAnalysisResultID:       analysisID,
			Kind:                         ReminderKindGeneral,
			Status:                       ReminderStatusDeclined,
			Title:                        title,
			Detail:                       "Declined during check-in. Caregiver follow-up recommended.",
			PersonID:                     matchMentionedPersonInText(title, people),
			CaregiverFollowUpRecommended: true,
			CreatedBy:                    ReminderCreatedByAnalysisWorker,
			CreatedAt:                    input.GeneratedAt,
		}); err != nil {
			return err
		}
	}

	if err := s.updateRunningCheckInProfileTx(ctx, tx, input.PatientID, *checkIn, input.Result.NextCallRecommendation); err != nil {
		return err
	}

	if err := s.insertCheckInMemoryBankEntryTx(ctx, tx, input, analysisID, people); err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) materializeReminiscenceTx(ctx context.Context, tx *sql.Tx, input SaveAnalysisResultInput, analysisID string) error {
	reminiscence := input.Result.Reminiscence
	people, err := s.upsertMentionedPeopleTx(ctx, tx, input.PatientID, input.CallRunID, input.GeneratedAt, reminiscence.MentionedPeople)
	if err != nil {
		return err
	}

	entryID, err := idgen.New()
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		insert into memory_bank_entries (
			id,
			patient_id,
			source_call_run_id,
			source_analysis_result_id,
			topic,
			summary,
			emotional_tone,
			responded_well_to,
			anchor_offered,
			anchor_type,
			anchor_accepted,
			anchor_detail,
			suggested_followup,
			occurred_at,
			updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
	`, entryID, input.PatientID, input.CallRunID, analysisID, chooseString(strings.TrimSpace(reminiscence.Topic), "Untitled memory"), chooseString(strings.TrimSpace(reminiscence.Summary), strings.TrimSpace(input.Result.Summary)), nullableString(reminiscence.EmotionalTone), marshalStringList(reminiscence.RespondedWellTo), reminiscence.AnchorOffered, chooseString(strings.TrimSpace(reminiscence.AnchorType), AnchorTypeNone), reminiscence.AnchorAccepted, nullableString(reminiscence.AnchorDetail), nullableString(reminiscence.SuggestedFollowUp), input.GeneratedAt); err != nil {
		return fmt.Errorf("insert memory bank entry: %w", err)
	}

	for _, person := range people {
		if strings.TrimSpace(person.ID) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			insert into memory_bank_entry_people (
				memory_bank_entry_id,
				patient_person_id
			) values ($1, $2)
			on conflict (memory_bank_entry_id, patient_person_id) do nothing
		`, entryID, person.ID); err != nil {
			return fmt.Errorf("attach memory bank person: %w", err)
		}
	}

	if err := s.updateRunningMemoryProfileTx(ctx, tx, input.PatientID, *reminiscence); err != nil {
		return err
	}

	if reminiscence.AnchorAccepted {
		kind := anchorTypeToReminderKind(reminiscence.AnchorType)
		personID := ""
		if kind == ReminderKindCallPerson {
			personID, err = s.matchSafePersonForAnchorTx(ctx, tx, input.PatientID, reminiscence.AnchorDetail)
			if err != nil {
				return err
			}
			if personID == "" {
				kind = ReminderKindGeneral
			}
		}
		title := reminderTitleForAnchor(*reminiscence, people, personID)
		if _, err := insertPatientReminderTx(ctx, tx, createReminderParams{
			PatientID:                    input.PatientID,
			SourceCallRunID:              input.CallRunID,
			SourceAnalysisResultID:       analysisID,
			Kind:                         kind,
			Status:                       ReminderStatusPending,
			Title:                        title,
			Detail:                       strings.TrimSpace(reminiscence.AnchorDetail),
			PersonID:                     personID,
			CaregiverFollowUpRecommended: false,
			CreatedBy:                    ReminderCreatedByAnalysisWorker,
			CreatedAt:                    input.GeneratedAt,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *PostgresStore) upsertMentionedPeopleTx(ctx context.Context, tx *sql.Tx, patientID, callRunID string, now time.Time, mentioned []MentionedPerson) ([]PatientPerson, error) {
	people := make([]PatientPerson, 0, len(mentioned))
	for _, mentionedPerson := range mentioned {
		normalizedName := normalizePersonName(mentionedPerson.Name)
		if normalizedName == "" {
			continue
		}

		existing, err := s.findPatientPeopleByNormalizedNameTx(ctx, tx, patientID, normalizedName)
		if err != nil {
			return nil, err
		}
		if len(existing) == 1 {
			person, err := s.touchPatientPersonTx(ctx, tx, existing[0], callRunID, now, mentionedPerson)
			if err != nil {
				return nil, err
			}
			people = append(people, person)
			continue
		}

		person, err := s.insertPatientPersonTx(ctx, tx, patientID, callRunID, now, mentionedPerson)
		if err != nil {
			return nil, err
		}
		people = append(people, person)
	}
	return people, nil
}

func (s *PostgresStore) findPatientPeopleByNormalizedNameTx(ctx context.Context, tx *sql.Tx, patientID, normalizedName string) ([]PatientPerson, error) {
	rows, err := tx.QueryContext(ctx, `
		select
			id,
			patient_id,
			name,
			coalesce(relationship, ''),
			status,
			relationship_quality,
			safe_to_suggest_call,
			first_mentioned_at,
			coalesce(first_mentioned_call_run_id, ''),
			last_mentioned_at,
			coalesce(last_mentioned_call_run_id, ''),
			coalesce(context, ''),
			coalesce(notes, ''),
			created_at,
			updated_at
		from patient_people
		where patient_id = $1
		  and regexp_replace(lower(trim(name)), '\s+', ' ', 'g') = $2
		order by created_at asc, id asc
	`, strings.TrimSpace(patientID), normalizedName)
	if err != nil {
		return nil, fmt.Errorf("query matching patient people: %w", err)
	}
	defer rows.Close()

	people := make([]PatientPerson, 0)
	for rows.Next() {
		person, scanErr := scanPatientPerson(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan matching patient person: %w", scanErr)
		}
		people = append(people, person)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate matching patient people: %w", err)
	}

	return people, nil
}

func (s *PostgresStore) touchPatientPersonTx(ctx context.Context, tx *sql.Tx, current PatientPerson, callRunID string, now time.Time, mentioned MentionedPerson) (PatientPerson, error) {
	row := tx.QueryRowContext(ctx, `
		update patient_people
		set relationship = case
				when coalesce(relationship, '') = '' and nullif($3, '') is not null then nullif($3, '')
				else relationship
			end,
		    context = case
				when coalesce(context, '') = '' and nullif($4, '') is not null then nullif($4, '')
				else context
			end,
		    last_mentioned_at = $5,
		    last_mentioned_call_run_id = nullif($6, ''),
		    updated_at = $5
		where id = $1
		  and patient_id = $2
		returning
			id,
			patient_id,
			name,
			coalesce(relationship, ''),
			status,
			relationship_quality,
			safe_to_suggest_call,
			first_mentioned_at,
			coalesce(first_mentioned_call_run_id, ''),
			last_mentioned_at,
			coalesce(last_mentioned_call_run_id, ''),
			coalesce(context, ''),
			coalesce(notes, ''),
			created_at,
			updated_at
	`, current.ID, current.PatientID, strings.TrimSpace(mentioned.Relationship), strings.TrimSpace(mentioned.Context), now, strings.TrimSpace(callRunID))

	person, err := scanPatientPerson(row)
	if err != nil {
		return PatientPerson{}, fmt.Errorf("update patient person mention: %w", err)
	}
	return person, nil
}

func (s *PostgresStore) insertPatientPersonTx(ctx context.Context, tx *sql.Tx, patientID, callRunID string, now time.Time, mentioned MentionedPerson) (PatientPerson, error) {
	personID, err := idgen.New()
	if err != nil {
		return PatientPerson{}, err
	}

	row := tx.QueryRowContext(ctx, `
		insert into patient_people (
			id,
			patient_id,
			name,
			relationship,
			status,
			relationship_quality,
			first_mentioned_at,
			first_mentioned_call_run_id,
			last_mentioned_at,
			last_mentioned_call_run_id,
			context,
			updated_at
		) values ($1, $2, $3, nullif($4, ''), $5, $6, $7, nullif($8, ''), $7, nullif($8, ''), nullif($9, ''), $7)
		returning
			id,
			patient_id,
			name,
			coalesce(relationship, ''),
			status,
			relationship_quality,
			safe_to_suggest_call,
			first_mentioned_at,
			coalesce(first_mentioned_call_run_id, ''),
			last_mentioned_at,
			coalesce(last_mentioned_call_run_id, ''),
			coalesce(context, ''),
			coalesce(notes, ''),
			created_at,
			updated_at
	`, personID, strings.TrimSpace(patientID), strings.TrimSpace(mentioned.Name), strings.TrimSpace(mentioned.Relationship), PersonStatusUnknown, RelationshipQualityUnknown, now, strings.TrimSpace(callRunID), strings.TrimSpace(mentioned.Context))

	person, err := scanPatientPerson(row)
	if err != nil {
		return PatientPerson{}, fmt.Errorf("insert patient person: %w", err)
	}
	return person, nil
}

func (s *PostgresStore) updateRunningMemoryProfileTx(ctx context.Context, tx *sql.Tx, patientID string, reminiscence ReminiscenceAnalysis) error {
	patient, ok, err := s.getPatientTx(ctx, tx, patientID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPatientNotFound
	}

	topicsToRevisit := patient.MemoryProfile.TopicsToRevisit
	if followUp := strings.TrimSpace(reminiscence.SuggestedFollowUp); followUp != "" {
		topicsToRevisit = append(topicsToRevisit, followUp)
	}

	if _, err := tx.ExecContext(ctx, `
		update patient_memory_profiles
		set likes = $2,
		    significant_places = $3,
		    life_chapters = $4,
		    favorite_music = $5,
		    favorite_shows_films = $6,
		    topics_to_revisit = $7,
		    updated_at = now()
		where patient_id = $1
	`, strings.TrimSpace(patientID), marshalStringList(mergeStringLists(patient.MemoryProfile.Likes, reminiscence.RespondedWellTo)), marshalStringList(mergeStringLists(patient.MemoryProfile.SignificantPlaces, reminiscence.MentionedPlaces)), marshalStringList(mergeStringLists(patient.MemoryProfile.LifeChapters, reminiscence.LifeChapters)), marshalStringList(mergeStringLists(patient.MemoryProfile.FavoriteMusic, reminiscence.MentionedMusic)), marshalStringList(mergeStringLists(patient.MemoryProfile.FavoriteShowsFilms, reminiscence.MentionedShowsFilms)), marshalStringList(mergeStringLists(patient.MemoryProfile.TopicsToRevisit, topicsToRevisit))); err != nil {
		return fmt.Errorf("update running memory profile: %w", err)
	}

	return nil
}

func (s *PostgresStore) updateRunningCheckInProfileTx(ctx context.Context, tx *sql.Tx, patientID string, checkIn CheckInAnalysis, nextCall *NextCallRecommendation) error {
	patient, ok, err := s.getPatientTx(ctx, tx, patientID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPatientNotFound
	}

	familyMembers := mergeFamilyMembers(patient.MemoryProfile.FamilyMembers, checkIn.MentionedPeople)
	topicsToRevisit := append([]string{}, patient.MemoryProfile.TopicsToRevisit...)
	for _, reminder := range checkIn.RemindersNoted {
		topic := chooseString(strings.TrimSpace(reminder.Title), strings.TrimSpace(reminder.Detail))
		if topic != "" {
			topicsToRevisit = append(topicsToRevisit, topic)
		}
	}
	if nextCall != nil && strings.TrimSpace(nextCall.Goal) != "" {
		topicsToRevisit = append(topicsToRevisit, strings.TrimSpace(nextCall.Goal))
	}

	if _, err := tx.ExecContext(ctx, `
		update patient_memory_profiles
		set family_members = $2,
		    topics_to_revisit = $3,
		    updated_at = now()
		where patient_id = $1
	`, strings.TrimSpace(patientID), marshalJSON(familyMembers), marshalStringList(mergeStringLists(patient.MemoryProfile.TopicsToRevisit, topicsToRevisit))); err != nil {
		return fmt.Errorf("update running check-in profile: %w", err)
	}

	return nil
}

func (s *PostgresStore) insertCheckInMemoryBankEntryTx(ctx context.Context, tx *sql.Tx, input SaveAnalysisResultInput, analysisID string, people []PatientPerson) error {
	checkIn := input.Result.CheckIn
	topic := deriveCheckInMemoryTopic(*checkIn, input.Result.NextCallRecommendation, people)
	summary := chooseString(strings.TrimSpace(checkIn.CaregiverSummary), strings.TrimSpace(input.Result.Summary))
	if topic == "" || summary == "" {
		return nil
	}

	entryID, err := idgen.New()
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into memory_bank_entries (
			id,
			patient_id,
			source_call_run_id,
			source_analysis_result_id,
			topic,
			summary,
			emotional_tone,
			responded_well_to,
			anchor_offered,
			anchor_type,
			anchor_accepted,
			anchor_detail,
			suggested_followup,
			occurred_at,
			updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, false, $9, false, null, $10, $11, $11)
	`, entryID, input.PatientID, input.CallRunID, analysisID, topic, summary, nullableString(checkIn.Mood), marshalStringList(deriveCheckInRespondedWellTo(*checkIn, people)), AnchorTypeNone, nullableString(deriveCheckInSuggestedFollowUp(*checkIn, input.Result.NextCallRecommendation)), input.GeneratedAt); err != nil {
		return fmt.Errorf("insert check-in memory bank entry: %w", err)
	}

	for _, person := range people {
		if strings.TrimSpace(person.ID) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			insert into memory_bank_entry_people (
				memory_bank_entry_id,
				patient_person_id
			) values ($1, $2)
			on conflict (memory_bank_entry_id, patient_person_id) do nothing
		`, entryID, person.ID); err != nil {
			return fmt.Errorf("attach check-in memory bank person: %w", err)
		}
	}

	return nil
}

func (s *PostgresStore) matchSafePersonForAnchorTx(ctx context.Context, tx *sql.Tx, patientID, anchorDetail string) (string, error) {
	detail := strings.ToLower(strings.TrimSpace(anchorDetail))
	if detail == "" {
		return "", nil
	}

	rows, err := tx.QueryContext(ctx, `
		select id, name
		from patient_people
		where patient_id = $1
		  and safe_to_suggest_call = true
		order by last_mentioned_at desc, created_at desc
	`, strings.TrimSpace(patientID))
	if err != nil {
		return "", fmt.Errorf("query safe patient people: %w", err)
	}
	defer rows.Close()

	matches := make([]string, 0, 1)
	for rows.Next() {
		var personID string
		var name string
		if err := rows.Scan(&personID, &name); err != nil {
			return "", fmt.Errorf("scan safe patient person: %w", err)
		}
		if name == "" {
			continue
		}
		if strings.Contains(detail, strings.ToLower(strings.TrimSpace(name))) {
			matches = append(matches, personID)
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate safe patient people: %w", err)
	}
	if len(matches) != 1 {
		return "", nil
	}
	return matches[0], nil
}

func matchMentionedPersonInText(text string, people []PatientPerson) string {
	normalizedText := strings.ToLower(strings.TrimSpace(text))
	if normalizedText == "" {
		return ""
	}

	matches := make([]string, 0, 1)
	for _, person := range people {
		name := strings.ToLower(strings.TrimSpace(person.Name))
		if name == "" {
			continue
		}
		if strings.Contains(normalizedText, name) {
			matches = append(matches, person.ID)
		}
	}
	if len(matches) != 1 {
		return ""
	}
	return matches[0]
}

func insertPatientReminderTx(ctx context.Context, tx *sql.Tx, params createReminderParams) (string, error) {
	reminderID, err := idgen.New()
	if err != nil {
		return "", err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into patient_reminders (
			id,
			patient_id,
			source_call_run_id,
			source_analysis_result_id,
			kind,
			status,
			title,
			detail,
			person_id,
			caregiver_follow_up_recommended,
			suggested_for,
			created_by,
			created_at,
			updated_at
		) values ($1, $2, nullif($3, ''), nullif($4, ''), $5, $6, $7, nullif($8, ''), nullif($9, ''), $10, $11, $12, $13, $13)
	`, reminderID, strings.TrimSpace(params.PatientID), strings.TrimSpace(params.SourceCallRunID), strings.TrimSpace(params.SourceAnalysisResultID), params.Kind, params.Status, strings.TrimSpace(params.Title), strings.TrimSpace(params.Detail), strings.TrimSpace(params.PersonID), params.CaregiverFollowUpRecommended, params.SuggestedFor, params.CreatedBy, params.CreatedAt); err != nil {
		return "", fmt.Errorf("insert patient reminder: %w", err)
	}

	return reminderID, nil
}

func scanPatientPerson(row scanner) (PatientPerson, error) {
	var person PatientPerson
	if err := row.Scan(&person.ID, &person.PatientID, &person.Name, &person.Relationship, &person.Status, &person.RelationshipQuality, &person.SafeToSuggestCall, &person.FirstMentionedAt, &person.FirstMentionedCallRunID, &person.LastMentionedAt, &person.LastMentionedCallRunID, &person.Context, &person.Notes, &person.CreatedAt, &person.UpdatedAt); err != nil {
		return PatientPerson{}, err
	}
	return person, nil
}

func scanPatientPersonWithPrefix(row scanner, entryID *string) (PatientPerson, error) {
	var person PatientPerson
	if err := row.Scan(entryID, &person.ID, &person.PatientID, &person.Name, &person.Relationship, &person.Status, &person.RelationshipQuality, &person.SafeToSuggestCall, &person.FirstMentionedAt, &person.FirstMentionedCallRunID, &person.LastMentionedAt, &person.LastMentionedCallRunID, &person.Context, &person.Notes, &person.CreatedAt, &person.UpdatedAt); err != nil {
		return PatientPerson{}, err
	}
	return person, nil
}

func scanMemoryBankEntry(row scanner) (MemoryBankEntry, error) {
	var (
		entry           MemoryBankEntry
		respondedWellTo []byte
	)
	if err := row.Scan(&entry.ID, &entry.PatientID, &entry.SourceCallRunID, &entry.SourceAnalysisResultID, &entry.Topic, &entry.Summary, &entry.EmotionalTone, &respondedWellTo, &entry.AnchorOffered, &entry.AnchorType, &entry.AnchorAccepted, &entry.AnchorDetail, &entry.SuggestedFollowUp, &entry.OccurredAt, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
		return MemoryBankEntry{}, err
	}
	entry.RespondedWellTo = parseStringList(respondedWellTo)
	return entry, nil
}

func scanReminder(row scanner) (Reminder, error) {
	var (
		reminder                      Reminder
		suggestedFor                  sql.NullTime
		personID                      sql.NullString
		personRowID                   sql.NullString
		person                        PatientPerson
		personPatientID               sql.NullString
		personName                    sql.NullString
		personRelationship            sql.NullString
		personStatus                  sql.NullString
		personRelationshipQuality     sql.NullString
		personFirstMentionedAt        sql.NullTime
		personFirstMentionedCallRunID sql.NullString
		personLastMentionedAt         sql.NullTime
		personLastMentionedCallRunID  sql.NullString
		personContext                 sql.NullString
		personNotes                   sql.NullString
		personCreatedAt               sql.NullTime
		personUpdatedAt               sql.NullTime
	)
	if err := row.Scan(&reminder.ID, &reminder.PatientID, &reminder.SourceCallRunID, &reminder.SourceAnalysisResultID, &reminder.Kind, &reminder.Status, &reminder.Title, &reminder.Detail, &personID, &reminder.CaregiverFollowUpRecommended, &suggestedFor, &reminder.CreatedBy, &reminder.CreatedAt, &reminder.UpdatedAt, &personRowID, &personPatientID, &personName, &personRelationship, &personStatus, &personRelationshipQuality, &person.SafeToSuggestCall, &personFirstMentionedAt, &personFirstMentionedCallRunID, &personLastMentionedAt, &personLastMentionedCallRunID, &personContext, &personNotes, &personCreatedAt, &personUpdatedAt); err != nil {
		return Reminder{}, err
	}
	if personID.Valid {
		reminder.PersonID = personID.String
	}
	if suggestedFor.Valid {
		reminder.SuggestedFor = &suggestedFor.Time
	}
	if personRowID.Valid {
		person.ID = personRowID.String
		person.PatientID = personPatientID.String
		person.Name = personName.String
		person.Relationship = personRelationship.String
		person.Status = personStatus.String
		person.RelationshipQuality = personRelationshipQuality.String
		if personFirstMentionedAt.Valid {
			person.FirstMentionedAt = personFirstMentionedAt.Time
		}
		person.FirstMentionedCallRunID = personFirstMentionedCallRunID.String
		if personLastMentionedAt.Valid {
			person.LastMentionedAt = personLastMentionedAt.Time
		}
		person.LastMentionedCallRunID = personLastMentionedCallRunID.String
		person.Context = personContext.String
		person.Notes = personNotes.String
		if personCreatedAt.Valid {
			person.CreatedAt = personCreatedAt.Time
		}
		if personUpdatedAt.Valid {
			person.UpdatedAt = personUpdatedAt.Time
		}
		reminder.Person = &person
	}
	return reminder, nil
}

func normalizePersonName(name string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(name)))
	return strings.Join(fields, " ")
}

func mergeStringLists(existing []string, incoming []string) []string {
	seen := make(map[string]struct{})
	merged := make([]string, 0, len(existing)+len(incoming))
	for _, candidate := range append(append([]string{}, existing...), incoming...) {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, trimmed)
	}
	return merged
}

func mergeFamilyMembers(existing []FamilyMember, mentioned []MentionedPerson) []FamilyMember {
	seen := make(map[string]struct{}, len(existing))
	merged := make([]FamilyMember, 0, len(existing)+len(mentioned))
	for _, member := range existing {
		normalizedName := normalizePersonName(member.Name)
		if normalizedName == "" {
			continue
		}
		seen[normalizedName] = struct{}{}
		merged = append(merged, member)
	}

	for _, person := range mentioned {
		normalizedName := normalizePersonName(person.Name)
		if normalizedName == "" || !isLikelyPersonalRelationship(person.Relationship) {
			continue
		}
		if _, ok := seen[normalizedName]; ok {
			continue
		}
		seen[normalizedName] = struct{}{}
		merged = append(merged, FamilyMember{
			Name:     strings.TrimSpace(person.Name),
			Relation: strings.TrimSpace(person.Relationship),
			Notes:    strings.TrimSpace(person.Context),
		})
	}

	return merged
}

func isLikelyPersonalRelationship(relationship string) bool {
	normalized := strings.ToLower(strings.TrimSpace(relationship))
	if normalized == "" {
		return false
	}
	for _, blocked := range []string{"historical figure", "politician", "celebrity", "fictional"} {
		if strings.Contains(normalized, blocked) {
			return false
		}
	}
	return true
}

func deriveCheckInMemoryTopic(checkIn CheckInAnalysis, nextCall *NextCallRecommendation, people []PatientPerson) string {
	if len(people) > 0 {
		for _, reminder := range checkIn.RemindersNoted {
			detail := strings.ToLower(strings.TrimSpace(reminder.Title + " " + reminder.Detail))
			if strings.Contains(detail, strings.ToLower(strings.TrimSpace(people[0].Name))) {
				return fmt.Sprintf("Reconnecting with %s", strings.TrimSpace(people[0].Name))
			}
		}
		return fmt.Sprintf("Conversation about %s", strings.TrimSpace(people[0].Name))
	}
	if len(checkIn.RemindersNoted) > 0 {
		return chooseString(strings.TrimSpace(checkIn.RemindersNoted[0].Title), "Check-in follow-up")
	}
	if nextCall != nil && strings.TrimSpace(nextCall.Goal) != "" {
		return "Check-in follow-up"
	}
	return ""
}

func deriveCheckInSuggestedFollowUp(checkIn CheckInAnalysis, nextCall *NextCallRecommendation) string {
	if nextCall != nil && strings.TrimSpace(nextCall.Goal) != "" {
		return strings.TrimSpace(nextCall.Goal)
	}
	if len(checkIn.RemindersNoted) > 0 {
		return chooseString(strings.TrimSpace(checkIn.RemindersNoted[0].Detail), strings.TrimSpace(checkIn.RemindersNoted[0].Title))
	}
	return ""
}

func deriveCheckInRespondedWellTo(checkIn CheckInAnalysis, people []PatientPerson) []string {
	values := make([]string, 0, len(people)+len(checkIn.RemindersNoted))
	for _, person := range people {
		if strings.TrimSpace(person.Name) != "" {
			values = append(values, strings.TrimSpace(person.Name))
		}
	}
	for _, reminder := range checkIn.RemindersNoted {
		topic := chooseString(strings.TrimSpace(reminder.Title), strings.TrimSpace(reminder.Detail))
		if topic != "" {
			values = append(values, topic)
		}
	}
	return mergeStringLists(nil, values)
}

func anchorTypeToReminderKind(anchorType string) string {
	switch strings.TrimSpace(anchorType) {
	case AnchorTypeCall:
		return ReminderKindCallPerson
	case AnchorTypeMusic:
		return ReminderKindMusic
	case AnchorTypeShowFilm:
		return ReminderKindShowFilm
	case AnchorTypeJournal:
		return ReminderKindJournal
	default:
		return ReminderKindGeneral
	}
}

func reminderTitleForAnchor(reminiscence ReminiscenceAnalysis, people []PatientPerson, matchedPersonID string) string {
	switch anchorTypeToReminderKind(reminiscence.AnchorType) {
	case ReminderKindCallPerson:
		if matchedPersonID != "" {
			for _, person := range people {
				if person.ID == matchedPersonID && strings.TrimSpace(person.Name) != "" {
					return "Call " + strings.TrimSpace(person.Name)
				}
			}
		}
		return chooseString(strings.TrimSpace(reminiscence.AnchorDetail), "Follow up on shared memory")
	case ReminderKindMusic:
		return chooseString(strings.TrimSpace(reminiscence.AnchorDetail), "Listen to a favourite song")
	case ReminderKindShowFilm:
		return chooseString(strings.TrimSpace(reminiscence.AnchorDetail), "Watch a favourite show or film")
	case ReminderKindJournal:
		return chooseString(strings.TrimSpace(reminiscence.AnchorDetail), "Write down a memory")
	default:
		return chooseString(strings.TrimSpace(reminiscence.AnchorDetail), "Follow up on shared memory")
	}
}

func buildINQuery(prefix string, suffix string, ids []string) (string, []any) {
	args := make([]any, 0, len(ids))
	placeholders := make([]string, 0, len(ids))
	for index, id := range ids {
		args = append(args, id)
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
	}
	return prefix + strings.Join(placeholders, ", ") + suffix, args
}

type createReminderParams struct {
	PatientID                    string
	SourceCallRunID              string
	SourceAnalysisResultID       string
	Kind                         string
	Status                       string
	Title                        string
	Detail                       string
	PersonID                     string
	CaregiverFollowUpRecommended bool
	SuggestedFor                 *time.Time
	CreatedBy                    string
	CreatedAt                    time.Time
}

func (s *PostgresStore) getCallTemplateByIDTx(ctx context.Context, tx *sql.Tx, templateID string) (CallTemplate, bool, error) {
	row := tx.QueryRowContext(ctx, callTemplateSelectBase+` where id = $1`, strings.TrimSpace(templateID))
	template, err := scanCallTemplate(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return CallTemplate{}, false, nil
		}
		return CallTemplate{}, false, fmt.Errorf("get call template tx: %w", err)
	}
	return template, true, nil
}
