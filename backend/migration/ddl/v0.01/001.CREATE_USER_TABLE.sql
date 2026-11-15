CREATE TABLE users (
    id UUID PRIMARY KEY,
    identity_subject VARCHAR(255) NOT NULL,
    employee_code VARCHAR(10) NOT NULL,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NULL,
    display_name VARCHAR(200) NOT NULL,

    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by_id UUID NULL,
    updated_at TIMESTAMPTZ NULL,
    updated_by_id UUID NULL,

    CONSTRAINT uq_users_identity_subject UNIQUE (identity_subject),
    CONSTRAINT uq_users_employee_code UNIQUE (employee_code),
    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_email UNIQUE (email),
    CONSTRAINT fk_users_created_by_id FOREIGN KEY (created_by_id) REFERENCES users (id),
    CONSTRAINT fk_users_updated_by_id FOREIGN KEY (updated_by_id) REFERENCES users (id)
);

CREATE INDEX idx_users_display_name ON users (display_name);
