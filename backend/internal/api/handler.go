// Package api exposes the implemented core domain workflows as versioned HTTP endpoints.
package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"elimu-core/backend/internal/domain"
	"elimu-core/backend/internal/pkg/identity"
	"elimu-core/backend/internal/pkg/response"
)

type Handler struct{ Service *domain.Service }

func (h Handler) Register(mux *http.ServeMux, protect func(http.HandlerFunc) http.Handler) {
	mux.Handle("POST /api/v1/auth/password", protect(h.ChangePassword))
	mux.Handle("POST /api/v1/users", protect(h.CreateUser))
	mux.Handle("GET /api/v1/students", protect(h.ListStudents))
	mux.Handle("POST /api/v1/students", protect(h.CreateStudent))
	mux.Handle("POST /api/v1/staff", protect(h.CreateStaff))
	mux.Handle("GET /api/v1/terms", protect(h.ListTerms))
	mux.Handle("POST /api/v1/terms", protect(h.CreateTerm))
	mux.Handle("GET /api/v1/streams", protect(h.ListStreams))
	mux.Handle("POST /api/v1/streams", protect(h.CreateStream))
	mux.Handle("POST /api/v1/attendance/prefect-count", protect(h.SubmitPrefectCount))
	mux.Handle("POST /api/v1/attendance/morning-roll", protect(h.PostMorningRoll))
	mux.Handle("POST /api/v1/attendance/lessons", protect(h.RecordLesson))
	mux.Handle("POST /api/v1/attendance/reconcile", protect(h.Reconcile))
	mux.Handle("GET /api/v1/attendance/summary", protect(h.AttendanceSummary))
	mux.Handle("POST /api/v1/leave-requests", protect(h.RequestLeave))
	mux.Handle("POST /api/v1/leave-requests/{id}/decision", protect(h.DecideLeave))
	mux.Handle("POST /api/v1/cbc/enrollments", protect(h.EnrollCBC))
	mux.Handle("POST /api/v1/cbc/assessments", protect(h.RecordCompetencyAssessment))
	mux.Handle("POST /api/v1/exams/sheets", protect(h.CreateExamSheet))
	mux.Handle("PUT /api/v1/exams/sheets/{id}/marks/{student_id}", protect(h.SetExamMark))
	mux.Handle("POST /api/v1/exams/sheets/{id}/submit", protect(h.SubmitExamSheet))
	mux.Handle("GET /api/v1/exams/sheets/{id}/verification", protect(h.VerificationList))
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req passwordChangeRequest
	if !decode(w, r, &req) {
		return
	}
	result(w, h.Service.ChangePassword(actor, req.CurrentPassword, req.NewPassword), http.StatusOK, map[string]string{"status": "password_changed"})
}

type studentListItem struct {
	ID              string `json:"id"`
	AdmissionNumber string `json:"admission_number"`
	Name            string `json:"name"`
	ClassStream     string `json:"class_stream"`
}

func (h Handler) ListStudents(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	value, err := h.Service.ListStudents(actor)
	if err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	items := make([]studentListItem, 0, len(value))
	for _, student := range value {
		items = append(items, studentListItem{ID: student.ID, AdmissionNumber: student.AdmissionNumber, Name: student.Name, ClassStream: student.ClassStream})
	}
	_ = response.JSON(w, http.StatusOK, map[string]any{"students": items})
}
func (h Handler) ListStreams(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	value, err := h.Service.ListStreams(actor)
	result(w, err, http.StatusOK, map[string]any{"streams": value})
}
func (h Handler) ListTerms(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	value, err := h.Service.ListTerms(actor)
	result(w, err, http.StatusOK, map[string]any{"terms": value})
}

func (h Handler) actor(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	actor, ok := identity.Actor(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return domain.User{}, false
	}
	return actor, true
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON request")
		return false
	}
	return true
}

