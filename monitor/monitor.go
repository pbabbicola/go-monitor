package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/config"
)

// Monitorer is an interface that is used for passing to Ticks how we want to monitor a certain website.
type Monitorer func(context.Context, config.SiteElement) error

// Ticks creates a ticker that controls the interval for a certain monitor, and will execute the monitorer when the time has passed.
// Adapted from [time.NewTicker] example.
func Ticks(ctx context.Context, website config.SiteElement, monitorer Monitorer) {
	ticker := time.NewTicker(time.Duration(website.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "Done!", slog.String("url", website.URL))
			return
		case t := <-ticker.C:
			err := monitorer(ctx, website)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to monitor", slog.String("url", website.URL), slog.String("error", err.Error()))
			}

			slog.DebugContext(ctx, "Monitored", slog.String("url", website.URL), slog.Time("ticked_time", t))
		}
	}
}

// DefaultMonitorer is the currently implemented monitorer. It makes a request and prints the results.
type DefaultMonitorer struct {
	client *http.Client
}

// NewDefaultMonitorer creates a new default monitorer with an http client.
func NewDefaultMonitorer(client *http.Client) *DefaultMonitorer {
	return &DefaultMonitorer{
		client: client,
	}
}

var (
	ErrNilMonitorer = errors.New("monitorer is nil")
	ErrNilClient    = errors.New("client is nil")
)

// Monitor monitors one website and prints in debug the monitoring information.
func (m *DefaultMonitorer) Monitor(ctx context.Context, website config.SiteElement) error {
	if m == nil {
		return ErrNilMonitorer
	}

	if m.client == nil {
		return ErrNilClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, website.URL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request to %v: %w", website, err)
	}

	start := time.Now()

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("making request to %v: %w", website, err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	slog.DebugContext(ctx, "Request done", slog.String("url", website.URL), slog.Duration("duration", duration), slog.Int("status_code", resp.StatusCode))

	return nil
}
