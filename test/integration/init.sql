CREATE TABLE IF NOT EXISTS merge_requests (
    id SERIAL PRIMARY KEY,
    target_project_id INTEGER NOT NULL,
    iid INTEGER NOT NULL,
    description TEXT,
    merge_status VARCHAR(50) NOT NULL DEFAULT 'unchecked',
    merge_error TEXT
);

INSERT INTO merge_requests (target_project_id, iid, description, merge_status)
VALUES
    (42, 1, 'Test MR 1', 'unchecked'),
    (42, 2, 'Test MR 2', 'unchecked'),
    (42, 3, 'Test MR 3', 'can_be_merged');
