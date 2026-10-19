package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	var err error
	dsn, err = NormalizeDSN(dsn)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if err := Migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// NormalizeDSN accepts standard PostgreSQL URLs and JDBC-prefixed PostgreSQL
// URLs. If the URL omits userinfo, PGUSER and PGPASSWORD may provide it.
func NormalizeDSN(dsn string) (string, error) {
	dsn = strings.TrimSpace(dsn)
	dsn = strings.TrimPrefix(dsn, "jdbc:")
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		return "", fmt.Errorf("DATABASE_URL must be a valid PostgreSQL URL")
	}
	if u.User == nil {
		user := os.Getenv("PGUSER")
		password, hasPassword := os.LookupEnv("PGPASSWORD")
		if user != "" {
			if hasPassword {
				u.User = url.UserPassword(user, password)
			} else {
				u.User = url.User(user)
			}
		}
	}
	return u.String(), nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var applied bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)`, name).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			continue
		}
		contents, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err = tx.ExecContext(ctx, string(contents)); err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name)
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}

func EnsureSchool(ctx context.Context, db *sql.DB, id, name, mode string, public bool) error {
	_, err := db.ExecContext(ctx, `INSERT INTO tenants (id,name,mode,public_school) VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO NOTHING`, id, name, mode, public)
	if err != nil {
		return fmt.Errorf("ensure school tenant: %w", err)
	}
	var storedName, storedMode string
	var storedPublic bool
	if err := db.QueryRowContext(ctx, `SELECT name,mode,public_school FROM tenants WHERE id=$1`, id).Scan(&storedName, &storedMode, &storedPublic); err != nil {
		return fmt.Errorf("read school tenant: %w", err)
	}
	if storedName != name || storedMode != mode || storedPublic != public {
		return fmt.Errorf("configured school %q conflicts with persisted tenant settings", id)
	}
	return nil
}

// Seed inserts only fixed school finance categories; it never invents people,
// admissions, terms, or financial balances.
func Seed(ctx context.Context, db *sql.DB, schoolID string) error {
	voteHeads := []struct{ id, name string }{
		{"tuition", "Tuition"}, {"boarding", "Boarding"}, {"activities", "Activities"},
		{"medical", "Medical"}, {"caution", "Caution Money"}, {"pta-development", "PTA / Development"},
		{"fdse-capitation", "FDSE Capitation"},
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin school seed: %w", err)
	}
	defer tx.Rollback()
	for _, head := range voteHeads {
		id := schoolID + ":" + head.id
		if _, err := tx.ExecContext(ctx, `INSERT INTO vote_heads (id,school_id,name) VALUES ($1,$2,$3) ON CONFLICT (id) DO NOTHING`, id, schoolID, head.name); err != nil {
			return fmt.Errorf("seed vote head %s: %w", head.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit school seed: %w", err)
	}
	return nil
}
