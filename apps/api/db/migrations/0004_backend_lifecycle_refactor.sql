create table if not exists patient_memory_profiles (
	patient_id text primary key references patients(id) on delete cascade,
	likes jsonb not null default '[]'::jsonb,
	family_members jsonb not null default '[]'::jsonb,
	life_events jsonb not null default '[]'::jsonb,
	reminiscence_notes text,
	preferred_greeting_style text,
	calming_topics jsonb not null default '[]'::jsonb,
	upsetting_topics jsonb not null default '[]'::jsonb,
	hearing_or_pacing_notes text,
	best_time_of_day text,
	do_not_mention jsonb not null default '[]'::jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into patient_memory_profiles (patient_id)
select p.id
from patients p
on conflict (patient_id) do nothing;

create table if not exists screening_schedules (
	patient_id text primary key references patients(id) on delete cascade,
	enabled boolean not null default false,
	cadence text not null default 'weekly' check (cadence in ('weekly', 'biweekly')),
	timezone text not null,
	preferred_weekday integer not null default 1 check (preferred_weekday between 0 and 6),
	preferred_local_time text not null default '09:00',
	next_due_at timestamptz,
	last_scheduled_window_start timestamptz,
	last_scheduled_window_end timestamptz,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into screening_schedules (patient_id, timezone)
select p.id, p.timezone
from patients p
on conflict (patient_id) do nothing;

alter table voice_transcript_turns
	add column if not exists speaker_role text;

update voice_transcript_turns
set speaker_role = case
	when direction = 'user' then 'patient'
	when direction = 'assistant' then 'agent'
	else null
end
where speaker_role is null;

alter table voice_transcript_turns
	drop constraint if exists voice_transcript_turns_speaker_role_check;

alter table voice_transcript_turns
	add constraint voice_transcript_turns_speaker_role_check
	check (speaker_role is null or speaker_role in ('patient', 'agent', 'caregiver', 'system'));

alter table call_templates
	add column if not exists call_prompt_version text,
	add column if not exists analysis_prompt_version text,
	add column if not exists analysis_prompt_template text;

update call_templates
set call_prompt_version = coalesce(nullif(call_prompt_version, ''), prompt_version, 'v1'),
	analysis_prompt_version = coalesce(nullif(analysis_prompt_version, ''), prompt_version, 'v1'),
	analysis_prompt_template = coalesce(
		nullif(analysis_prompt_template, ''),
		'You are Echo''s structured extraction worker. Return JSON only using the requested schema and avoid diagnostic language.'
	);

alter table call_templates
	alter column call_prompt_version set not null;

alter table call_templates
	alter column analysis_prompt_version set not null;

alter table call_templates
	alter column analysis_prompt_template set not null;

drop index if exists idx_call_templates_active_call_type;

alter table call_templates
	drop constraint if exists call_templates_call_type_check;

update call_templates
set call_type = 'check_in'
where call_type in ('orientation', 'reminder', 'wellbeing');

update call_templates
set is_active = false
where slug in ('orientation-v1', 'reminder-v1', 'wellbeing-v1', 'reminiscence-v1');

alter table call_templates
	add constraint call_templates_call_type_check
	check (call_type in ('screening', 'check_in', 'reminiscence'));

create unique index if not exists idx_call_templates_active_call_type
on call_templates (call_type)
where is_active;

alter table call_runs
	add column if not exists call_type text,
	add column if not exists schedule_window_start timestamptz,
	add column if not exists schedule_window_end timestamptz;

update call_runs cr
set call_type = ct.call_type
from call_templates ct
where ct.id = cr.call_template_id
  and cr.call_type is null;

update call_runs
set trigger_type = 'caregiver_requested'
where trigger_type = 'manual';

update call_runs
set trigger_type = 'follow_up_recommendation'
where trigger_type = 'approved_next_call';

update call_runs
set call_type = 'check_in'
where call_type in ('orientation', 'reminder', 'wellbeing');

alter table call_runs
	alter column call_type set not null;

alter table call_runs
	drop constraint if exists call_runs_trigger_type_check;

alter table call_runs
	drop constraint if exists call_runs_status_check;

alter table call_runs
	add constraint call_runs_trigger_type_check
	check (trigger_type in ('caregiver_requested', 'scheduled', 'follow_up_recommendation'));

alter table call_runs
	add constraint call_runs_status_check
	check (status in ('scheduled', 'requested', 'in_progress', 'completed', 'failed', 'cancelled'));

alter table call_runs
	drop constraint if exists call_runs_call_type_check;

alter table call_runs
	add constraint call_runs_call_type_check
	check (call_type in ('screening', 'check_in', 'reminiscence'));

create unique index if not exists idx_call_runs_schedule_window
on call_runs (patient_id, call_type, schedule_window_start, schedule_window_end)
where schedule_window_start is not null and schedule_window_end is not null;

alter table analysis_results
	add column if not exists call_template_id text references call_templates(id) on delete set null,
	add column if not exists call_prompt_version text,
	add column if not exists analysis_prompt_version text,
	add column if not exists analysis_schema_version text,
	add column if not exists model_provider text,
	add column if not exists model_name text,
	add column if not exists generated_at timestamptz,
	add column if not exists caregiver_review_reason text,
	add column if not exists follow_up_requested_by_patient boolean not null default false,
	add column if not exists follow_up_evidence text;

update analysis_results ar
set call_template_id = cr.call_template_id
from call_runs cr
where cr.id = ar.call_run_id
  and ar.call_template_id is null;

update analysis_results
set call_prompt_version = coalesce(nullif(call_prompt_version, ''), schema_version, 'v1'),
	analysis_prompt_version = coalesce(nullif(analysis_prompt_version, ''), schema_version, 'v1'),
	analysis_schema_version = coalesce(nullif(analysis_schema_version, ''), schema_version, 'v2'),
	model_provider = coalesce(nullif(model_provider, ''), 'amazon'),
	model_name = coalesce(nullif(model_name, ''), model_id),
	generated_at = coalesce(generated_at, created_at);

alter table analysis_results
	alter column call_prompt_version set not null;

alter table analysis_results
	alter column analysis_prompt_version set not null;

alter table analysis_results
	alter column analysis_schema_version set not null;

alter table analysis_results
	alter column model_provider set not null;

alter table analysis_results
	alter column model_name set not null;

alter table analysis_results
	alter column generated_at set not null;

update analysis_results
set recommended_call_type = 'check_in'
where recommended_call_type in ('orientation', 'reminder', 'wellbeing');

alter table analysis_results
	drop constraint if exists analysis_results_recommended_call_type_check;

alter table analysis_results
	add constraint analysis_results_recommended_call_type_check
	check (recommended_call_type in ('screening', 'check_in', 'reminiscence'));

create table if not exists analysis_jobs (
	id text primary key,
	call_run_id text not null unique references call_runs(id) on delete cascade,
	status text not null check (status in ('pending', 'running', 'succeeded', 'failed')),
	attempt_count integer not null default 0,
	last_error text,
	locked_at timestamptz,
	started_at timestamptz,
	finished_at timestamptz,
	analysis_prompt_version text not null,
	analysis_schema_version text not null,
	model_provider text not null,
	model_name text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_analysis_jobs_status_created_at
on analysis_jobs (status, created_at);

alter table next_call_plans
	add column if not exists suggested_window_start_at timestamptz,
	add column if not exists suggested_window_end_at timestamptz,
	add column if not exists follow_up_requested_by_patient boolean not null default false,
	add column if not exists follow_up_evidence text,
	add column if not exists caregiver_review_reason text;

update next_call_plans
set call_type = 'check_in'
where call_type in ('orientation', 'reminder', 'wellbeing');

alter table next_call_plans
	drop constraint if exists next_call_plans_call_type_check;

alter table next_call_plans
	add constraint next_call_plans_call_type_check
	check (call_type in ('screening', 'check_in', 'reminiscence'));

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
	is_active
) values
	(
		'tmpl-screening-v1',
		'screening-v1',
		'Screening Call',
		'screening',
		'Structured supportive screening call with checklist-driven prompts.',
		8,
		'v1',
		'v1',
		'You are Echo, a calm and respectful AI support assistant. This is a structured screening call. Ask one question at a time, keep a neutral tone, and avoid diagnosis or treatment language.',
		'v1',
		'You are Echo''s extraction worker for screening calls. Return structured JSON only and use observational, non-diagnostic language.',
		'["Administer the screening prompts in order.","Do not skip ahead without acknowledging missing responses.","Mark the call partial if it ends early."]'::jsonb,
		true
	),
	(
		'tmpl-check-in-v1',
		'check-in-v1',
		'Check-In Call',
		'check_in',
		'Routine wellbeing and day-in-the-life check-in with optional follow-up request detection.',
		6,
		'v1',
		'v1',
		'You are Echo, a calm and respectful AI support assistant. This is a check-in call. Ask about the person''s day, what they ate or drank, how they are feeling, and whether they want another check-in soon.',
		'v1',
		'You are Echo''s extraction worker for check-in calls. Return structured JSON only and capture follow-up requests conservatively.',
		'["Ask about the day so far.","Ask about food or hydration.","Ask about mood or comfort.","Ask whether they would like another check-in soon."]'::jsonb,
		true
	),
	(
		'tmpl-reminiscence-v2',
		'reminiscence-v2',
		'Reminiscence Call',
		'reminiscence',
		'Comfort-focused reminiscence call grounded in familiar people, places, and memories.',
		8,
		'v2',
		'v2',
		'You are Echo, a calm and respectful AI support assistant. This is a reminiscence call. Focus on comfort, familiar memories, and a gentle return to the present. Avoid quizzing or diagnosis.',
		'v2',
		'You are Echo''s extraction worker for reminiscence calls. Return structured JSON only and identify supportive future topics without medical claims.',
		'["Start with a familiar person, place, or activity.","Follow the patient''s lead.","Notice signs of comfort or distress.","End with a gentle grounding close."]'::jsonb,
		true
	)
on conflict (slug) do update
set display_name = excluded.display_name,
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
	updated_at = now();
