create table if not exists voice_sessions (
	id text primary key,
	patient_id text not null,
	status text not null check (status in ('awaiting_stream', 'streaming', 'disconnect_grace', 'completed', 'failed', 'expired')),
	voice_id text not null,
	input_sample_rate_hz integer not null check (input_sample_rate_hz in (8000, 16000, 24000)),
	output_sample_rate_hz integer not null check (output_sample_rate_hz in (8000, 16000, 24000)),
	endpointing_sensitivity text not null check (endpointing_sensitivity in ('LOW', 'MEDIUM', 'HIGH')),
	model_id text not null,
	aws_region text not null,
	bedrock_region text not null,
	bedrock_session_id text,
	prompt_name text,
	stream_token_hash bytea not null,
	stream_token_expires_at timestamptz not null,
	stream_token_consumed_at timestamptz,
	client_connected_at timestamptz,
	client_disconnected_at timestamptz,
	disconnect_grace_expires_at timestamptz,
	session_expires_at timestamptz,
	last_activity_at timestamptz not null,
	stop_reason text,
	failure_code text,
	failure_message text,
	created_at timestamptz not null default now(),
	ended_at timestamptz,
	updated_at timestamptz not null default now()
);

create index if not exists idx_voice_sessions_patient_id on voice_sessions (patient_id);
create index if not exists idx_voice_sessions_status on voice_sessions (status);

create table if not exists patient_preferences (
	patient_id text primary key,
	default_voice_id text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists voice_transcript_turns (
	id bigint generated always as identity primary key,
	voice_session_id text not null references voice_sessions(id) on delete cascade,
	sequence_no integer not null,
	direction text not null check (direction in ('user', 'assistant')),
	modality text not null check (modality in ('audio', 'text')),
	transcript_text text not null,
	bedrock_session_id text,
	prompt_name text,
	completion_id text,
	content_id text,
	generation_stage text,
	stop_reason text,
	occurred_at timestamptz not null,
	created_at timestamptz not null default now(),
	unique (voice_session_id, sequence_no)
);

create index if not exists idx_voice_transcript_turns_session on voice_transcript_turns (voice_session_id, occurred_at);

create table if not exists voice_usage_events (
	id bigint generated always as identity primary key,
	voice_session_id text not null references voice_sessions(id) on delete cascade,
	sequence_no integer not null,
	bedrock_session_id text,
	prompt_name text,
	completion_id text,
	input_speech_tokens_delta integer not null default 0,
	input_text_tokens_delta integer not null default 0,
	output_speech_tokens_delta integer not null default 0,
	output_text_tokens_delta integer not null default 0,
	total_input_speech_tokens integer,
	total_input_text_tokens integer,
	total_output_speech_tokens integer,
	total_output_text_tokens integer,
	total_input_tokens integer,
	total_output_tokens integer,
	total_tokens integer,
	payload jsonb not null,
	emitted_at timestamptz not null,
	created_at timestamptz not null default now(),
	unique (voice_session_id, sequence_no)
);

create index if not exists idx_voice_usage_events_session on voice_usage_events (voice_session_id, emitted_at);

create table if not exists check_ins (
	id text primary key,
	patient_id text not null,
	source_voice_session_id text unique references voice_sessions(id) on delete set null,
	summary text not null,
	status text not null check (status in ('scheduled', 'completed', 'needs_follow_up')),
	agent text not null,
	reminder text,
	recorded_at timestamptz not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_check_ins_patient_id on check_ins (patient_id, recorded_at desc);
