package config

import "os"

type Config struct {
	HTTPAddress  string
	SchoolID     string
	SchoolName   string
	SchoolMode   string
	PublicSchool bool
}

func FromEnvironment() Config {
	return Config{
		HTTPAddress:  value("HTTP_ADDRESS", ":8080"),
		SchoolID:     value("SCHOOL_ID", "default-school"),
		SchoolName:   value("SCHOOL_NAME", "ElimuCore Demo School"),
		SchoolMode:   value("SCHOOL_MODE", "day"),
		PublicSchool: value("PUBLIC_SCHOOL", "true") == "true",
	}
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
