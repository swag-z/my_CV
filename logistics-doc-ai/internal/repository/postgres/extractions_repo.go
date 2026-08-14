package repository

import (
"context"
"encoding/json"

"github.com/jackc/pgx/v5"
"github.com/jackc/pgx/v5/pgxpool"
"github.com/logistics-doc-ai/internal/domain"
)

// ExtractionRepository handles document extraction persistence
type ExtractionRepository struct {
pool *pgxpool.Pool
}

// NewExtractionRepository creates a new extraction repository
func NewExtractionRepository(pool *pgxpool.Pool) *ExtractionRepository {
return &ExtractionRepository{pool: pool}
}

// Create creates a new extraction record
func (r *ExtractionRepository) Create(ctx context.Context, ext *domain.DocumentExtraction) error {
query := `
INSERT INTO document_extractions (id, document_id, document_type, payload, confidence, validation, source, version, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
`

payloadJSON, _ := json.Marshal(ext.Payload)
confidenceJSON, _ := json.Marshal(ext.Confidence)
validationJSON, _ := json.Marshal(ext.Validation)

_, err := r.pool.Exec(ctx, query,
ext.ID,
ext.DocumentID,
ext.DocumentType,
payloadJSON,
confidenceJSON,
validationJSON,
ext.Source,
ext.Version,
)

return err
}

// GetLatestByDocumentID retrieves the latest extraction for a document
func (r *ExtractionRepository) GetLatestByDocumentID(ctx context.Context, documentID string) (*domain.DocumentExtraction, error) {
query := `
SELECT id, document_id, document_type, payload, confidence, validation, source, version, created_at
FROM document_extractions
WHERE document_id = $1
ORDER BY version DESC
LIMIT 1
`

var ext domain.DocumentExtraction
var payloadJSON, confidenceJSON, validationJSON []byte

err := r.pool.QueryRow(ctx, query, documentID).Scan(
&ext.ID,
&ext.DocumentID,
&ext.DocumentType,
&payloadJSON,
&confidenceJSON,
&validationJSON,
&ext.Source,
&ext.Version,
&ext.CreatedAt,
)

if err == pgx.ErrNoRows {
return nil, nil
}
if err != nil {
return nil, err
}

json.Unmarshal(payloadJSON, &ext.Payload)
json.Unmarshal(confidenceJSON, &ext.Confidence)
json.Unmarshal(validationJSON, &ext.Validation)

return &ext, nil
}

// GetByVersion retrieves an extraction by document ID and version
func (r *ExtractionRepository) GetByVersion(ctx context.Context, documentID string, version int) (*domain.DocumentExtraction, error) {
query := `
SELECT id, document_id, document_type, payload, confidence, validation, source, version, created_at
FROM document_extractions
WHERE document_id = $1 AND version = $2
`

var ext domain.DocumentExtraction
var payloadJSON, confidenceJSON, validationJSON []byte

err := r.pool.QueryRow(ctx, query, documentID, version).Scan(
&ext.ID,
&ext.DocumentID,
&ext.DocumentType,
&payloadJSON,
&confidenceJSON,
&validationJSON,
&ext.Source,
&ext.Version,
&ext.CreatedAt,
)

if err == pgx.ErrNoRows {
return nil, nil
}
if err != nil {
return nil, err
}

json.Unmarshal(payloadJSON, &ext.Payload)
json.Unmarshal(confidenceJSON, &ext.Confidence)
json.Unmarshal(validationJSON, &ext.Validation)

return &ext, nil
}

// GetNextVersion returns the next version number for a document
func (r *ExtractionRepository) GetNextVersion(ctx context.Context, documentID string) (int, error) {
query := `
SELECT COALESCE(MAX(version), 0) + 1
FROM document_extractions
WHERE document_id = $1
`

var nextVersion int
err := r.pool.QueryRow(ctx, query, documentID).Scan(&nextVersion)
if err != nil {
return 1, err
}

return nextVersion, nil
}
