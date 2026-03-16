alter table patient_memory_profiles
	add column if not exists significant_places jsonb not null default '[]'::jsonb,
	add column if not exists life_chapters jsonb not null default '[]'::jsonb,
	add column if not exists favorite_music jsonb not null default '[]'::jsonb,
	add column if not exists favorite_shows_films jsonb not null default '[]'::jsonb,
	add column if not exists topics_to_revisit jsonb not null default '[]'::jsonb;

create table if not exists patient_people (
	id text primary key,
	patient_id text not null references patients(id) on delete cascade,
	name text not null,
	relationship text,
	status text not null default 'unknown' check (status in ('confirmed_living', 'unknown', 'deceased')),
	relationship_quality text not null default 'unknown' check (relationship_quality in ('close_active', 'unclear', 'estranged', 'unknown')),
	safe_to_suggest_call boolean generated always as (
		status = 'confirmed_living' and relationship_quality = 'close_active'
	) stored,
	first_mentioned_at timestamptz not null,
	first_mentioned_call_run_id text references call_runs(id) on delete set null,
	last_mentioned_at timestamptz not null,
	last_mentioned_call_run_id text references call_runs(id) on delete set null,
	context text,
	notes text,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_patient_people_patient_last_mentioned
	on patient_people (patient_id, last_mentioned_at desc, created_at desc);

create table if not exists memory_bank_entries (
	id text primary key,
	patient_id text not null references patients(id) on delete cascade,
	source_call_run_id text not null references call_runs(id) on delete cascade,
	source_analysis_result_id text not null references analysis_results(id) on delete cascade,
	topic text not null,
	summary text not null,
	emotional_tone text,
	responded_well_to jsonb not null default '[]'::jsonb,
	anchor_offered boolean not null default false,
	anchor_type text not null default 'none' check (anchor_type in ('call', 'music', 'show_film', 'journal', 'none')),
	anchor_accepted boolean not null default false,
	anchor_detail text,
	suggested_followup text,
	occurred_at timestamptz not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_memory_bank_entries_patient_occurred
	on memory_bank_entries (patient_id, occurred_at desc, created_at desc);

create table if not exists memory_bank_entry_people (
	memory_bank_entry_id text not null references memory_bank_entries(id) on delete cascade,
	patient_person_id text not null references patient_people(id) on delete cascade,
	created_at timestamptz not null default now(),
	primary key (memory_bank_entry_id, patient_person_id)
);

create table if not exists patient_reminders (
	id text primary key,
	patient_id text not null references patients(id) on delete cascade,
	source_call_run_id text references call_runs(id) on delete set null,
	source_analysis_result_id text references analysis_results(id) on delete set null,
	kind text not null check (kind in ('call_person', 'music', 'show_film', 'journal', 'appointment', 'general')),
	status text not null check (status in ('pending', 'completed', 'declined', 'cancelled')),
	title text not null,
	detail text,
	person_id text references patient_people(id) on delete set null,
	caregiver_follow_up_recommended boolean not null default false,
	suggested_for timestamptz,
	created_by text not null check (created_by in ('analysis_worker', 'admin')),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_patient_reminders_patient_created
	on patient_reminders (patient_id, created_at desc);

create index if not exists idx_patient_reminders_person
	on patient_reminders (person_id)
	where person_id is not null;
