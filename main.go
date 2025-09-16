package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/config"
	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/monitor"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/spf13/cobra"
)

// The website monitor should perform the checks periodically and collect the request timestamp,
// the response time, the HTTP status code, as well as optionally checking the returned page
// contents for a regex pattern that is expected to be found on the page. Each URL should be
// checked periodically, with the ability to configure the interval (between 5 and 300 seconds) and
// the regexp on a per-URL basis. The monitored URLs can be anything found online. In case the
// check fails the details of the failure should be logged into the database.

func run(_ *cobra.Command, args []string) error {
	cfg, err := config.Parse(args[0]) // Guaranteed to exist by cobra.ExactArgs.
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

	client := cleanhttp.DefaultClient() // This sets sensible defaults for the client.

	var wg sync.WaitGroup
	for _, website := range cfg { // values don't need to be copied over for correct concurrency since go 1.21
		wg.Go(func() {
			monitor.Ticks(ctx, website, monitor.NewDefaultMonitorer(client).Monitor)
		})
	}

	wg.Wait()

	return nil
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	rootCmd := &cobra.Command{
		Use:   "gomonitor configfile.json",
		Short: "Go Monitor checks a list of websites periodically.",
		Long:  "Go Monitor checks a list of websites periodically.",
		Args:  cobra.ExactArgs(1), // Only allows one argument.
		RunE:  run,
	}

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
