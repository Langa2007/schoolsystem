package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"elimu-core/backend/internal/api"
	"elimu-core/backend/internal/config"
	databasepkg "elimu-core/backend/internal/database"
	"elimu-core/backend/internal/domain"
	"elimu-core/backend/internal/modules/auth"
	"elimu-core/backend/internal/modules/finance"
	"elimu-core/backend/internal/modules/school_config"
	"elimu-core/backend/internal/pkg/daraja"
	"elimu-core/backend/internal/pkg/response"
	"elimu-core/backend/internal/pkg/sms"
)

func main() {
	cfg := config.FromEnvironment()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	if cfg.Environment == "production" {
		log.Fatal("production startup is disabled: domain repositories are still in-memory; refusing to lose business data on restart")
	}
	var db *sql.DB
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		var err error
		db, err = databasepkg.Open(ctx, cfg.DatabaseURL)
		if err == nil {
			err = databasepkg.EnsureSchool(ctx, db, cfg.SchoolID, cfg.SchoolName, cfg.SchoolMode, cfg.PublicSchool)
		}
		cancel()
		if err != nil {
			log.Fatalf("initialize database: %v", err)
		}
		log.Print("PostgreSQL connected and migrations applied")
	}
	schoolService, err := school_config.NewService(school_config.SchoolSettings{
		ID: cfg.SchoolID, Name: cfg.SchoolName,
		Mode: school_config.SchoolMode(cfg.SchoolMode), PublicSchool: cfg.PublicSchool,
	}, sms.NewMemoryAdapter())
	if err != nil {
		log.Fatalf("initialize school service: %v", err)
	}
	if cfg.SessionSecret == "" {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			log.Fatalf("generate development session secret: %v", err)
		}
		cfg.SessionSecret = string(secret)
		log.Print("SESSION_SECRET is unset; using an ephemeral development secret (sessions expire on restart)")
	}
	if cfg.BootstrapAdminPassword != "" {
		admin := domain.User{ID: cfg.BootstrapAdminID, SchoolID: cfg.SchoolID, Name: "School Administrator", Roles: []domain.Role{domain.RolePrincipal, domain.RoleRegistrar}}
		if err := schoolService.AddUser(admin); err != nil {
			log.Fatalf("bootstrap administrator: %v", err)
		}
		if err := schoolService.SetPassword(admin, admin.ID, cfg.BootstrapAdminPassword); err != nil {
			log.Fatalf("set bootstrap administrator password: %v", err)
		}
		log.Printf("bootstrapped administrator account %q", admin.ID)
	}
	authService := &auth.Service{Domain: schoolService, Secret: []byte(cfg.SessionSecret), TTL: cfg.SessionTTL}
	var darajaClient daraja.Client
	if cfg.DarajaConsumerKey != "" && cfg.DarajaConsumerSecret != "" && cfg.DarajaPasskey != "" && cfg.DarajaCallbackURL != "" && cfg.DarajaBusinessShortCode != "" {
		darajaClient = daraja.NewHTTPClient(cfg.DarajaBaseURL, cfg.DarajaConsumerKey, cfg.DarajaConsumerSecret, cfg.DarajaPasskey, cfg.DarajaCallbackURL)
	}
	handler := routes(schoolService, authService, db, darajaClient, cfg.DarajaBusinessShortCode, cfg.MPesaCallbackToken, cfg.CORSAllowedOrigins)
	server := &http.Server{
		Addr: cfg.HTTPAddress, Handler: handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout, ReadTimeout: cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout, IdleTimeout: cfg.IdleTimeout,
	}

	go func() {
		log.Printf("ElimuCore API listening on %s", cfg.HTTPAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}
	if db != nil {
		_ = db.Close()
	}
}

