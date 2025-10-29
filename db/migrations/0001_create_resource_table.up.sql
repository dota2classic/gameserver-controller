CREATE TABLE IF NOT EXISTS match_resources (
    match_id BIGINT UNIQUE NOT NULL,
    job_name TEXT NOT NULL,
    secret_name TEXT NOT NULL,
    config_map_name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
