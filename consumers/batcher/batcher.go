package batcher

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	"github.com/pbabbicola/go-monitor/consumers/batcher/postgres"
	"github.com/pbabbicola/go-monitor/monitor"
)

type WriteOrLog func(ctx context.Context, pool *sql.DB, batch []monitor.Message)

// Batcher keeps the current message batch and stores the connection pool.
type Batcher struct {
	mut        *sync.Mutex
	batch      []monitor.Message
	batchSize  int
	pool       *sql.DB
	writeOrLog WriteOrLog
}

// New creates a new message queue Batcher. It also satisifies the Consumer interface.
// Probably could make it also with nice functional options. WriteOrLog should also be an option but it is only partially separated for testability.
func New(ctx context.Context, batchSize int, databaseURL string) (*Batcher, error) {
	pool, err := postgres.NewConnection(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("establishing new connection: %w", err)
	}

	return &Batcher{
		mut:        &sync.Mutex{},
		batch:      make([]monitor.Message, 0, batchSize),
		batchSize:  batchSize,
		pool:       pool,
		writeOrLog: postgres.WriteOrLog,
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

// Close closes the database pool and logs an error to slog if there is any problem.
func (b *Batcher) Close(ctx context.Context) {
	err := b.pool.Close()
	if err != nil {
		slog.ErrorContext(ctx, "Failed closing the database connection.", slog.String("error", fmt.Sprintf("%s", err)))
	}
}

// Consume consumes the message queue and writes to postgres.
// While doing this, I realised Batcher should definitely be its own package and writeOrLog should have been a dependency injected somewhere as an interface it to make it more testable, but I am running out of time, so I separated it but only partially (postgres is still a subpackage of batcher and batcher is  only prepared to deal with SQL).
func (b *Batcher) Consume(ctx context.Context, messageQueue chan monitor.Message) {
	for {
		select {
		case <-ctx.Done(): // ignore the non-written messages, but you could write them here if you want to ignore the context cancellation
			return
		case msg := <-messageQueue:
			b.Add(msg)

			if len(b.batch) >= b.batchSize { // doesn't matter if it's the exact batch size
				batch := b.DuplicateAndClear()

				go b.writeOrLog(ctx, b.pool, batch)
			}
		}
	}
}