func routes(schoolService *school_config.Service, authService *auth.Service, db *sql.DB, darajaClient daraja.Client, darajaShortCode, mpesaCallbackToken, allowedOrigins string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if db != nil {
			if err := db.PingContext(r.Context()); err != nil {
				_ = response.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "database": "unavailable"})
				return
			}
		}
		_ = response.JSON(w, http.StatusOK, map[string]any{"status": "ready", "persistence": "in_memory", "durable": false})
	})
	mux.HandleFunc("GET /api/v1/settings", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.JSON(w, http.StatusOK, schoolService.Settings())
	})
	mux.HandleFunc("POST /api/v1/auth/login", authServiceHandler(authService).Login)
	protect := func(handler http.HandlerFunc) http.Handler { return authService.Middleware(handler) }
	(api.Handler{Service: schoolService}).Register(mux, func(handler http.HandlerFunc) http.Handler { return protect(handler) })
	mux.Handle("GET /api/v1/auth/me", protect(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := auth.ActorFromContext(r.Context())
		if !ok {
			_ = response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		_ = response.JSON(w, http.StatusOK, actor)
	}))

	// Phase 3: Finance, M-Pesa & Fee-Gate routes.
	// DarajaClient remains unavailable until all Daraja settings are configured.
	fh := finance.Handler{Service: schoolService, DarajaClient: darajaClient, DarajaShortCode: darajaShortCode, MPesaCallbackToken: mpesaCallbackToken}
	mux.Handle("POST /api/v1/finance/vote-heads", protect(fh.CreateVoteHead))
	mux.Handle("POST /api/v1/finance/invoices", protect(fh.CreateInvoice))
	mux.Handle("GET /api/v1/finance/ledger/{student_id}", protect(fh.StudentLedger))
	mux.Handle("POST /api/v1/finance/ledger/reverse", protect(fh.ReverseLedgerEntry))
	mux.Handle("POST /api/v1/finance/fee-withholding", protect(fh.SetFeeWithholding))
	mux.Handle("POST /api/v1/finance/waivers", protect(fh.GrantFeeWaiver))
	mux.Handle("GET /api/v1/finance/results-access/{student_id}", protect(fh.CanViewResults))
	// M-Pesa Daraja C2B callback uses its dedicated callback token, not a user session.
	mux.HandleFunc("POST /api/v1/finance/mpesa/callback", fh.MPesaC2BCallback)
	mux.Handle("POST /api/v1/finance/mpesa/stk-push", protect(fh.InitiateSTKPush))
	mux.Handle("GET /api/v1/finance/mpesa/unmatched", protect(fh.ListUnmatchedPayments))
	mux.Handle("POST /api/v1/finance/mpesa/unmatched/resolve", protect(fh.ResolveUnmatchedPayment))

	mux.HandleFunc("GET /api/v1/phase-three", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.JSON(w, http.StatusOK, map[string]any{
			"implemented": []string{
				"vote_heads", "fee_invoices", "append_only_double_entry_ledger",
				"ledger_reversal_with_segregation_of_duties",
				"mpesa_c2b_callback_allocation", "unmatched_payment_queue",
				"principal_fee_withholding", "individual_fee_waivers",
				"results_access_gate", "stk_push_boundary",
				"daraja_http_client_with_token_caching",
			},
			"open": []string{"production_daraja_credentials", "persistent_sqlc_runtime_wiring"},
		})
	})
	mux.HandleFunc("GET /api/v1/phase-four", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.JSON(w, http.StatusOK, map[string]any{
			"implemented": []string{"cp_sat_lesson_solver_contract", "go_hard_constraint_revalidation", "discipline_incidents", "counseling_access_control", "boarding_allocations", "exeat_tracking", "sickbay_referrals", "library_replacement_charges", "inventory_transactions", "bulk_sms_boundary"},
			"open":        []string{"solver_http_client_and_job_registration", "timetable_and_operations_api_endpoints", "persistent_repository_wiring", "exam_and_duty_roster_solver_endpoints", "frontend_timetable_editor", "production_sms_gateway", "library_lost_book_ledger_reconciliation"},
		})
	})
	mux.HandleFunc("GET /api/v1/phase-five", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.JSON(w, http.StatusOK, map[string]any{
			"implemented": []string{"indexeddb_offline_queue", "idempotent_sync_contract", "conflict_detection", "ussd_fee_and_attendance_queries", "tenant_isolation_boundary", "nemis_csv_export", "knec_csv_export", "performance_deviation_analysis", "installable_pwa_shell"},
			"open":        []string{"persistent_sync_repository_wiring", "production_ussd_gateway", "certified_ministry_templates", "horizontal_load_validation"},
		})
	})
	return securityHeaders(cors(mux, allowedOrigins))
}

func cors(next http.Handler, allowedOrigins string) http.Handler {
	allowed := make(map[string]bool)
	for _, origin := range strings.Split(allowedOrigins, ",") {
		if value := strings.TrimSpace(origin); value != "" {
			allowed[value] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
		}
		if r.Method == http.MethodOptions {
			if !allowed[origin] {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authServiceHandler(service *auth.Service) auth.Handler {
	return auth.Handler{Service: service}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
