package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port       string
	MetricsPort string

	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration

	MaxConcurrentRequests int
	MaxChecksPerRequest   int
	MaxURLsPerRequest     int

	CheckDefaultTimeout time.Duration
	CheckMaxTimeout     time.Duration
	CheckUserAgent      string

	LogLevel  string
	LogFormat string
}

func Load() (Config, error) {
	c := Config{
		Port:                  getEnv("PORT", "8080"),
		MetricsPort:           getEnv("METRICS_PORT", "9090"),
		ShutdownTimeout:       mustDuration("SHUTDOWN_TIMEOUT", "30s"),
		ReadTimeout:           mustDuration("READ_TIMEOUT", "5s"),
		WriteTimeout:          mustDuration("WRITE_TIMEOUT", "30s"),
		IdleTimeout:           mustDuration("IDLE_TIMEOUT", "120s"),
		MaxConcurrentRequests: mustInt("MAX_CONCURRENT_REQUESTS", "100"),
		MaxChecksPerRequest:   mustInt("MAX_CHECKS_PER_REQUEST", "50"),
		MaxURLsPerRequest:     mustInt("MAX_URLS_PER_REQUEST", "200"),
		CheckDefaultTimeout:   mustDuration("CHECK_DEFAULT_TIMEOUT", "3s"),
		CheckMaxTimeout:       mustDuration("CHECK_MAX_TIMEOUT", "10s"),
		CheckUserAgent:        getEnv("CHECK_USER_AGENT", "url-checker/1.0"),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		LogFormat:             getEnv("LOG_FORMAT", "json"),
	}

	if err := c.validate(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}

	return c, nil
}

func (c Config) validate() error {
	if c.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("MAX_CONCURRENT_REQUESTS must be > 0")
	}
	if c.MaxChecksPerRequest <= 0 {
		return fmt.Errorf("MAX_CHECKS_PER_REQUEST must be > 0")
	}
	if c.MaxURLsPerRequest <= 0 {
		return fmt.Errorf("MAX_URLS_PER_REQUEST must be > 0")
	}
	if c.CheckDefaultTimeout > c.CheckMaxTimeout {
		return fmt.Errorf("CHECK_DEFAULT_TIMEOUT must be <= CHECK_MAX_TIMEOUT")
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustDuration(key, def string) time.Duration {
	raw := getEnv(key, def)
	d, err := time.ParseDuration(raw)
	if err != nil {
		panic(fmt.Sprintf("config: invalid duration for %s=%q: %v", key, raw, err))
	}
	return d
}

func mustInt(key, def string) int {
	raw := getEnv(key, def)
	n, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Sprintf("config: invalid int for %s=%q: %v", key, raw, err))
	}
	return n
}
