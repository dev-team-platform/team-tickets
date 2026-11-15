CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    action VARCHAR(20) NOT NULL,
    trace_id VARCHAR(64) NULL,
    span_id VARCHAR(32) NULL,
    request_id VARCHAR(255) NULL,
    transaction_id VARCHAR(255) NULL,
    client_action_id VARCHAR(255) NULL,
    client_request_id VARCHAR(255) NULL,
    entity_name VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    old_values JSONB NULL,
    new_values JSONB NULL,
    actor_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
CREATE INDEX idx_audit_logs_entity_name_entity_id ON audit_logs (entity_name, entity_id);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id);
CREATE INDEX idx_audit_logs_transaction_id ON audit_logs (transaction_id);
