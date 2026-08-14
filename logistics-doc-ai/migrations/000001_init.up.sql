-- +goose Up
-- +goose StatementBegin

-- Companies table
CREATE TABLE IF NOT EXISTS companies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    inn text,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Documents table
CREATE TABLE IF NOT EXISTS documents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id uuid NULL REFERENCES companies(id),
    original_filename text NOT NULL,
    mime_type text NOT NULL,
    file_key text NOT NULL,
    file_size bigint NOT NULL,
    document_type text,
    status text NOT NULL DEFAULT 'uploaded',
    error_message text,
    attempts int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Document extractions table
CREATE TABLE IF NOT EXISTS document_extractions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    document_type text NOT NULL,
    payload jsonb NOT NULL,
    confidence jsonb NOT NULL DEFAULT '{}',
    validation jsonb NOT NULL DEFAULT '{}',
    source text NOT NULL,
    version int NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Review corrections table
CREATE TABLE IF NOT EXISTS review_corrections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    field_path text NOT NULL,
    old_value text,
    new_value text,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Export jobs table
CREATE TABLE IF NOT EXISTS export_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    format text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    result_file_key text,
    error_message text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id uuid NULL REFERENCES documents(id) ON DELETE SET NULL,
    action text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_document_type ON documents(document_type);
CREATE INDEX IF NOT EXISTS idx_documents_created_at_desc ON documents(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_document_extractions_document_id ON document_extractions(document_id, version DESC);
CREATE INDEX IF NOT EXISTS idx_export_jobs_document_id ON export_jobs(document_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_document_id ON audit_logs(document_id);

-- Seed default company
INSERT INTO companies (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Company')
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS export_jobs;
DROP TABLE IF EXISTS review_corrections;
DROP TABLE IF EXISTS document_extractions;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS companies;

-- +goose StatementEnd
