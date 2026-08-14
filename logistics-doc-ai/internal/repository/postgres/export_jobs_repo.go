package repository

import (
"context"
"time"

"github.com/jackc/pgx/v5"
"github.com/jackc/pgx/v5/pgxpool"
"github.com/logistics-doc-ai/internal/domain"
)

// ExportJobRepository handles export job persistence
type ExportJobRepository struct {
pool *pgxpool.Pool
}

// NewExportJobRepository creates a new export job repository
func NewExportJobRepository(pool *pgxpool.Pool) *ExportJobRepository {
return &ExportJobRepository{pool: pool}
}

// Create creates a new export job
func (r *ExportJobRepository) Create(ctx context.Context, job *domain.ExportJob) error {
query := `
INSERT INTO export_jobs (id, document_id, format, status, result_file_key, error_message, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now(), now())
`

_, err := r.pool.Exec(ctx, query,
job.ID,
job.DocumentID,
job.Format,
job.Status,
job.ResultFileKey,
job.ErrorMessage,
)

return err
}

// GetByID retrieves an export job by ID
func (r *ExportJobRepository) GetByID(ctx context.Context, id string) (*domain.ExportJob, error) {
query := `
SELECT id, document_id, format, status, result_file_key, error_message, created_at, updated_at
FROM export_jobs
WHERE id = $1
`

var job domain.ExportJob
err := r.pool.QueryRow(ctx, query, id).Scan(
&job.ID,
&job.DocumentID,
&job.Format,
&job.Status,
&job.ResultFileKey,
&job.ErrorMessage,
&job.CreatedAt,
&job.UpdatedAt,
)

if err == pgx.ErrNoRows {
return nil, nil
}
if err != nil {
return nil, err
}

return &job, nil
}

// UpdateStatus updates the status of an export job
func (r *ExportJobRepository) UpdateStatus(ctx context.Context, id string, status domain.ExportStatus, resultFileKey *string, errorMessage *string) error {
query := `
UPDATE export_jobs 
SET status = $1, result_file_key = $2, error_message = $3, updated_at = now()
WHERE id = $4
`
_, err := r.pool.Exec(ctx, query, status, resultFileKey, errorMessage, id)
return err
}

// SetCompleted marks an export job as completed
func (r *ExportJobRepository) SetCompleted(ctx context.Context, id string, resultFileKey string) error {
return r.UpdateStatus(ctx, id, domain.ExportStatusCompleted, &resultFileKey, nil)
}

// SetFailed marks an export job as failed
func (r *ExportJobRepository) SetFailed(ctx context.Context, id string, errorMessage string) error {
return r.UpdateStatus(ctx, id, domain.ExportStatusFailed, nil, &errorMessage)
}
