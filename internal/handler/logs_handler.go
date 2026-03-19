package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"ecom-analytics-go/internal/models"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type LogsHandler struct {
	conn driver.Conn
}

func NewLogsHandler(conn driver.Conn) *LogsHandler {
	return &LogsHandler{conn: conn}
}

// System Audit Logs
func (h *LogsHandler) CreateSystemLog(w http.ResponseWriter, r *http.Request) {
	var logItem models.SystemAuditLog
	if err := json.NewDecoder(r.Body).Decode(&logItem); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logItem.ID = uint64(time.Now().UnixNano())
	if logItem.CreatedAt.IsZero() {
		logItem.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO audit_logs (
			id, user_id, username, email, action, target_entity, target_id, 
			details, ip_address, role, created_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`

	err := h.conn.Exec(r.Context(), query,
		logItem.ID, logItem.UserID, logItem.Username, logItem.Email, logItem.Action, 
		logItem.TargetEntity, logItem.TargetID, logItem.Details, logItem.IPAddress, 
		logItem.Role, logItem.CreatedAt,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(logItem)
}

func (h *LogsHandler) GetSystemLogs(w http.ResponseWriter, r *http.Request) {
	page, take, sortBy, sortDesc := getPaginationParams(r)
	offset := (page - 1) * take

	// Get total count
	var total uint64
	err := h.conn.QueryRow(r.Context(), "SELECT COUNT(*) FROM audit_logs").Scan(&total)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orderDir := "DESC"
	if !sortDesc {
		orderDir = "ASC"
	}

	// Validate sortBy or default to created_at
	orderCol := "created_at"
	if sortBy != "" {
		// Basic validation to prevent SQL injection (audit_logs columns only)
		allowed := map[string]bool{
			"id": true, "user_id": true, "username": true, "email": true,
			"action": true, "target_entity": true, "target_id": true,
			"ip_address": true, "role": true, "created_at": true,
		}
		if allowed[sortBy] {
			orderCol = sortBy
		}
	}

	query := fmt.Sprintf(`
		SELECT 
			id, user_id, username, email, action, target_entity, 
			target_id, details, ip_address, role, created_at 
		FROM audit_logs 
		ORDER BY %s %s 
		LIMIT %d OFFSET %d
	`, orderCol, orderDir, take, offset)

	rows, err := h.conn.Query(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []models.SystemAuditLog
	for rows.Next() {
		var l models.SystemAuditLog
		var detailsStr *string
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.Username, &l.Email, &l.Action, &l.TargetEntity, 
			&l.TargetID, &detailsStr, &l.IPAddress, &l.Role, &l.CreatedAt,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if detailsStr != nil {
			l.Details = json.RawMessage(*detailsStr)
		}
		logs = append(logs, l)
	}

	meta := &models.PageMeta{
		Page:            page,
		Take:            take,
		ItemCount:       int(total),
		PageCount:       int(math.Ceil(float64(total) / float64(take))),
		HasPreviousPage: page > 1,
		HasNextPage:     page < int(math.Ceil(float64(total)/float64(take))),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ServiceResponse{
		IsSuccess:  true,
		Data:       logs,
		Message:    "Audit logs retrieved successfully",
		Meta:       meta,
		StatusCode: http.StatusOK,
	})
}

// User Audit Logs
func (h *LogsHandler) CreateUserLog(w http.ResponseWriter, r *http.Request) {
	var logItem models.UserAuditLog
	if err := json.NewDecoder(r.Body).Decode(&logItem); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logItem.ID = uint64(time.Now().UnixNano())
	if logItem.CreatedAt.IsZero() {
		logItem.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO user_audit_logs (
			id, user_id, cart_id, cart_uuid, email, order_id, 
			target_entity, action, details, ip_address, created_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`

	err := h.conn.Exec(r.Context(), query,
		logItem.ID, logItem.UserID, logItem.CartID, logItem.CartUUID, logItem.Email, 
		logItem.OrderID, logItem.TargetEntity, logItem.Action, logItem.Details, 
		logItem.IPAddress, logItem.CreatedAt,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(logItem)
}

func (h *LogsHandler) GetUserLogs(w http.ResponseWriter, r *http.Request) {
	page, take, sortBy, sortDesc := getPaginationParams(r)
	offset := (page - 1) * take

	// Get total count
	var total uint64
	err := h.conn.QueryRow(r.Context(), "SELECT COUNT(*) FROM user_audit_logs").Scan(&total)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orderDir := "DESC"
	if !sortDesc {
		orderDir = "ASC"
	}

	orderCol := "created_at"
	if sortBy != "" {
		allowed := map[string]bool{
			"id": true, "user_id": true, "cart_id": true, "cart_uuid": true,
			"email": true, "order_id": true, "target_entity": true,
			"action": true, "ip_address": true, "created_at": true,
		}
		if allowed[sortBy] {
			orderCol = sortBy
		}
	}

	query := fmt.Sprintf(`
		SELECT 
			id, user_id, cart_id, cart_uuid, email, order_id, 
			target_entity, action, details, ip_address, created_at 
		FROM user_audit_logs 
		ORDER BY %s %s 
		LIMIT %d OFFSET %d
	`, orderCol, orderDir, take, offset)

	rows, err := h.conn.Query(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []models.UserAuditLog
	for rows.Next() {
		var l models.UserAuditLog
		var detailsStr *string
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.CartID, &l.CartUUID, &l.Email, &l.OrderID, 
			&l.TargetEntity, &l.Action, &detailsStr, &l.IPAddress, &l.CreatedAt,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if detailsStr != nil {
			l.Details = json.RawMessage(*detailsStr)
		}
		logs = append(logs, l)
	}

	meta := &models.PageMeta{
		Page:            page,
		Take:            take,
		ItemCount:       int(total),
		PageCount:       int(math.Ceil(float64(total) / float64(take))),
		HasPreviousPage: page > 1,
		HasNextPage:     page < int(math.Ceil(float64(total)/float64(take))),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ServiceResponse{
		IsSuccess:  true,
		Data:       logs,
		Message:    "User audit logs retrieved successfully",
		Meta:       meta,
		StatusCode: http.StatusOK,
	})
}

// Internal Helpers
func getPaginationParams(r *http.Request) (page int, take int, sortBy string, sortDesc bool) {
	pageStr := r.URL.Query().Get("page")
	takeStr := r.URL.Query().Get("take")
	sortBy = r.URL.Query().Get("sortBy")
	sortDescStr := r.URL.Query().Get("sortDesc")

	page = 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	take = 10
	if takeStr != "" {
		if t, err := strconv.Atoi(takeStr); err == nil && t > 0 {
			take = t
		}
	}

	sortDesc = true
	if sortDescStr != "" {
		if b, err := strconv.ParseBool(sortDescStr); err == nil {
			sortDesc = b
		}
	}

	return
}