func result(w http.ResponseWriter, err error, successStatus int, body any) {
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	_ = response.JSON(w, successStatus, body)
}

func hasAnyRole(user domain.User, roles ...domain.Role) bool {
	for _, actual := range user.Roles {
		for _, required := range roles {
			if actual == required {
				return true
			}
		}
	}
	return false
}

type userRequest struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Phone    string        `json:"phone"`
	Email    string        `json:"email"`
	Roles    []domain.Role `json:"roles"`
	Password string        `json:"password"`
}

func (h Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	if !hasAnyRole(actor, domain.RolePrincipal, domain.RoleRegistrar) {
		response.Error(w, http.StatusForbidden, "registrar or principal required")
		return
	}
	var req userRequest
	if !decode(w, r, &req) {
		return
	}
	if len(req.Password) < 8 {
		response.Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	letter, digit := false, false
	for _, char := range req.Password {
		if char >= '0' && char <= '9' {
			digit = true
		}
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			letter = true
		}
	}
	if !letter || !digit {
		response.Error(w, http.StatusBadRequest, "password must contain a letter and a number")
		return
	}
	if !hasAnyRole(actor, domain.RolePrincipal) {
		for _, role := range req.Roles {
			if role == domain.RolePrincipal || role == domain.RoleBursar || role == domain.RoleDeputy || role == domain.RoleDOS || role == domain.RoleRegistrar {
				response.Error(w, http.StatusForbidden, "only a principal can assign privileged roles")
				return
			}
		}
	}
	u := domain.User{ID: strings.TrimSpace(req.ID), SchoolID: actor.SchoolID, Name: strings.TrimSpace(req.Name), Phone: strings.TrimSpace(req.Phone), Email: strings.TrimSpace(req.Email), Roles: req.Roles}
	if err := h.Service.AddUser(u); err != nil {
		result(w, err, http.StatusCreated, nil)
		return
	}
	if err := h.Service.SetPassword(actor, u.ID, req.Password); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	result(w, nil, http.StatusCreated, map[string]string{"id": u.ID})
}

func (h Handler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	if !hasAnyRole(actor, domain.RolePrincipal, domain.RoleRegistrar) {
		response.Error(w, http.StatusForbidden, "registrar or principal required")
		return
	}
	var value domain.Student
	if !decode(w, r, &value) {
		return
	}
	value.SchoolID = actor.SchoolID
	result(w, h.Service.AddStudent(value), http.StatusCreated, map[string]string{"id": value.ID})
}
func (h Handler) CreateStaff(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	if !hasAnyRole(actor, domain.RolePrincipal, domain.RoleRegistrar) {
		response.Error(w, http.StatusForbidden, "registrar or principal required")
		return
	}
	var value domain.Staff
	if !decode(w, r, &value) {
		return
	}
	value.SchoolID = actor.SchoolID
	result(w, h.Service.AddStaff(value), http.StatusCreated, map[string]string{"id": value.ID})
}
func (h Handler) CreateTerm(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var value domain.AcademicTerm
	if !decode(w, r, &value) {
		return
	}
	result(w, h.Service.CreateTerm(actor, value), http.StatusCreated, map[string]string{"id": value.ID})
}
func (h Handler) CreateStream(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var value domain.Stream
	if !decode(w, r, &value) {
		return
	}
	result(w, h.Service.CreateStream(actor, value), http.StatusCreated, map[string]string{"id": value.ID})
}

type attendanceRequest struct {
	Date         string                             `json:"date"`
	StreamID     string                             `json:"stream_id"`
	Subject      string                             `json:"subject"`
	Statuses     map[string]domain.AttendanceStatus `json:"students"`
	Reasons      map[string]string                  `json:"reasons"`
	Topic        string                             `json:"topic"`
	LessonStatus string                             `json:"status"`
	Reason       string                             `json:"reason"`
}
type prefectCountRequest struct {
	Date     string `json:"date"`
	StreamID string `json:"stream_id"`
	Present  int    `json:"present"`
	Absent   int    `json:"absent"`
}

