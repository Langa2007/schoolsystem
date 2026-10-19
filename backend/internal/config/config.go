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
	HTTPAddress             string
	DatabaseURL             string
	Environment             string
	SchoolID                string
	SchoolName              string
	SchoolMode              string
	PublicSchool            bool
	ShutdownTimeout         time.Duration
	ReadHeaderTimeout       time.Duration
	ReadTimeout             time.Duration
	WriteTimeout            time.Duration
	IdleTimeout             time.Duration
	SessionTTL              time.Duration
	SessionSecret           string
	BootstrapAdminID        string
	BootstrapAdminPassword  string
	DarajaBaseURL           string
	DarajaConsumerKey       string
	DarajaConsumerSecret    string
	DarajaPasskey           string
	DarajaCallbackURL       string
	DarajaBusinessShortCode string
	MPesaCallbackToken      string
	CORSAllowedOrigins      string
}

func FromEnvironment() Config {
	return Config{
		HTTPAddress:             env("HTTP_ADDRESS", ":8080"),
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		Environment:             env("APP_ENV", "development"),
		SchoolID:                env("SCHOOL_ID", "default-school"),
		SchoolName:              env("SCHOOL_NAME", "ElimuCore Demo School"),
		SchoolMode:              env("SCHOOL_MODE", "day"),
		PublicSchool:            boolEnv("PUBLIC_SCHOOL", true),
		ShutdownTimeout:         durationEnv("SHUTDOWN_TIMEOUT", 15*time.Second),
		ReadHeaderTimeout:       durationEnv("READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:             durationEnv("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:            durationEnv("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:             durationEnv("IDLE_TIMEOUT", 60*time.Second),
		SessionTTL:              durationEnv("SESSION_TTL", 8*time.Hour),
		SessionSecret:           os.Getenv("SESSION_SECRET"),
		BootstrapAdminID:        env("BOOTSTRAP_ADMIN_ID", "admin"),
		BootstrapAdminPassword:  os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		DarajaBaseURL:           env("DARAJA_BASE_URL", "https://sandbox.safaricom.co.ke"),
		DarajaConsumerKey:       os.Getenv("DARAJA_CONSUMER_KEY"),
		DarajaConsumerSecret:    os.Getenv("DARAJA_CONSUMER_SECRET"),
		DarajaPasskey:           os.Getenv("DARAJA_PASSKEY"),
		DarajaCallbackURL:       os.Getenv("DARAJA_CALLBACK_URL"),
		DarajaBusinessShortCode: os.Getenv("DARAJA_BUSINESS_SHORT_CODE"),
		MPesaCallbackToken:      os.Getenv("MPESA_CALLBACK_TOKEN"),
		CORSAllowedOrigins:      env("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173"),
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
	if c.BootstrapAdminPassword != "" && c.BootstrapAdminID == "" {
		return errors.New("BOOTSTRAP_ADMIN_ID is required when BOOTSTRAP_ADMIN_PASSWORD is set")
	}
	if c.MPesaCallbackToken != "" && len(c.MPesaCallbackToken) < 32 {
		return errors.New("MPESA_CALLBACK_TOKEN must contain at least 32 characters")
	}
	darajaValues := []string{c.DarajaConsumerKey, c.DarajaConsumerSecret, c.DarajaPasskey, c.DarajaCallbackURL, c.DarajaBusinessShortCode}
	darajaConfigured := 0
	for _, value := range darajaValues {
		if value != "" {
			darajaConfigured++
		}
	}
	if darajaConfigured != 0 && darajaConfigured != len(darajaValues) {
		return errors.New("DARAJA_CONSUMER_KEY, DARAJA_CONSUMER_SECRET, DARAJA_PASSKEY, DARAJA_CALLBACK_URL, and DARAJA_BUSINESS_SHORT_CODE must be configured together")
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
