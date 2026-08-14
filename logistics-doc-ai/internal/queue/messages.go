package queue

import "time"

// ProcessingMessage represents a message in the processing queue
type ProcessingMessage struct {
DocumentID string    `json:"document_id"`
Attempt    int       `json:"attempt"`
QueuedAt   time.Time `json:"queued_at"`
}

// StreamName is the name of the Redis stream for document processing
const StreamName = "documents:processing"

// ConsumerGroup is the name of the consumer group
const ConsumerGroup = "doc-workers"
