package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"elimu-core/backend/internal/database"
)

func main() {
	values, err := readEnvFile("../.env")
	if err != nil {
		fatal(err)
	}
	dsn := envOr("DATABASE_URL", values["DATABASE_URL"])
	if dsn == "" {
		fatal(fmt.Errorf("DATABASE_URL is not set in environment or ../.env"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := database.Open(ctx, dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	schoolID := envOr("SCHOOL_ID", values["SCHOOL_ID"])
	if schoolID == "" {
		schoolID = "default-school"
	}
	schoolName := envOr("SCHOOL_NAME", values["SCHOOL_NAME"])
	if schoolName == "" {
		schoolName = "ElimuCore Demo School"
	}
	schoolMode := envOr("SCHOOL_MODE", values["SCHOOL_MODE"])
	if schoolMode == "" {
		schoolMode = "day"
	}
	publicSchool := true
	if raw := envOr("PUBLIC_SCHOOL", values["PUBLIC_SCHOOL"]); raw != "" {
		parsed, e := strconv.ParseBool(raw)
		if e != nil {
			fatal(fmt.Errorf("PUBLIC_SCHOOL must be true or false"))
		}
		publicSchool = parsed
	}
	if err := database.EnsureSchool(ctx, db, schoolID, schoolName, schoolMode, publicSchool); err != nil {
		fatal(err)
	}
	if err := database.Seed(ctx, db, schoolID); err != nil {
		fatal(err)
	}
	var applied int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&applied); err != nil {
		fatal(err)
	}
	fmt.Printf("Database setup complete: %d migrations applied; school tenant %q is present.\n", applied, schoolID)
	fmt.Println("Seeded standard finance vote heads idempotently; no student or staff rows were invented.")
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func readEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read .env: %w", err)
	}
	defer file.Close()
	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		values[strings.TrimSpace(key)] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
