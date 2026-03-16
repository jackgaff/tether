package prompts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

//go:embed files/**/*.md
var files embed.FS

type TemplateDefinition struct {
	ID              string
	Slug            string
	DisplayName     string
	CallType        string
	Description     string
	DurationMinutes int
	CallFile        string
	AnalysisFile    string
	Checklist       []string
}

type RenderContext struct {
	PatientFirstName             string
	CurrentWeekday               string
	CurrentDateLong              string
	RoutineAnchorsBlock          string
	FavoriteTopicsBlock          string
	CalmingCuesBlock             string
	TopicsToAvoidBlock           string
	KnownInterestsBlock          string
	SignificantPlacesBlock       string
	LifeChaptersBlock            string
	FavoriteMusicBlock           string
	FavoriteShowsFilmsBlock      string
	TopicsToRevisitBlock         string
	SafePeopleForCallAnchorBlock string
	PeopleToAvoidNamingBlock     string
	RecentMemoryFollowUpsBlock   string
}

var activeTemplates = []TemplateDefinition{
	{
		ID:              "tmpl-check-in",
		Slug:            "check-in",
		DisplayName:     "Check-In Call",
		CallType:        "check_in",
		Description:     "Routine wellbeing and day-in-the-life check-in with reminder capture and quiet delirium watch.",
		DurationMinutes: 6,
		CallFile:        "files/check-in/call.md",
		AnalysisFile:    "files/check-in/analysis.md",
		Checklist: []string{
			"Open warmly and orient once.",
			"Capture meals, fluids, activities, mood, and sleep.",
			"Capture reminder requests or declined reminders.",
			"Record quiet delirium-watch observations only in structured notes.",
		},
	},
	{
		ID:              "tmpl-reminiscence",
		Slug:            "reminiscence",
		DisplayName:     "Reminiscence Call",
		CallType:        "reminiscence",
		Description:     "Comfort-focused reminiscence call with memory-bank capture and one optional real-world anchor.",
		DurationMinutes: 8,
		CallFile:        "files/reminiscence/call.md",
		AnalysisFile:    "files/reminiscence/analysis.md",
		Checklist: []string{
			"Find one topic and stay with it.",
			"Reflect something specific before closing.",
			"Offer at most one real-world anchor.",
			"Only name a person in a call anchor if they are verified safe.",
		},
	},
}

var legacyTemplateIDs = []string{
	"tmpl-screening-v1",
	"tmpl-check-in-v1",
	"tmpl-reminiscence-v1",
	"tmpl-reminiscence-v2",
	"tmpl-orientation-v1",
	"tmpl-reminder-v1",
	"tmpl-wellbeing-v1",
}

func Definitions() []TemplateDefinition {
	result := make([]TemplateDefinition, len(activeTemplates))
	copy(result, activeTemplates)
	return result
}

func ActiveCallTypes() []string {
	types := make([]string, 0, len(activeTemplates))
	for _, definition := range activeTemplates {
		types = append(types, definition.CallType)
	}
	return types
}

func SyncCallTemplates(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is required")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin prompt sync tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		update call_templates
		set is_active = false,
		    updated_at = $3
		where call_type in ('screening', 'check_in', 'reminiscence')
		  and id not in ($1, $2)
	`, activeTemplates[0].ID, activeTemplates[1].ID, now); err != nil {
		return fmt.Errorf("deactivate non-current templates: %w", err)
	}

	for _, definition := range activeTemplates {
		callBody, err := loadFile(definition.CallFile)
		if err != nil {
			return err
		}
		analysisBody, err := loadFile(definition.AnalysisFile)
		if err != nil {
			return err
		}
		checklist, err := json.Marshal(definition.Checklist)
		if err != nil {
			return fmt.Errorf("marshal checklist for %s: %w", definition.ID, err)
		}
		callVersion := contentVersion(callBody)
		analysisVersion := contentVersion(analysisBody)

		if _, err := tx.ExecContext(ctx, `
			insert into call_templates (
				id,
				slug,
				display_name,
				call_type,
				description,
				duration_minutes,
				prompt_version,
				call_prompt_version,
				system_prompt_template,
				analysis_prompt_version,
				analysis_prompt_template,
				checklist_json,
				is_active,
				updated_at
			) values ($1, $2, $3, $4, $5, $6, $7, $7, $8, $9, $10, $11, true, $12)
			on conflict (id) do update
			set slug = excluded.slug,
			    display_name = excluded.display_name,
			    call_type = excluded.call_type,
			    description = excluded.description,
			    duration_minutes = excluded.duration_minutes,
			    prompt_version = excluded.prompt_version,
			    call_prompt_version = excluded.call_prompt_version,
			    system_prompt_template = excluded.system_prompt_template,
			    analysis_prompt_version = excluded.analysis_prompt_version,
			    analysis_prompt_template = excluded.analysis_prompt_template,
			    checklist_json = excluded.checklist_json,
			    is_active = excluded.is_active,
			    updated_at = excluded.updated_at
		`, definition.ID, definition.Slug, definition.DisplayName, definition.CallType, definition.Description, definition.DurationMinutes, callVersion, callBody, analysisVersion, analysisBody, checklist, now); err != nil {
			return fmt.Errorf("upsert call template %s: %w", definition.ID, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		update call_templates
		set is_active = false,
		    updated_at = $4
		where id in ($1, $2, $3)
	`, legacyTemplateIDs[0], legacyTemplateIDs[1], legacyTemplateIDs[2], now); err != nil {
		return fmt.Errorf("deactivate legacy templates: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		update call_templates
		set is_active = false,
		    updated_at = $5
		where id in ($1, $2, $3, $4)
	`, legacyTemplateIDs[3], legacyTemplateIDs[4], legacyTemplateIDs[5], legacyTemplateIDs[6], now); err != nil {
		return fmt.Errorf("deactivate legacy templates batch 2: %w", err)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit prompt sync tx: %w", commitErr)
	}

	return nil
}

func RenderCallPrompt(raw string, ctx RenderContext) (string, error) {
	tpl, err := template.New("call-prompt").Option("missingkey=error").Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse call prompt template: %w", err)
	}

	var builder strings.Builder
	if err := tpl.Execute(&builder, ctx); err != nil {
		return "", fmt.Errorf("render call prompt template: %w", err)
	}

	return strings.TrimSpace(builder.String()), nil
}

func loadFile(path string) (string, error) {
	body, err := files.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt file %s: %w", path, err)
	}
	return strings.TrimSpace(string(body)), nil
}

func contentVersion(body string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(body)))
	return fmt.Sprintf("sha256:%x", sum[:6])
}
