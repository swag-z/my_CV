package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/logistics-doc-ai/internal/domain"
)

// DocumentRepository handles document persistence
type DocumentRepository struct {
	pool *pgxpool.Pool
}

// NewDocumentRepository creates a new document repository
func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

// Create creates a new document
func (r *DocumentRepository) Create(ctx context.Context, doc *domain.Document) error {
	query := `
		INSERT INTO documents (id, company_id, original_filename, mime_type, file_key, file_size, document_type, status, error_message, attempts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now())
	`

	_, err := r.pool.Exec(ctx, query,
		doc.ID,
		doc.CompanyID,
		doc.OriginalFilename,
		doc.MimeType,
		doc.FileKey,
		doc.FileSize,
		doc.DocumentType,
		doc.Status,
		doc.ErrorMessage,
		doc.Attempts,
	)

	return err
}

// GetByID retrieves a document by ID
func (r *DocumentRepository) GetByID(ctx context.Context, id string) (*domain.Document, error) {
	query := `
		SELECT id, company_id, original_filename, mime_type, file_key, file_size, document_type, status, error_message, attempts, created_at, updated_at
		FROM documents
		WHERE id = $1
	`

	var doc domain.Document
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&doc.ID,
		&doc.CompanyID,
		&doc.OriginalFilename,
		&doc.MimeType,
		&doc.FileKey,
		&doc.FileSize,
		&doc.DocumentType,
		&doc.Status,
		&doc.ErrorMessage,
		&doc.Attempts,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

// List retrieves documents with pagination and filters
func (r *DocumentRepository) List(ctx context.Context, limit, offset int, status, docType *string) ([]*domain.Document, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM documents WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if status != nil && *status != "" {
		countQuery += " AND status = $" + fmtSprintf("%d", argNum)
		args = append(args, *status)
		argNum++
	}
	if docType != nil && *docType != "" {
		countQuery += " AND document_type = $" + fmtSprintf("%d", argNum)
		args = append(args, *docType)
		argNum++
	}

	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := `
		SELECT id, company_id, original_filename, mime_type, file_key, file_size, document_type, status, error_message, attempts, created_at, updated_at
		FROM documents WHERE 1=1
	`
	args = []interface{}{}
	argNum = 1

	if status != nil && *status != "" {
		dataQuery += " AND status = $" + fmtSprintf("%d", argNum)
		args = append(args, *status)
		argNum++
	}
	if docType != nil && *docType != "" {
		dataQuery += " AND document_type = $" + fmtSprintf("%d", argNum)
		args = append(args, *docType)
		argNum++
	}

	dataQuery += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var docs []*domain.Document
	for rows.Next() {
		var doc domain.Document
		err := rows.Scan(
			&doc.ID,
			&doc.CompanyID,
			&doc.OriginalFilename,
			&doc.MimeType,
			&doc.FileKey,
			&doc.FileSize,
			&doc.DocumentType,
			&doc.Status,
			&doc.ErrorMessage,
			&doc.Attempts,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		docs = append(docs, &doc)
	}

	return docs, total, rows.Err()
}

// UpdateStatus updates the status of a document
func (r *DocumentRepository) UpdateStatus(ctx context.Context, id string, status domain.DocumentStatus, errorMessage *string) error {
	query := `
		UPDATE documents 
		SET status = $1, error_message = $2, updated_at = now()
		WHERE id = $3
	`
	_, err := r.pool.Exec(ctx, query, status, errorMessage, id)
	return err
}

// IncrementAttempts increments the attempt counter
func (r *DocumentRepository) IncrementAttempts(ctx context.Context, id string) error {
	query := `
		UPDATE documents 
		SET attempts = attempts + 1, updated_at = now()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// SetError sets an error message on a document
func (r *DocumentRepository) SetError(ctx context.Context, id string, errorMessage string) error {
	query := `
		UPDATE documents 
		SET error_message = $1, updated_at = now()
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, errorMessage, id)
	return err
}

// SetDocumentType sets the document type
func (r *DocumentRepository) SetDocumentType(ctx context.Context, id string, docType domain.DocumentType) error {
	query := `
		UPDATE documents 
		SET document_type = $1, updated_at = now()
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, docType, id)
	return err
}

// Helper to format argument numbers for SQL queries
func fmtSprintf(format string, num int) string {
	return format
}

var _ interface {
	Create(context.Context, *domain.Document) error
	GetByID(context.Context, string) (*domain.Document, error)
	List(context.Context, int, int, *string, *string) ([]*domain.Document, int, error)
	UpdateStatus(context.Context, string, domain.DocumentStatus, *string) error
} = (*DocumentRepository)(nil)
