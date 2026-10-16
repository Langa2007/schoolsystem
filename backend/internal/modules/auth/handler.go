package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"elimu-core/backend/internal/pkg/response"
)

type Handler struct {
	Service *Service
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		response.Error(w, http.StatusInternalServerError, "authentication service is not configured")
		return
	}
	var request LoginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	request.Identifier = strings.TrimSpace(request.Identifier)
	if request.Identifier == "" || request.Password == "" {
		response.Error(w, http.StatusBadRequest, "identifier and password are required")
		return
	}
	session, err := h.Service.Authenticate(r.Context(), request.Identifier, request.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		response.Error(w, http.StatusInternalServerError, "authentication failed")
		return
	}
	_ = response.JSON(w, http.StatusOK, session)
}
