CREATE TABLE file_storage (
    id UUID PRIMARY KEY,
    reference_id VARCHAR(255) NULL,
    object_key VARCHAR(4096) NOT NULL,
    file_name VARCHAR(2048) NOT NULL,
    file_type VARCHAR(20) NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    expired_at TIMESTAMPTZ NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    created_by_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by_id UUID NULL,
    updated_at TIMESTAMPTZ NULL
);

ALTER TABLE file_storage
ADD CONSTRAINT fk_file_storage_created_by_id 
    FOREIGN KEY (created_by_id) REFERENCES users (id),
ADD CONSTRAINT fk_file_storage_updated_by_id 
    FOREIGN KEY (updated_by_id) REFERENCES users (id);

CREATE INDEX idx_file_storage_reference_id ON file_storage (reference_id);
CREATE INDEX idx_file_storage_file_type ON file_storage (file_type);