CREATE TABLE worker_profile (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    working_hours JSONB NOT NULL,
    appointment_duration INTERVAL NOT NULL,
    pause_between INTERVAL NOT NULL
);