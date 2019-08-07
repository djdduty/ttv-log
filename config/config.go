package config

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/djdduty/ttv-log/health"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Config ...
type Config struct {
	BindPort            int    `mapstructure:"PORT" yaml:"-"`
	BindHost            string `mapstructure:"HOST" yaml:"-"`
	ForceHTTP           bool   `yaml:"-"`
	AllowTLSTermination string `mapstructure:"HTTPS_ALLOW_TERMINATION_FROM" yaml:"-"`
	LogLevel            string `mapstructure:"LOG_LEVEL" yaml:"-"`
	LogFormat           string `mapstructure:"LOG_FORMAT" yaml:"-"`

	ElasticHost string `mapstructure:"ELASTIC_HOST" yaml:"-"`
	ElasticUser string `mapstructure:"ELASTIC_USER" yaml:"-"`
	ElasticPass string `mapstructure:"ELASTIC_PASS" yaml:"-"`

	TwitchUser     string `mapstructure:"TWITCH_USER" yaml:"-"`
	TwitchPass     string `mapstructure:"TWITCH_PASS" yaml:"-"`
	TwitchClientID string `mapstructure:"TWITCH_CLIENT_ID" yaml:"-"`

	BuildVersion string         `yaml:"-"`
	BuildHash    string         `yaml:"-"`
	BuildTime    string         `yaml:"-"`
	logger       *logrus.Logger `yaml:"-"`
	context      *Context       `yaml:"-"`

	StreamWhilelist []string `mapstructure:"STREAM_WHITELIST" yaml:"-"`
}

func newLogger(c *Config) *logrus.Logger {
	var (
		err    error
		logger = logrus.New()
	)

	if c.LogFormat == "json" {
		logger.Formatter = new(logrus.JSONFormatter)
	}

	logger.Level, err = logrus.ParseLevel(c.LogLevel)
	if err != nil {
		logger.Errorf("Couldn't parse log level: %s", c.LogLevel)
		logger.Level = logrus.InfoLevel
	}

	return logger
}

// GetLogger returns the configured logger
func (c *Config) GetLogger() *logrus.Logger {
	if c.logger == nil {
		c.logger = newLogger(c)
	}

	return c.logger
}

// GetAddress ...
func (c *Config) GetAddress() string {
	if strings.HasPrefix(c.BindHost, "unix:") {
		return c.BindHost
	}

	return fmt.Sprintf("%s:%d", c.BindHost, c.BindPort)
}

// DoesRequestSatisfyTermination ...
func (c *Config) DoesRequestSatisfyTermination(r *http.Request) error {
	if c.AllowTLSTermination == "" {
		return errors.New("TLS termination is not enabled")
	}

	if r.URL.Path == health.AliveCheckPath || r.URL.Path == health.ReadyCheckPath {
		return nil
	}

	ranges := strings.Split(c.AllowTLSTermination, ",")
	if err := matchesRange(r, ranges); err != nil {
		return err
	}

	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		return errors.New("X-Forwarded-Proto header is missing")
	} else if proto != "https" {
		return errors.Errorf("Expected X-Forwarded-Proto header to be https, got %s", proto)
	}

	return nil
}

func matchesRange(r *http.Request, ranges []string) error {
	_, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return errors.WithStack(err)
	}

	// TODO: Fill this out
	return nil
}

// Context lazily gets the context from config
func (c *Config) Context() *Context {
	if c.context != nil {
		return c.context
	}

	if c.ElasticHost == "" {
		c.GetLogger().Fatalf(`ELASTIC_HOST is not set, use "export ELASTIC_HOST=url".`)
	}

	if c.TwitchUser == "" {
		c.GetLogger().Fatalf(`TWITCH_USER is not set, use "export TWITCH_USER=user".`)
	}

	if c.TwitchPass == "" {
		c.GetLogger().Fatalf(`TWITCH_PASS is not set, use "export TWITCH_PASS=pass".`)
	}

	if c.TwitchClientID == "" {
		c.GetLogger().Fatalf(`TWITCH_CLIENT_ID is not set, use "export TWITCH_CLIENT_ID=client-id".`)
	}

	connection := &ElasticConnector{}
	if err := connection.Init(c.ElasticHost, c.ElasticUser, c.ElasticPass, c.GetLogger()); err != nil {
		c.GetLogger().Fatalf(`Could not connect to elasticsearch cluster: %s`, err)
	}

	c.context = &Context{
		ElasticConnection: connection,
	}

	return c.context
}
