-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS export_jobs;
DROP TABLE IF EXISTS review_corrections;
DROP TABLE IF EXISTS document_extractions;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS companies;

-- +goose StatementEnd
