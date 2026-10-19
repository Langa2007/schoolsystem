package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"elimu-core/backend/internal/domain"
	"elimu-core/backend/internal/pkg/identity"
	"elimu-core/backend/internal/pkg/response"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	Domain *domain.Service
	Secret []byte
	TTL    time.Duration
}

type Session struct {
	AccessToken        string      `json:"access_token"`
	TokenType          string      `json:"token_type"`
	ExpiresAt          time.Time   `json:"expires_at"`
	MustChangePassword bool        `json:"must_change_password"`
	User               domain.User `json:"user"`
}

type tokenClaims struct {
	UserID   string `json:"sub"`
	SchoolID string `json:"school_id"`
	Expires  int64  `json:"exp"`
}

// ActorFromContext returns the authenticated user attached by Middleware.
func ActorFromContext(ctx context.Context) (domain.User, bool) {
	return identity.Actor(ctx)
}

// Middleware verifies a Bearer token and reloads the user so role changes take
// effect immediately instead of trusting role data embedded in a token.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s == nil || s.Domain == nil {
			response.Error(w, http.StatusServiceUnavailable, "authentication unavailable")
			return
		}
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			response.Error(w, http.StatusUnauthorized, "authentication required")
			return
		}
		user, err := VerifyToken(strings.TrimSpace(strings.TrimPrefix(header, prefix)), s.Secret, time.Now())
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		current, ok := s.Domain.User(user.ID)
		if !ok || current.SchoolID != user.SchoolID {
			response.Error(w, http.StatusUnauthorized, "account unavailable")
			return
		}
		ctx := identity.WithActor(r.Context(), current)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) Authenticate(_ context.Context, identifier, password string) (Session, error) {
	if s == nil || s.Domain == nil || len(s.Secret) < 32 || s.TTL <= 0 {
		return Session{}, errors.New("authentication service is not configured")
	}
	user, err := s.Domain.Authenticate(identifier, password)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	expires := time.Now().UTC().Add(s.TTL)
	claims := tokenClaims{UserID: user.ID, SchoolID: user.SchoolID, Expires: expires.Unix()}
	token, err := signClaims(claims, s.Secret)
	if err != nil {
		return Session{}, fmt.Errorf("issue session: %w", err)
	}
	return Session{AccessToken: token, TokenType: "Bearer", ExpiresAt: expires, MustChangePassword: s.Domain.PasswordChangeRequired(user.ID), User: user}, nil
}

func VerifyToken(token string, secret []byte, now time.Time) (domain.User, error) {
	if len(secret) < 32 {
		return domain.User{}, errors.New("session secret is not configured")
	}
	claims, err := verifyClaims(token, secret)
	if err != nil || claims.Expires <= now.Unix() || claims.UserID == "" || claims.SchoolID == "" {
		return domain.User{}, ErrInvalidCredentials
	}
	return domain.User{ID: claims.UserID, SchoolID: claims.SchoolID}, nil
}

func signClaims(claims tokenClaims, secret []byte) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "ELIMU"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := hmac.New(sha256.New, secret)
	_, _ = signature.Write([]byte(encodedHeader + "." + encodedPayload))
	return encodedHeader + "." + encodedPayload + "." + base64.RawURLEncoding.EncodeToString(signature.Sum(nil)), nil
}

func verifyClaims(token string, secret []byte) (tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return tokenClaims{}, ErrInvalidCredentials
	}
	signature := hmac.New(sha256.New, secret)
	_, _ = signature.Write([]byte(parts[0] + "." + parts[1]))
	expected := signature.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(actual, expected) {
		return tokenClaims{}, ErrInvalidCredentials
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}, ErrInvalidCredentials
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return tokenClaims{}, ErrInvalidCredentials
	}
	return claims, nil
}
