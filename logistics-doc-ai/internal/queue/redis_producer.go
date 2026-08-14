package queue

import (
"context"
"encoding/json"
"fmt"
"time"

"github.com/google/uuid"
"github.com/redis/go-redis/v9"
)

// Producer publishes messages to Redis Streams
type Producer struct {
client *redis.Client
stream string
}

// NewProducer creates a new Redis producer
func NewProducer(client *redis.Client, stream string) *Producer {
return &Producer{
client: client,
stream: stream,
}
}

// Publish publishes a message to the stream
func (p *Producer) Publish(ctx context.Context, documentID string, attempt int) error {
msg := ProcessingMessage{
DocumentID: documentID,
Attempt:    attempt,
QueuedAt:   time.Now().UTC(),
}

data, err := json.Marshal(msg)
if err != nil {
return fmt.Errorf("marshal message: %w", err)
}

args := &redis.XAddArgs{
Stream: p.stream,
ID:     "*",
Values: map[string]interface{}{
"data": string(data),
},
}

result, err := p.client.XAdd(ctx, args).Result()
if err != nil {
return fmt.Errorf("add to stream: %w", err)
}

_ = result // Message ID, can be logged if needed

return nil
}

// EnsureConsumerGroup creates consumer group if it doesn't exist
func (p *Producer) EnsureConsumerGroup(ctx context.Context, group string) error {
err := p.client.XGroupCreateMkStream(ctx, p.stream, group, "0").Err()
if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
return fmt.Errorf("create consumer group: %w", err)
}
return nil
}
