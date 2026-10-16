package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"elimu-core/backend/internal/config"
	"elimu-core/backend/internal/modules/auth"
	"elimu-core/backend/internal/modules/school_config"
	"elimu-core/backend/internal/pkg/response"
	"elimu-core/backend/internal/pkg/sms"
)

func main() {
	cfg := config.FromEnvironment()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	schoolService, err := school_config.NewService(school_config.SchoolSettings{
		ID: cfg.SchoolID, Name: cfg.SchoolName,
		Mode: school_config.SchoolMode(cfg.SchoolMode), PublicSchool: cfg.PublicSchool,
	}, sms.NewMemoryAdapter())
	if err != nil {
		log.Fatalf("initialize school service: %v", err)
	}
	authService := &auth.Service{Domain: schoolService, Secret: []byte(cfg.SessionSecret), TTL: cfg.SessionTTL}
	handler := routes(schoolService, authService)
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
}

func routes(schoolService *school_config.Service, authService *auth.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/settings", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.JSON(w, http.StatusOK, schoolService.Settings())
	})
	mux.HandleFunc("POST /api/v1/auth/login", authServiceHandler(authService).Login)
	mux.HandleFunc("GET /api/v1/phase-three", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.JSON(w, http.StatusOK, map[string]any{
			"implemented": []string{"vote_heads", "fee_invoices", "append_only_double_entry_ledger", "mpesa_callback_allocation", "unmatched_payment_queue", "principal_fee_withholding", "fee_waivers", "daraja_stk_push_boundary"},
			"open":        []string{"production_daraja_credentials", "persistent_sqlc_runtime_wiring"},
		})
	})
	return securityHeaders(mux)
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