func (h Handler) SubmitPrefectCount(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req prefectCountRequest
	if !decode(w, r, &req) {
		return
	}
	result(w, h.Service.SubmitPrefectCount(actor, req.Date, req.StreamID, req.Present, req.Absent), http.StatusCreated, map[string]string{"status": "submitted"})
}

func (h Handler) PostMorningRoll(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req attendanceRequest
	if !decode(w, r, &req) {
		return
	}
	result(w, h.Service.PostMorningRoll(actor, req.Date, req.StreamID, req.Statuses), http.StatusCreated, map[string]string{"status": "posted"})
}
func (h Handler) RecordLesson(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req attendanceRequest
	if !decode(w, r, &req) {
		return
	}
	delivery, err := h.Service.RecordLessonDelivery(actor, req.Date, req.StreamID, req.Subject, req.LessonStatus, req.Topic, req.Reason)
	result(w, err, http.StatusCreated, delivery)
}
func (h Handler) Reconcile(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req attendanceRequest
	if !decode(w, r, &req) {
		return
	}
	value, err := h.Service.Reconcile(actor, req.Date, req.StreamID, req.Reasons)
	result(w, err, http.StatusOK, value)
}
func (h Handler) AttendanceSummary(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	value, err := h.Service.AttendanceSummary(actor, r.URL.Query().Get("date"), r.URL.Query().Get("stream_id"))
	result(w, err, http.StatusOK, value)
}

type leaveRequest struct {
	StudentID string    `json:"student_id"`
	Reason    string    `json:"reason"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
}

func (h Handler) RequestLeave(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req leaveRequest
	if !decode(w, r, &req) {
		return
	}
	value, err := h.Service.RequestLeave(actor, req.StudentID, req.Reason, req.Start, req.End)
	result(w, err, http.StatusCreated, value)
}

type leaveDecision struct {
	Approve bool `json:"approve"`
}

func (h Handler) DecideLeave(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req leaveDecision
	if !decode(w, r, &req) {
		return
	}
	result(w, h.Service.DecideLeave(actor, r.PathValue("id"), req.Approve), http.StatusOK, map[string]string{"status": "decided"})
}

func (h Handler) EnrollCBC(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var value domain.CBCEnrollment
	if !decode(w, r, &value) {
		return
	}
	result(w, h.Service.EnrollCBC(actor, value), http.StatusCreated, map[string]string{"status": "enrolled"})
}
func (h Handler) RecordCompetencyAssessment(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var value domain.CompetencyAssessment
	if !decode(w, r, &value) {
		return
	}
	result(w, h.Service.RecordCompetencyAssessment(actor, value), http.StatusCreated, map[string]string{"status": "recorded"})
}

type examSheetRequest struct {
	ClassStream string `json:"class_stream"`
	Subject     string `json:"subject"`
}

func (h Handler) CreateExamSheet(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req examSheetRequest
	if !decode(w, r, &req) {
		return
	}
	value, err := h.Service.CreateExamSheet(actor, req.ClassStream, req.Subject)
	result(w, err, http.StatusCreated, value)
}

type markRequest struct {
	Score int `json:"score"`
}

func (h Handler) SetExamMark(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req markRequest
	if !decode(w, r, &req) {
		return
	}
	result(w, h.Service.SetExamMark(actor, r.PathValue("id"), r.PathValue("student_id"), req.Score), http.StatusOK, map[string]string{"status": "saved"})
}
func (h Handler) SubmitExamSheet(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	result(w, h.Service.SubmitExamSheet(actor, r.PathValue("id")), http.StatusOK, map[string]string{"status": "submitted"})
}
func (h Handler) VerificationList(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	value, err := h.Service.VerificationList(actor, r.PathValue("id"))
	result(w, err, http.StatusOK, value)
}
