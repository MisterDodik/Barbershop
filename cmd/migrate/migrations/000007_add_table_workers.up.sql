CREATE TABLE workers (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    working_hours JSONB,
);