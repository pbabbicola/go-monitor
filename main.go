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

	"github.com/pbabbicola/go-monitor/config"
	"github.com/pbabbicola/go-monitor/consumers/batcher"
	"github.com/pbabbicola/go-monitor/consumers/postgres"
	"github.com/pbabbicola/go-monitor/monitor"
)

func run(envConfig *config.EnvConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		stop := make(chan os.Signal, 1)

		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

		defer signal.Stop(stop)

		<-stop
		cancel()
	}()

	client := cleanhttp.DefaultClient() // This sets sensible defaults for the client.

	cfg, err := config.ParseRemote(ctx, client, envConfig.FileURL)
	if err != nil {
		return fmt.Errorf("parsing configuration: %w", err)
	}

	pool, err := postgres.NewConsumer(ctx, envConfig.DatabaseURL)
	if err != nil {
		return fmt.Errorf("creating postgres consumer: %w", err)
	}
	defer pool.Close(ctx)

	batchQueue := make(chan []monitor.Message, len(cfg)) // buffer in case that someone is waiting to send while the context cancellation happens. overkill? maybe. better ways? most likely

	batch, err := batcher.New(ctx, envConfig.BatchSize, batchQueue)
	if err != nil {
		return fmt.Errorf("creating batcher: %w", err)
	}

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

	wg.Go(func() {
		pool.Consume(ctx, batchQueue)
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
