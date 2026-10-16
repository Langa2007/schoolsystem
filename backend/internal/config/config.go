package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddress       string
	DatabaseURL       string
	Environment       string
	SchoolID          string
	SchoolName        string
	SchoolMode        string
	PublicSchool      bool
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	SessionTTL        time.Duration
	SessionSecret     string
}

func FromEnvironment() Config {
	return Config{
		HTTPAddress:       env("HTTP_ADDRESS", ":8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		Environment:       env("APP_ENV", "development"),
		SchoolID:          env("SCHOOL_ID", "default-school"),
		SchoolName:        env("SCHOOL_NAME", "ElimuCore Demo School"),
		SchoolMode:        env("SCHOOL_MODE", "day"),
		PublicSchool:      boolEnv("PUBLIC_SCHOOL", true),
		ShutdownTimeout:   durationEnv("SHUTDOWN_TIMEOUT", 15*time.Second),
		ReadHeaderTimeout: durationEnv("READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:       durationEnv("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:      durationEnv("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:       durationEnv("IDLE_TIMEOUT", 60*time.Second),
		SessionTTL:        durationEnv("SESSION_TTL", 8*time.Hour),
		SessionSecret:     os.Getenv("SESSION_SECRET"),
	}
}

func (c Config) Validate() error {
	if c.HTTPAddress == "" {
		return errors.New("HTTP_ADDRESS is required")
	}
	if _, _, err := net.SplitHostPort(c.HTTPAddress); err != nil {
		return fmt.Errorf("invalid HTTP_ADDRESS: %w", err)
	}
	if c.Environment != "development" && c.Environment != "test" && c.Environment != "production" {
		return fmt.Errorf("invalid APP_ENV %q", c.Environment)
	}
	if c.SchoolID == "" || c.SchoolName == "" {
		return errors.New("SCHOOL_ID and SCHOOL_NAME are required")
	}
	switch c.SchoolMode {
	case "day", "boarding", "mixed":
	default:
		return fmt.Errorf("invalid SCHOOL_MODE %q", c.SchoolMode)
	}
	if c.Environment == "production" {
		if c.DatabaseURL == "" {
			return errors.New("DATABASE_URL is required in production")
		}
		if len(c.SessionSecret) < 32 {
			return errors.New("SESSION_SECRET must contain at least 32 characters in production")
		}
	}
	if c.SessionTTL <= 0 || c.ShutdownTimeout <= 0 {
		return errors.New("session and shutdown timeouts must be positive")
	}
	return nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
