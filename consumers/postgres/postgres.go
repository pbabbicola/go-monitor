package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq" // postgres driver

	"github.com/pbabbicola/go-monitor/monitor"
)

// In a world in which I had more time I would make this maybe nice optional parameters... like:
// type Option func(Options) Option
// type Options struct {
// 	maxIdleConns    *int
// 	maxOpenConns    *int
// 	connMaxIdleTime *time.Duration
// 	connMaxLifetime *time.Duration
// }
// func NewClient(databaseURL string, options ...Option) (*sql.DB, error) {


type Postgres struct {
	pool *sql.DB
}

// NewConsumer creates a Postgres consumer that satisfies the implicit Consumer interface.
func NewConsumer(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := NewConnection(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("establishing new connection: %w", err)
	}

	return &Postgres{
		pool: pool,
	}, nil
}

// NewConnection creates a *sql.DB with default options.
func NewConnection(ctx context.Context, databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(23)                 //nolint:mnd // Explains above why I am not passing this as a configuration. Assumes I need 2 or so for reading somewhere.
	db.SetMaxIdleConns(23)                 //nolint:mnd // Same reason as above, referenced from https://www.alexedwards.net/blog/configuring-sqldb
	db.SetConnMaxLifetime(5 * time.Minute) //nolint:mnd // Same reason as above, references from https://www.alexedwards.net/blog/configuring-sqldb

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("pinging database: %w", err) // Could also reattempt but I've never had this issue of the connection being just flaky.
	}

	return db, nil
}

// Close closes the database pool and logs an error to slog if there is any problem.
func (p *Postgres) Close(ctx context.Context) {
	err := p.pool.Close()
	if err != nil {
		slog.ErrorContext(ctx, "Failed closing the database connection.", slog.String("error", fmt.Sprintf("%s", err)))
	}
}

// Consume consumes the batch queue of monitor.Message and writes them to postgres. It just logs an error in slog if it fails writing a batch.
func (p *Postgres) Consume(ctx context.Context, batchQueue chan []monitor.Message) {
	for {
		select {
		case <-ctx.Done(): // ignore the non-written messages, but you could write them here if you want to ignore the context cancellation
			return
		case batch := <-batchQueue:
			err := writeToPostgres(ctx, p.pool, batch)
			if err != nil {
				slog.ErrorContext(
					ctx,
					"Failed writing to Postgres.",
					slog.String("error", fmt.Sprintf("%s", err)),
				)
			}
		}
	}
}

const insertQuery = "insert into logs (ts, url, duration_milliseconds, status_code, regexp_matches, error) values($1, $2, $3, $4, $5, $6)"

// writeToPostgres writes a batch of inserts in a transaction.
func writeToPostgres(ctx context.Context, pool *sql.DB, batch []monitor.Message) error {
	tx, err := pool.BeginTx(ctx, &sql.TxOptions{}) // do we want a certain isolation level? unknown, this is something I would ask whoever is in charge of the product, because it will affect the experience (eg. phantom reads)
	if err != nil {
		return fmt.Errorf("beginning the transaction: %w", err)
	}

	defer tx.Rollback() //nolint:errcheck // Doesn't matter whether it errors or not.

	stmt, err := tx.PrepareContext(ctx, insertQuery)
	if err != nil {
		return fmt.Errorf("preparing the statement: %w", err)
	}
	defer stmt.Close()

	for _, msg := range batch {
		messageError := ""
		if msg.Err != nil {
			messageError = msg.Err.Error() // There is probably a more elegant way, but also I feel like go should be handling nil errors better.
		}

		_, err := stmt.ExecContext(
			ctx,
			msg.Timestamp,
			msg.URL,
			msg.Duration.Milliseconds(),
			msg.StatusCode,
			msg.RegexpMatches,
			messageError,
		)
		if err != nil { // making the assumption here that we want to keep writing despite the error
			slog.ErrorContext(
				ctx,
				"Failed to execute SQL transaction!",
				slog.Time("timestamp", msg.Timestamp),
				slog.String("url", msg.URL),
				slog.Duration("duration", msg.Duration),
				slog.Int("status_code", msg.StatusCode),
				slog.Bool("regexp_matches", msg.RegexpMatches),
				slog.String("error", fmt.Sprintf("%s", err)),
			)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}
