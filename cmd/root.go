package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/djdduty/ttv-log/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version is the build version
	Version = "dev-master"
	// BuildTime is the time the build was created
	BuildTime = "undefined"
	// GitHash is the git commit hash of the build
	GitHash = "undefined"
)

var c = new(config.Config)

// RootCmd is the base command without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ttv-log",
	Short: "Twitch Log Ecosystem",
}

// Execute ...
func Execute() {
	c.BuildTime = BuildTime
	c.BuildVersion = Version
	c.BuildHash = GitHash

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.AutomaticEnv()
	viper.BindEnv("HOST")
	viper.SetDefault("HOST", "0.0.0.0")

	viper.BindEnv("PORT")
	viper.SetDefault("PORT", 3000)

	viper.BindEnv("LOG_LEVEL")
	viper.SetDefault("LOG_LEVEL", "info")

	viper.BindEnv("LOG_FORMAT")
	viper.SetDefault("LOG_FORMAT", "json")

	viper.BindEnv("ELASTIC_HOST")
	viper.SetDefault("ELASTIC_HOST", "")

	viper.BindEnv("ELASTIC_USER")
	viper.SetDefault("ELASTIC_USER", "")

	viper.BindEnv("ELASTIC_PASS")
	viper.SetDefault("ELASTIC_PASS", "")

	viper.BindEnv("TWITCH_USER")
	viper.SetDefault("TWITCH_USER", "")

	viper.BindEnv("TWITCH_PASS")
	viper.SetDefault("TWITCH_PASS", "")

	viper.BindEnv("TWITCH_CLIENT_ID")
	viper.SetDefault("TWITCH_CLIENT_ID", "")

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s\n", err))
	}

	if err := viper.Unmarshal(c); err != nil {
		log.Fatal(fmt.Sprintf("Could not read config because %s.", err))
	}
}
