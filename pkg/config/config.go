package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultClientTimeout = 10 * time.Second
	defaultHTTPPort      = 9747
	defaultRetryCount    = 3
)

// InitConfig initializes a config and configure viper to receive config from file and environment.
func InitConfig() (*zap.Logger, error) {
	log, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Unable to create logger", zap.Error(err))
	}

	// Find a home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		return log, fmt.Errorf("unable to determine home directory: %w", err)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName(".statuspage-exporter")
	viper.SetEnvPrefix("")

	viper.AutomaticEnv() // read in environment variables that match

	viper.SetDefault("http_port", defaultHTTPPort)
	viper.SetDefault("client_timeout", defaultClientTimeout)
	viper.SetDefault("retry_count", defaultRetryCount)

	// If a config file found, read it in.
	readConfigErr := viper.ReadInConfig()
	if readConfigErr == nil {
		log.Warn("Using config file: " + viper.ConfigFileUsed())
	}

	zpConfig := zap.NewProductionConfig()
	zpConfig.OutputPaths = []string{"stdout"}
	zpConfig.ErrorOutputPaths = []string{"stdout"}

	level, err := zapcore.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		log.Warn("Unable to parse log level", zap.Error(err))
	} else {
		zpConfig.Level = zap.NewAtomicLevelAt(level)
	}

	log, err = zpConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("unable to build logger: %w", err)
	}

	return log, nil
}

// HTTPPort returns a port for http server.
func HTTPPort() int {
	return viper.GetInt("http_port")
}

// ClientTimeout returns a timeout for http client.
func ClientTimeout() time.Duration {
	return viper.GetDuration("client_timeout")
}

// RetryCount returns amount of retries for http client.
func RetryCount() int {
	return viper.GetInt("retry_count")
}
