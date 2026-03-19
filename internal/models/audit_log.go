package models

import (
	"encoding/json"
	"time"
	"github.com/google/uuid"
)

type SystemAuditLog struct {
	ID           uuid.UUID       `json:"id" clickhouse:"id"`
	UserID       uint64          `json:"userId" clickhouse:"user_id"`
	Username     *string         `json:"username" clickhouse:"username"`
	Email        string          `json:"email" clickhouse:"email"`
	Action       string          `json:"action" clickhouse:"action"`
	TargetEntity *string         `json:"targetEntity" clickhouse:"target_entity"`
	TargetID     *uint64         `json:"targetId" clickhouse:"target_id"`
	Details      json.RawMessage `json:"details" clickhouse:"details"`
	IPAddress    *string         `json:"ipAddress" clickhouse:"ip_address"`
	Role         string          `json:"role" clickhouse:"role"`
	CreatedAt    time.Time       `json:"createdAt" clickhouse:"created_at"`
}

type UserAuditLog struct {
	ID           uuid.UUID       `json:"id" clickhouse:"id"`
	UserID       *uint64         `json:"userId" clickhouse:"user_id"`
	CartID       *uint64         `json:"cartId" clickhouse:"cart_id"`
	CartUUID     *string         `json:"cartUuid" clickhouse:"cart_uuid"`
	Email        *string         `json:"email" clickhouse:"email"`
	OrderID      *uint64         `json:"orderId" clickhouse:"order_id"`
	TargetEntity *string         `json:"targetEntity" clickhouse:"target_entity"`
	Action       string          `json:"action" clickhouse:"action"`
	Details      json.RawMessage `json:"details" clickhouse:"details"`
	IPAddress    *string         `json:"ipAddress" clickhouse:"ip_address"`
	CreatedAt    time.Time       `json:"createdAt" clickhouse:"created_at"`
}

type PageMeta struct {
	Page            int  `json:"page"`
	Take            int  `json:"take"`
	ItemCount       int  `json:"itemCount"`
	PageCount       int  `json:"pageCount"`
	HasPreviousPage bool `json:"hasPreviousPage"`
	HasNextPage     bool `json:"hasNextPage"`
}

type ServiceResponse struct {
	IsSuccess  bool        `json:"isSuccess"`
	Data       interface{} `json:"data"`
	Meta       *PageMeta   `json:"meta,omitempty"`
	Message    string      `json:"message"` // Match user's "messasge" (typo intended or should I fix it? Sample had "messasge" but also "message" in a comment. I'll use "message" as it's more standard)
	StatusCode int         `json:"statusCode"`
}
