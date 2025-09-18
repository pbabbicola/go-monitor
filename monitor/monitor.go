package monitor

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	client       *http.Client
	messageQueue chan Message
}

// NewDefaultMonitorer creates a new default monitorer with an http client.
func NewDefaultMonitorer(client *http.Client, messageQueue chan Message) *DefaultMonitorer {
	return &DefaultMonitorer{
		client:       client,
		messageQueue: messageQueue,
	}
}

var (
	ErrNilMonitorer = errors.New("monitorer is nil")
	ErrNilClient    = errors.New("client is nil")
)

// Message is a monitoring message. It adds all the possible data that a monitor may want to show.
// Here I could have created two message types, and two queues, but I am running out of time.
type Message struct {
	URL           string
	Duration      time.Duration
	Timestamp     time.Time
	StatusCode    int
	RegexpMatches bool
	Err           error
}

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

	responseBody, err := io.ReadAll(resp.Body) // if status == 200?
	if err != nil {
		return fmt.Errorf("reading response body for %v: %w", website, err)
	}

	var regexpMatches bool

	if website.Regexp != nil {
		website.Regexp.Match(responseBody)
	}

	m.messageQueue <- Message{
		URL:           website.URL,
		Duration:      duration,
		Timestamp:     start,
		StatusCode:    resp.StatusCode,
		RegexpMatches: regexpMatches,
		Err:           nil,
	}

	return nil
}
