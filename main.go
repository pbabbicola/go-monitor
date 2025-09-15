package main

import (
	"os"

	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/config"
	"github.com/spf13/cobra"
)

// The website monitor should perform the checks periodically and collect the request timestamp,
// the response time, the HTTP status code, as well as optionally checking the returned page
// contents for a regex pattern that is expected to be found on the page. Each URL should be
// checked periodically, with the ability to configure the interval (between 5 and 300 seconds) and
// the regexp on a per-URL basis. The monitored URLs can be anything found online. In case the
// check fails the details of the failure should be logged into the database.

func run(cmd *cobra.Command, args []string) {
	config.Parse(args[0])
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "gomonitor configfile.json",
		Short: "Go Monitor checks a list of websites periodically.",
		Long:  "Go Monitor checks a list of websites periodically.",
		Args:  cobra.ExactArgs(1), // Only allows one argument.
		Run:   run,
	}

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
