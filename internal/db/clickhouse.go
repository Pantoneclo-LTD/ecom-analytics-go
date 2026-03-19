package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func Connect() (driver.Conn, error) {
	addr := os.Getenv("CLICKHOUSE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:9000"
	}
	dbName := os.Getenv("CLICKHOUSE_DATABASE")
	if dbName == "" {
		dbName = "default"
	}
	user := os.Getenv("CLICKHOUSE_USER")
	if user == "" {
		user = "default"
	}
	password := os.Getenv("CLICKHOUSE_PASSWORD")

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: dbName,
			Username: user,
			Password: password,
		},
		DialTimeout:     time.Second * 30,
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(context.Background()); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}
	return conn, nil
}

func InitSchema(conn driver.Conn) error {
	ctx := context.Background()

	// System Audit Logs
	query1 := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID,
		user_id UInt64,
		username Nullable(String),
		email String,
		action String,
		target_entity Nullable(String),
		target_id Nullable(UInt64),
		details Nullable(String),
		ip_address Nullable(String),
		role String,
		created_at DateTime64(3, 'UTC')
	) ENGINE = MergeTree()
	ORDER BY (created_at, action)
	`
	if err := conn.Exec(ctx, query1); err != nil {
		return err
	}

	// User Audit Logs
	query2 := `
	CREATE TABLE IF NOT EXISTS user_audit_logs (
		id UUID,
		user_id Nullable(UInt64),
		cart_id Nullable(UInt64),
		cart_uuid Nullable(String),
		email Nullable(String),
		order_id Nullable(UInt64),
		target_entity Nullable(String),
		action String,
		details Nullable(String),
		ip_address Nullable(String),
		created_at DateTime64(3, 'UTC')
	) ENGINE = MergeTree()
	ORDER BY (created_at, action)
	`
	if err := conn.Exec(ctx, query2); err != nil {
		return err
	}

	// Affiliate Clicks
	query3 := `
	CREATE TABLE IF NOT EXISTS affiliate_clicks (
		id UUID,
		affiliate_id UInt64,
		affiliate_discount_id Nullable(UInt64),
		session_id String,
		ip_address Nullable(String),
		user_agent Nullable(String),
		referrer_url Nullable(String),
		landing_page Nullable(String),
		source String,
		country_id Nullable(UInt64),
		city Nullable(String),
		device_type Nullable(String),
		browser Nullable(String),
		os Nullable(String),
		utm_source Nullable(String),
		utm_medium Nullable(String),
		utm_campaign Nullable(String),
		utm_content Nullable(String),
		utm_term Nullable(String),
		converted Boolean,
		conversion_date Nullable(DateTime64(3, 'UTC')),
		order_id Nullable(UInt64),
		commission_earned Float64,
		created_at DateTime64(3, 'UTC')
	) ENGINE = MergeTree()
	ORDER BY created_at`
	if err := conn.Exec(ctx, query3); err != nil {
		return err
	}

	return nil
}
