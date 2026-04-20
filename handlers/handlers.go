// Package handlers wires HTTP requests to business logic.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"notification-service/logger"
	"notification-service/models"
	pb "notification-service/proto"
	"notification-service/services"
	"strings"
)

var log = logger.NewLogger("notification-service")

// Handler holds a reference to the service layer.
type Handler struct {
	service *services.Service
}

type GRPCServer struct {
	pb.UnimplementedNotificationServiceServer
	h *Handler
}

// NewGRPCServer returns a GRPCServer backed by the given handler
func NewGRPCServer(h *Handler) pb.NotificationServiceServer {
	return &GRPCServer{h: h}
}

// New returns a Handler backed by the given service.
func New(s *services.Service) *Handler {
	return &Handler{service: s}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeLog(msg string) {
	log.Info(msg)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	// log.Error(msg)
	writeJSON(w, status, map[string]string{"error": msg})
}

// determineErrorStatus checks the error message/type to correctly respond with HTTP status.
func determineErrorStatus(err error) int {
	if errors.Is(err, models.ErrTemplateNotFound) || errors.Is(err, models.ErrNotificationNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, models.ErrTemplateAlreadyExists) {
		return http.StatusConflict
	}
	msg := err.Error()
	if strings.Contains(msg, "required") || strings.Contains(msg, "invalid") || strings.Contains(msg, "placeholder") || strings.Contains(msg, "missing template variables") {
		// "missing template variables" is unprocessable entity historically but bad request is fine,
		// actually let's keep unprocessable entity for 'missing template variables'.
		if strings.Contains(msg, "missing template variables") {
			return http.StatusUnprocessableEntity
		}
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// ─── Template Handlers ───────────────────────────────────────────────────────

// CreateTemplateRequest represents the payload to create a template
type CreateTemplateRequest struct {
	ID      string `json:"template_id"`
	Content string `json:"content"`
}

// CreateTemplate handles POST /templates
// @Summary      Create a new template
// @Description  Create a template with the given content
// @Tags         templates
// @Accept       json
// @Produce      json
// @Param        request body CreateTemplateRequest true "Template request payload"
// @Success      201  {object} models.Template
// @Failure      400  {object} map[string]string
// @Failure      409  {object} map[string]string
// @Failure      500  {object} map[string]string
// @Router       /templates [post]
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req CreateTemplateRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.service.CreateTemplate(req.ID, req.Content)
	if err != nil {
		writeError(w, determineErrorStatus(err), err.Error())
		return
	}

	// response includes Variables slice transparently handled by service layer
	writeLog("Created template with ID: " + resp.ID + "Content is: " + resp.Content)
	writeJSON(w, http.StatusCreated, resp)
}

// ListTemplates handles GET /templates
// @Summary      List templates
// @Description  Get a list of all templates
// @Tags         templates
// @Produce      json
// @Success      200  {array} models.Template
// @Router       /templates [get]
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	writeLog("Listing all templates")
	writeJSON(w, http.StatusOK, h.service.ListTemplates())
}

// GetTemplate handles GET /templates/{id}
// @Summary      Get a template
// @Description  Get a single template by its ID
// @Tags         templates
// @Produce      json
// @Param        id   path      string  true  "Template ID"
// @Success      200  {object}  models.Template
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /templates/{id} [get]
func (h *Handler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	t, err := h.service.GetTemplate(id)
	if err != nil {
		writeError(w, determineErrorStatus(err), err.Error())
		return
	}
	writeLog("Getting Template By ID: " + id)
	writeJSON(w, http.StatusOK, t)
}

// UpdateTemplateRequest represents the payload to update a template
type UpdateTemplateRequest struct {
	Content string `json:"content"`
}

// UpdateTemplate handles PUT /templates/{id}
// @Summary      Update a template
// @Description  Update an existing template by its ID
// @Tags         templates
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Template ID"
// @Param        request body UpdateTemplateRequest true "Template update payload"
// @Success      200  {object}  models.Template
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /templates/{id} [put]
func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "template id is required in path")
		return
	}

	var req UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.service.UpdateTemplate(id, req.Content)
	if err != nil {
		writeError(w, determineErrorStatus(err), err.Error())
		return
	}
	writeLog("Updating Template With Id:" + resp.ID + "Content is: " + resp.Content)
	writeJSON(w, http.StatusOK, resp)
}

