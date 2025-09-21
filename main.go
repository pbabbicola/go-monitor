package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	cleanhttp "github.com/hashicorp/go-cleanhttp"

	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/config"
	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/consumers/batcher"
	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/monitor"
)

// The website monitor should perform the checks periodically and collect the request timestamp,
// the response time, the HTTP status code, as well as optionally checking the returned page
// contents for a regex pattern that is expected to be found on the page. Each URL should be
// checked periodically, with the ability to configure the interval (between 5 and 300 seconds) and
// the regexp on a per-URL basis. The monitored URLs can be anything found online. In case the
// check fails the details of the failure should be logged into the database.

func run(envConfig *config.EnvConfig) error {
	cfg, err := config.Parse(envConfig.FileURL)
	if err != nil {
		return fmt.Errorf("parsing configuration: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		stop := make(chan os.Signal, 1)

		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

		defer signal.Stop(stop)

		<-stop
		cancel()
	}()

	batch, err := batcher.New(ctx, envConfig.BatchSize, envConfig.DatabaseURL)
	if err != nil {
		return fmt.Errorf("creating batcher: %w", err)
	}
	defer batch.Close(ctx)

	client := cleanhttp.DefaultClient() // This sets sensible defaults for the client.
	messageQueue := make(chan monitor.Message)

	var wg sync.WaitGroup
	for _, website := range cfg { // values don't need to be copied over for correct concurrency since go 1.21
		wg.Go(func() {
			monitor.Ticks(ctx, website, monitor.NewDefaultMonitorer(client, messageQueue).Monitor)
		})
	}

	wg.Go(func() {
		batch.Consume(ctx, messageQueue)
	})

	wg.Wait()

	return nil
}

func main() {
	envConfig, err := config.ParseEnv()
	if err != nil {
		panic(err)
	}

	slog.SetLogLoggerLevel(envConfig.LogLevel)

	err = run(envConfig)
	if err != nil {
		slog.Error("Exiting program.", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
