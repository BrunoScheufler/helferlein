package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/brunoscheufler/helferlein/worker"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func defaultConfig() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not load working directory: %w", err)
	}

	return filepath.Join(pwd, "config.yml"), nil
}

func setupLogger(level string) (*logrus.Logger, error) {
	parsedLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("could not parse log level %q: %w", level, err)
	}

	logger := logrus.New()
	logger.SetLevel(parsedLevel)
	return logger, nil
}

func main() {
	// Get default config path
	defaultConfigPath, err := defaultConfig()
	if err != nil {
		logrus.Fatalf("Could not load default config: %s", err.Error())
	}

	// Read log level from env or fall back to INFO default
	defaultLogLevel := os.Getenv("LOG_LEVEL")
	if defaultLogLevel == "" {
		defaultLogLevel = "INFO"
	}

	// Define and parse flags
	logLevel := flag.String("loglevel", defaultLogLevel, "Log level to use")
	configFile := flag.String("config", defaultConfigPath, "Path of the configuration file to be used")
	flag.Parse()

	// Set up logger
	logger, err := setupLogger(*logLevel)
	if err != nil {
		logrus.Fatalf("Could not setup logger: %s", err.Error())
	}

	// Create and load config
	config := &worker.Config{}
	err = config.LoadFromFile(*configFile)
	if err != nil {
		logrus.Fatalf("Could not load config: %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Infof("Received signal %s, shutting down..", sig.String())
		cancel()
	}()

	// Start worker
	err = worker.Start(ctx, config, logger)
	if err != nil {
		logrus.Fatalf("Failed to start up: %s", err.Error())
	}
}
