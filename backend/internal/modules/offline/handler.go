package offline

import (
	"encoding/json"
	"net/http"

	"elimu-core/backend/internal/pkg/response"
)

type Handler struct{ Service *Service }

func (h Handler) Apply(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		response.Error(w, http.StatusServiceUnavailable, "offline sync is unavailable")
		return
	}
	var operation Operation
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&operation); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid sync operation")
		return
	}
	result, err := h.Service.Apply(operation)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	status := http.StatusOK
	if result.Conflict {
		status = http.StatusConflict
	}
	_ = response.JSON(w, status, result)
}
