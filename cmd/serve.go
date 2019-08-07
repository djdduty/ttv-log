package cmd

import (
	"github.com/djdduty/ttv-log/cmd/server"
	"github.com/spf13/cobra"
)

// serveCmd represents the host command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP/2 APIs",
	Long:  ``,
	Run:   server.RunServe(c),
}

func init() {
	RootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	serveCmd.PersistentFlags().BoolVar(&c.ForceHTTP, "dangerous-force-http", false, "Disable HTTP/2 over TLS (HTTPS) and serve HTTP instead. Never use this in production.")
}
