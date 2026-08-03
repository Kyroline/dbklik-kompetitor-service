// Package queue defines a broker-agnostic contract for background jobs.
// Modules publish domain events or jobs through this interface; the
// concrete driver (SQS, RabbitMQ, in-memory, ...) lives in infrastructure/queue.
package queue

type Message struct {
	Topic   string
	Payload []byte
}

type Handler func(Message) error

type Queue interface {
	Publish(msg Message) error
	Subscribe(topic string, handler Handler)
}

// MemoryQueue is a synchronous in-process Queue, useful as a default
// driver or in tests when no external broker is configured.
type MemoryQueue struct {
	handlers map[string][]Handler
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{handlers: make(map[string][]Handler)}
}

func (q *MemoryQueue) Publish(msg Message) error {
	for _, h := range q.handlers[msg.Topic] {
		if err := h(msg); err != nil {
			return err
		}
	}
	return nil
}

func (q *MemoryQueue) Subscribe(topic string, handler Handler) {
	q.handlers[topic] = append(q.handlers[topic], handler)
}
