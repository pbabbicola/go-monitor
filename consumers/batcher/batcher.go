package batcher

import (
	"context"
	"sync"

	"github.com/pbabbicola/go-monitor/monitor"
)

// Batcher keeps the current message batch and stores the connection pool.
type Batcher struct {
	mut        *sync.Mutex
	batch      []monitor.Message
	batchSize  int
	batchQueue chan []monitor.Message
}

// New creates a new message queue Batcher. It also satisifies the implicit Consumer interface.
func New(ctx context.Context, batchSize int, batchQueue chan []monitor.Message) (*Batcher, error) {
	return &Batcher{
		mut:        &sync.Mutex{},
		batch:      make([]monitor.Message, 0, batchSize),
		batchSize:  batchSize,
		batchQueue: batchQueue,
	}, nil
}

// Add is a concurrency-safe append for the message slice.
func (b *Batcher) Add(message monitor.Message) {
	b.mut.Lock()
	b.batch = append(b.batch, message)
	b.mut.Unlock()
}

// DuplicateAndClear returns the current batch and clears it to continue using it. It's safe to use concurrently.
func (b *Batcher) DuplicateAndClear() []monitor.Message {
	b.mut.Lock()
	duplicated := b.batch                             // copies the pointer to the previous batch
	b.batch = make([]monitor.Message, 0, b.batchSize) // reassigns the batch to a new batch
	b.mut.Unlock()

	return duplicated
}

// Consume consumes the message queue, creates the batch and sends it to postgres.
func (b *Batcher) Consume(ctx context.Context, messageQueue chan monitor.Message) {
	for {
		select {
		case <-ctx.Done(): // ignore the non-written messages, but you could write them here if you want to ignore the context cancellation
			return
		case msg := <-messageQueue:
			b.Add(msg)

			if len(b.batch) >= b.batchSize { // doesn't matter if it's the exact batch 
				batch := b.DuplicateAndClear()

				b.batchQueue <- batch
			}
		}
	}
}
