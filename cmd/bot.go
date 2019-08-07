package cmd

import (
	"github.com/djdduty/ttv-log/cmd/bot"
	"github.com/spf13/cobra"
)

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Run the IRC bot",
	Long:  ``,
	Run:   bot.RunBot(c),
}

func init() {
	RootCmd.AddCommand(botCmd)
}
