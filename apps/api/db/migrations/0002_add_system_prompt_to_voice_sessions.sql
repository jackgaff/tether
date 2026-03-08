alter table voice_sessions
add column if not exists system_prompt text;