// DeleteTemplate handles DELETE /templates/{id}
// @Summary      Delete a template
// @Description  Delete a template by its ID
// @Tags         templates
// @Produce      json
// @Param        id   path      string  true  "Template ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /templates/{id} [delete]
func (h *Handler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "template id is required in path")
		return
	}

	if err := h.service.DeleteTemplate(id); err != nil {
		writeError(w, determineErrorStatus(err), err.Error())
		return
	}
	writeLog("Deleted Template With Id:" + id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "template deleted"})
}

// ─── Notification Handlers ───────────────────────────────────────────────────

// CreateNotificationRequest represents the payload to create a notification
type CreateNotificationRequest struct {
	UserID     string                  `json:"user_id"`
	Targets    map[string]string       `json:"targets"`
	TemplateID string                  `json:"template_id"`
	Data       map[string]string       `json:"data"`
	Priority   models.Priority         `json:"priority"`
	Type       models.NotificationType `json:"type"`
}

// CreateNotification handles POST /notifications
// @Summary      Create a notification
// @Description  Create a notification using a template
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        request body CreateNotificationRequest true "Notification request payload"
// @Success      201  {object}  models.Notification
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      422  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /notifications [post]

func (h *Handler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var req CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	n, err := h.service.CreateNotification(
		req.UserID,
		req.Targets,
		req.TemplateID,
		req.Data,
		req.Priority,
		req.Type,
	)

	if err != nil {
		writeError(w, determineErrorStatus(err), err.Error())
		return
	}
	writeLog("Created notification with ID: " + n.ID + " with TemplateID: " + n.TemplateID + "Content is: " + n.Message)
	writeJSON(w, http.StatusCreated, n)
}

func (h *Handler) SendJobNotification(req *pb.JobEvent) {
	log := logger.NewLogger("job-queue-notification")

	data := map[string]string{
		"job_id":    req.JobId,
		"stage":     req.Stage,
		"message":   req.Message,
		"file_name": req.FileName,
		"job_type":  req.Jobtype,
	}

	_, err := h.service.CreateNotification(
		"system_user",
		map[string]string{"email": "sahilkanani8320@gmail.com"},
		"job_notification_template",
		data,
		models.PriorityHigh,
		models.TypeInfo,
	)

	if err != nil {
		log.Error("Failed to send job notification: " + err.Error())
	} else {
		log.Info("Job notification sent successfully for job ID: " + req.JobId)
	}
}

func (s *GRPCServer) StreamEvents(stream pb.NotificationService_StreamEventsServer) error {

	var log = logger.NewLogger("job-queue")
	for {
		req, err := stream.Recv()

		if err == io.EOF {
			return stream.SendAndClose(&pb.Ack{
				Status: "done",
			})
		}

		if err != nil {
			return err
		}

		// :fire: NEW: check level
		if req.Jobtype == "WARNING" {
			var errLog = logger.NewLogger("job-queue-error")
			errLog.Error(req.Message + " with retry count: " + string(req.RetryCount) + " and file name: " + req.FileName)
		} else {
			log.Info("Received job: " + req.Message + " with retry count: " + string(req.RetryCount) + " and file name: " + req.FileName)
		}

		if req.Stage == "COMPLETED" {
			log.Info("Job completed: " + req.Message)
			s.h.SendJobNotification(req)
		} else if req.Stage == "FAILED" {
			log.Error("Job failed: " + req.Message)
		} else {

		}
	}
}

// GetUserNotifications handles GET /notifications/user/{user_id}
// @Summary      Get user notifications
// @Description  Get a list of notifications for a specific user
// @Tags         notifications
// @Produce      json
// @Param        user_id   path      string  true  "User ID"
// @Success      200  {array}   models.Notification
// @Failure      400  {object}  map[string]string
// @Router       /notifications/user/{user_id} [get]
func (h *Handler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	notifications, err := h.service.GetUserNotifications(userID)
	if err != nil {
		writeError(w, determineErrorStatus(err), err.Error())
		return
	}
	writeLog("Getting notifications for user: " + userID)
	writeJSON(w, http.StatusOK, notifications)
}

// MarkAsRead handles PATCH /notifications/{id}/read
// @Summary      Mark a notification as read
// @Description  Mark a specific notification as read by its ID
// @Tags         notifications
// @Produce      json
// @Param        id   path      string  true  "Notification ID"
// @Success      200  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /notifications/{id}/read [patch]
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.service.MarkNotificationRead(id); err != nil {
		writeError(w, determineErrorStatus(err), err.Error())
		return
	}
	writeLog("Marked notification as read with ID: " + id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "notification marked as read"})
}
