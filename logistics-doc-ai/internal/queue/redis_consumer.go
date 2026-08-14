package queue

import (
"context"
"encoding/json"
"fmt"
"time"

"github.com/redis/go-redis/v9"
)

// Consumer reads messages from Redis Streams
type Consumer struct {
client  *redis.Client
stream  string
group   string
consumer string
}

// NewConsumer creates a new Redis consumer
func NewConsumer(client *redis.Client, stream, group, consumer string) *Consumer {
return &Consumer{
client:   client,
stream:   stream,
group:    group,
consumer: consumer,
}
}

// Read reads messages from the stream
func (c *Consumer) Read(ctx context.Context, count int64, block time.Duration) ([]Message, error) {
args := &redis.XReadGroupArgs{
Group:    c.group,
Consumer: c.consumer,
Streams:  []string{c.stream, ">"},
Count:    count,
Block:    block,
}

result, err := c.client.XReadGroup(ctx, args).Result()
if err != nil {
if err == redis.Nil {
return nil, nil
}
return nil, fmt.Errorf("read group: %w", err)
}

if len(result) == 0 {
return nil, nil
}

var messages []Message
for _, stream := range result {
for _, msg := range stream.Messages {
dataStr, ok := msg.Values["data"].(string)
if !ok {
continue
}

var procMsg ProcessingMessage
if err := json.Unmarshal([]byte(dataStr), &procMsg); err != nil {
continue
}

messages = append(messages, Message{
ID:      msg.ID,
Payload: procMsg,
})
}
}

return messages, nil
}

// Ack acknowledges a message
func (c *Consumer) Ack(ctx context.Context, msgID string) error {
return c.client.XAck(ctx, c.stream, c.group, msgID).Err()
}

// Message represents a consumed message
type Message struct {
ID      string
Payload ProcessingMessage
}
