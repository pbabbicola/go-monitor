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

const minTimerSeconds = 5

const maxTimerSeconds = 300

// Monitorer is an interface that is used for passing to Ticks how we want to monitor a certain website.
type Monitorer func(context.Context, config.SiteElement) error

func adjustTimers(ctx context.Context, website config.SiteElement) config.SiteElement {
	if website.IntervalSeconds < minTimerSeconds {
		slog.InfoContext(
			ctx,
			"Interval too small. Will use 5 seconds.",
			slog.String("url", website.URL),
			slog.Int("interval", website.IntervalSeconds),
		)
		website.IntervalSeconds = minTimerSeconds
	}

	if website.IntervalSeconds > maxTimerSeconds {
		slog.InfoContext(
			ctx,
			"Interval too big. Will use 300 seconds.",
			slog.String("url", website.URL),
			slog.Int("interval", website.IntervalSeconds),
		)
		website.IntervalSeconds = maxTimerSeconds
	}

	return website
}

// Ticks creates a ticker that controls the interval for a certain monitor, and will execute the monitorer when the time has passed.
// Adapted from [time.NewTicker] example.
func Ticks(ctx context.Context, website config.SiteElement, monitorer Monitorer) {
	website = adjustTimers(ctx, website)

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
				slog.InfoContext(
					ctx,
					"Failed to monitor",
					slog.String("url", website.URL),
					slog.String("error", err.Error()),
				)
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

	start := time.Now() // theoretically I should do this after creating the request but I've written myself into a corner since I would have to go back and figure out how to modify the logs table, as I set start time as part of the primary key.

	// We build the message so we can send partial results as logs if we don't have the complete result.
	message := Message{
		URL:       website.URL,
		Timestamp: start,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, website.URL, http.NoBody)
	if err != nil {
		message.Err = fmt.Errorf("creating request to %v: %w", website, err)
		m.messageQueue <- message

		return nil
	}

	resp, err := m.client.Do(req)
	if err != nil {
		message.Err = fmt.Errorf("making request to %v: %w", website, err)
		m.messageQueue <- message

		return nil
	}
	defer resp.Body.Close()

	message.Duration = time.Since(start)
	message.StatusCode = resp.StatusCode

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		message.Err = fmt.Errorf("reading response body for %v: %w", website, err)
		m.messageQueue <- message

		return nil
	}

	if website.Regexp != nil {
		message.RegexpMatches = website.Regexp.Match(responseBody)
	}

	m.messageQueue <- message

	return nil
}
