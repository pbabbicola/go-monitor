package log

import (
	"context"
	"log/slog"

	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/monitor"
)

func Consume(ctx context.Context, messageQueue chan monitor.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-messageQueue:
			slog.DebugContext(ctx, "Request done", slog.String("url", msg.URL), slog.Duration("duration", msg.Duration), slog.Int("status_code", msg.StatusCode), slog.Bool("regexp_matches", msg.RegexpMatches))
		}
	}
}
