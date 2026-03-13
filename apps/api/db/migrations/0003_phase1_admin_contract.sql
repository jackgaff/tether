create table if not exists caregivers (
	id text primary key,
	display_name text not null,
	email text not null,
	phone_e164 text,
	timezone text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists patients (
	id text primary key,
	primary_caregiver_id text not null unique references caregivers(id) on delete restrict,
	display_name text not null,
	preferred_name text not null,
	phone_e164 text,
	timezone text not null,
	notes text,
	calling_state text not null default 'active' check (calling_state in ('active', 'paused')),
	pause_reason text,
	paused_at timestamptz,
	routine_anchors jsonb not null default '[]'::jsonb,
	favorite_topics jsonb not null default '[]'::jsonb,
	calming_cues jsonb not null default '[]'::jsonb,
	topics_to_avoid jsonb not null default '[]'::jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_patients_primary_caregiver_id on patients (primary_caregiver_id);

create table if not exists patient_consent_state (
	patient_id text primary key references patients(id) on delete cascade,
	outbound_call_status text not null check (outbound_call_status in ('pending', 'granted', 'revoked')),
	transcript_storage_status text not null check (transcript_storage_status in ('pending', 'granted', 'revoked')),
	granted_by_caregiver_id text references caregivers(id) on delete set null,
	granted_at timestamptz,
	revoked_at timestamptz,
	notes text,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists call_templates (
	id text primary key,
	slug text not null unique,
	display_name text not null,
	call_type text not null check (call_type in ('orientation', 'reminder', 'wellbeing', 'reminiscence')),
	description text not null,
	duration_minutes integer not null check (duration_minutes > 0 and duration_minutes <= 30),
	prompt_version text not null,
	system_prompt_template text not null,
	checklist_json jsonb not null default '[]'::jsonb,
	is_active boolean not null default true,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create unique index if not exists idx_call_templates_active_call_type on call_templates (call_type) where is_active;

create table if not exists call_runs (
	id text primary key,
	patient_id text not null references patients(id) on delete cascade,
	caregiver_id text not null references caregivers(id) on delete restrict,
	call_template_id text not null references call_templates(id) on delete restrict,
	channel text not null check (channel in ('browser', 'connect')),
	trigger_type text not null check (trigger_type in ('manual', 'approved_next_call')),
	status text not null check (status in ('requested', 'in_progress', 'completed', 'failed', 'cancelled')),
	source_voice_session_id text unique references voice_sessions(id) on delete set null,
	requested_at timestamptz not null,
	started_at timestamptz,
	ended_at timestamptz,
	stop_reason text,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_call_runs_patient_requested_at on call_runs (patient_id, requested_at desc);

create table if not exists analysis_results (
	id text primary key,
	call_run_id text not null unique references call_runs(id) on delete cascade,
	patient_id text not null references patients(id) on delete cascade,
	model_id text not null,
	schema_version text not null,
	raw_result_json jsonb not null,
	dashboard_summary text not null,
	caregiver_summary text not null,
	orientation text not null check (orientation in ('good', 'mixed', 'poor', 'unclear')),
	mood text not null check (mood in ('positive', 'neutral', 'anxious', 'sad', 'distressed', 'unclear')),
	engagement text not null check (engagement in ('high', 'medium', 'low')),
	confidence double precision not null check (confidence >= 0 and confidence <= 1),
	escalation_level text not null check (escalation_level in ('none', 'caregiver_soon', 'caregiver_now', 'clinical_review')),
	recommended_call_type text not null check (recommended_call_type in ('orientation', 'reminder', 'wellbeing', 'reminiscence')),
	recommended_time_note text,
	recommended_duration_minutes integer not null check (recommended_duration_minutes > 0 and recommended_duration_minutes <= 30),
	recommended_goal text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_analysis_results_patient_created_at on analysis_results (patient_id, created_at desc);

create table if not exists risk_flags (
	id text primary key,
	analysis_result_id text not null references analysis_results(id) on delete cascade,
	flag_type text not null,
	severity text not null check (severity in ('info', 'watch', 'urgent')),
	evidence_quote text,
	why_it_matters text,
	confidence double precision not null check (confidence >= 0 and confidence <= 1),
	created_at timestamptz not null default now()
);

create index if not exists idx_risk_flags_analysis_result_id on risk_flags (analysis_result_id);

create table if not exists next_call_plans (
	id text primary key,
	patient_id text not null references patients(id) on delete cascade,
	source_analysis_result_id text not null references analysis_results(id) on delete cascade,
	call_template_id text not null references call_templates(id) on delete restrict,
	call_type text not null check (call_type in ('orientation', 'reminder', 'wellbeing', 'reminiscence')),
	suggested_time_note text,
	planned_for timestamptz,
	duration_minutes integer not null check (duration_minutes > 0 and duration_minutes <= 30),
	goal text not null,
	approval_status text not null check (approval_status in ('pending_approval', 'approved', 'rejected', 'executed', 'superseded', 'cancelled')),
	approved_by_caregiver_id text references caregivers(id) on delete set null,
	approved_by_admin_username text,
	approved_at timestamptz,
	rejection_reason text,
	rejected_at timestamptz,
	executed_call_run_id text unique references call_runs(id) on delete set null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create unique index if not exists idx_next_call_plans_active_patient on next_call_plans (patient_id)
where approval_status in ('pending_approval', 'approved');

insert into call_templates (
	id,
	slug,
	display_name,
	call_type,
	description,
	duration_minutes,
	prompt_version,
	system_prompt_template,
	checklist_json,
	is_active
) values
	(
		'tmpl-orientation-v1',
		'orientation-v1',
		'Orientation Check-In',
		'orientation',
		'Short grounding call focused on today, routine anchors, and one gentle reminder.',
		4,
		'v1',
		'You are Echo, a calm and respectful AI check-in assistant for an older adult with mild memory changes. This is an orientation call. Speak briefly, ask one question at a time, help the person feel calm and grounded, and end with one clear recap.',
		'["How are you feeling right now?","Do you know what part of the day it is?","What is the main plan or reminder for today?"]'::jsonb,
		true
	),
	(
		'tmpl-reminder-v1',
		'reminder-v1',
		'Reminder and Routine',
		'reminder',
		'Brief reminder call focused on one routine or appointment cue.',
		3,
		'v1',
		'You are Echo, a calm and respectful AI check-in assistant for an older adult with mild memory changes. This is a reminder call. Focus on one routine or appointment reminder, confirm understanding gently, and keep the call short.',
		'["Confirm how the person is feeling.","Share one reminder clearly.","Repeat the reminder once if needed.","Close kindly and briefly."]'::jsonb,
		true
	),
	(
		'tmpl-wellbeing-v1',
		'wellbeing-v1',
		'Wellbeing Check',
		'wellbeing',
		'Monitoring call for mood, sleep, food or water, and safety.',
		6,
		'v1',
		'You are Echo, a calm and respectful AI check-in assistant for an older adult with mild memory changes. This is a wellbeing call. Ask one question at a time, keep the pacing gentle, and cover mood, sleep, food or water, routine completion, and safety.',
		'["Ask about mood.","Ask about sleep.","Ask about food or water today.","Ask whether one important routine step was completed.","Ask whether they feel safe or need help."]'::jsonb,
		true
	),
	(
		'tmpl-reminiscence-v1',
		'reminiscence-v1',
		'Reminiscence Call',
		'reminiscence',
		'Companionship-oriented call built around one familiar topic and a soft grounding close.',
		8,
		'v1',
		'You are Echo, a calm and respectful AI check-in assistant for an older adult with mild memory changes. This is a reminiscence call. Focus on comfort and connection, not assessment. Ask at most three memory prompts and end by gently grounding the person in today.',
		'["Start with one familiar topic.","Ask no more than three memory prompts.","Reflect emotion warmly.","Bridge back to the present once before closing."]'::jsonb,
		true
	)
on conflict (slug) do nothing;
