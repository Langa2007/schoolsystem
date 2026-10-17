package timetable

import "net/http"

type Handler struct {
	Service *Service
}

func (h Handler) Health(w http.ResponseWriter, _ *http.Request) {
	if h.Service == nil || h.Service.Solver == nil {
		http.Error(w, "timetable solver unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
