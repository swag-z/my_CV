package repository

import (
"context"
"encoding/json"

"github.com/jackc/pgx/v5/pgxpool"
"github.com/logistics-doc-ai/internal/domain"
)

// AuditRepository handles audit log persistence
type AuditRepository struct {
pool *pgxpool.Pool
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
return &AuditRepository{pool: pool}
}

// Log creates a new audit log entry
func (r *AuditRepository) Log(ctx context.Context, log *domain.AuditLog) error {
query := `
INSERT INTO audit_logs (id, document_id, action, payload, created_at)
VALUES ($1, $2, $3, $4, now())
`

payloadJSON, _ := json.Marshal(log.Payload)

_, err := r.pool.Exec(ctx, query,
log.ID,
log.DocumentID,
log.Action,
payloadJSON,
)

return err
}
