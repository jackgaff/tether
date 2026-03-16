alter table patients
	add column if not exists profile_photo_data_url text;

alter table memory_bank_entries
	add column if not exists created_by text not null default 'analysis_worker';

do $$
begin
	if not exists (
		select 1
		from pg_constraint
		where conname = 'memory_bank_entries_created_by_check'
	) then
		alter table memory_bank_entries
			add constraint memory_bank_entries_created_by_check
			check (created_by in ('analysis_worker', 'admin'));
	end if;
end
$$;

alter table memory_bank_entries
	alter column source_call_run_id drop not null;

alter table memory_bank_entries
	alter column source_analysis_result_id drop not null;
