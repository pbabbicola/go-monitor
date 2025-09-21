package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/consumers/log"
	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/monitor"
)

func TestConsume(t *testing.T) {
	// This test does not need to be a table test because we can test in one test all branches.
	//
	// temporaryLog is a temporary type to unmarshal the expected log arguments.
	type temporaryLog struct {
		URL           string        `json:"url"`
		Duration      time.Duration `json:"duration"`
		StatusCode    int           `json:"status_code"`
		RegexpMatches bool          `json:"regexp_matches"`
	}

	expectedLog := temporaryLog{
		URL:           "testURL",
		Duration:      time.Second,
		StatusCode:    http.StatusTeapot,
		RegexpMatches: true,
	}

	// Make a new logger with a buffer, so we can check the logs against each other.
	jsonLog := bytes.Buffer{}

	defaultLogger := slog.Default()

	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonLog, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	defer slog.SetDefault(defaultLogger)

	// Create a cancel-eable context so we can cancel our log receiver

	ctx, cancel := context.WithCancel(context.Background())

	messageQueue := make(chan monitor.Message)

	var wg sync.WaitGroup

	wg.Go(func() {
		messageQueue <- monitor.Message{
			URL:           expectedLog.URL,
			Duration:      expectedLog.Duration,
			StatusCode:    expectedLog.StatusCode,
			RegexpMatches: expectedLog.RegexpMatches,
		}

		cancel()
	})

	wg.Go(func() {
		log.Consume(ctx, messageQueue)
	})

	wg.Wait()

	var newLog temporaryLog
	require.NoError(t, json.Unmarshal(jsonLog.Bytes(), &newLog))
	assert.Equal(t, expectedLog, newLog)
}
