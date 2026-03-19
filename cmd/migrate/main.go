package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"ecom-analytics-go/internal/db"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, relying on environment variables: %v", err)
	}

	table := flag.String("table", "all", "Table to migrate (audit_logs, user_audit_logs, affiliate_clicks, all)")
	limit := flag.Int("limit", 0, "Limit number of rows to migrate per table (0 for no limit)")
	batchSize := flag.Int("batch", 1000, "Batch size for migration")
	clean := flag.Bool("clean", false, "Clean ClickHouse tables before migration")
	flag.Parse()

	// Connect to ClickHouse
	chConn, err := db.Connect()
	if err != nil {
		log.Fatalf("ClickHouse connection failed: %v", err)
	}
	defer chConn.Close()

	// Connect to Postgres
	pgDB, err := db.ConnectPostgres()
	if err != nil {
		log.Fatalf("Postgres connection failed: %v", err)
	}
	defer pgDB.Close()

	ctx := context.Background()

	if *clean {
		cleanClickHouse(ctx, chConn, *table)
		if err := db.InitSchema(chConn); err != nil {
			log.Fatalf("Failed to recreate schema: %v", err)
		}
	}

	if *table == "all" || *table == "audit_logs" {
		migrateTable(ctx, pgDB, chConn, "audit_logs", *limit, *batchSize, flushAuditLogs)
	}

	if *table == "all" || *table == "user_audit_logs" {
		migrateTable(ctx, pgDB, chConn, "user_audit_logs", *limit, *batchSize, flushUserAuditLogs)
	}

	if *table == "all" || *table == "affiliate_clicks" {
		migrateTable(ctx, pgDB, chConn, "affiliate_clicks", *limit, *batchSize, flushAffiliateClicks)
	}

	log.Println("\nMigration process completed!")
}

type FlushFunc func(ctx context.Context, chConn driver.Conn, batch []map[string]interface{}) error

func cleanClickHouse(ctx context.Context, chConn driver.Conn, table string) {
	tables := []string{"audit_logs", "user_audit_logs", "affiliate_clicks"}
	if table != "all" {
		tables = []string{table}
	}

	for _, t := range tables {
		log.Printf("Cleaning ClickHouse table: %s", t)
		if err := chConn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t)); err != nil {
			log.Printf("Failed to clean table %s: %v", t, err)
		}
	}
}

func migrateTable(ctx context.Context, pgDB *sqlx.DB, chConn driver.Conn, tableName string, limit int, batchSize int, flush FlushFunc) {
	log.Printf("\n>>> Starting migration for: %s", tableName)
	
	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if limit > 0 {
		total = limit
	} else {
		err := pgDB.Get(&total, countQuery)
		if err != nil {
			log.Printf("Failed to get total count for %s: %v", tableName, err)
			return
		}
	}
	log.Printf("Total rows to migrate: %d", total)

	query := fmt.Sprintf("SELECT * FROM %s ORDER BY id ASC", tableName)
	if limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	rows, err := pgDB.Queryx(query)
	if err != nil {
		log.Printf("Failed to query %s: %v", tableName, err)
		return
	}
	defer rows.Close()

	batch := make([]map[string]interface{}, 0, batchSize)
	count := 0
	startTime := time.Now()

	for rows.Next() {
		dest := make(map[string]interface{})
		if err := rows.MapScan(dest); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		// Pre-process details (JSONB)
		// Safe mapping for details (prevent double-encoding)
		if details, ok := dest["details"]; ok && details != nil {
			switch v := details.(type) {
			case string:
				dest["details_str"] = v
			case []byte:
				dest["details_str"] = string(v)
			default:
				detailsJSON, _ := json.Marshal(details)
				dest["details_str"] = string(detailsJSON)
			}
		} else {
			dest["details_str"] = nil
		}

		batch = append(batch, dest)
		count++

		if len(batch) >= batchSize {
			if err := flush(ctx, chConn, batch); err != nil {
				log.Printf("Failed to flush batch: %v", err)
			}
			batch = batch[:0]
			
			// Progress logs
			elapsed := time.Since(startTime)
			rate := float64(count) / elapsed.Seconds()
			percent := float64(count) / float64(total) * 100
			eta := time.Duration(float64(total-count)/rate) * time.Second

			fmt.Printf("\rProgress [%s]: %d/%d (%.2f%%) | Rate: %.0f/s | Elapsed: %v | ETA: %v", 
				tableName, count, total, percent, rate, elapsed.Round(time.Second), eta.Round(time.Second))
		}
	}

	if len(batch) > 0 {
		if err := flush(ctx, chConn, batch); err != nil {
			log.Printf("Failed to flush final batch: %v", err)
		}
	}
	
	log.Printf("\nFinished: %s (%d rows in %v)", tableName, count, time.Since(startTime).Round(time.Second))
}

func flushAuditLogs(ctx context.Context, chConn driver.Conn, batch []map[string]interface{}) error {
	batchInsert, err := chConn.PrepareBatch(ctx, "INSERT INTO audit_logs")
	if err != nil {
		return err
	}
	for _, row := range batch {
		err := batchInsert.Append(
			getUint64(row["id"]),
			getUint64(row["user_id"]),
			getStringPtr(row["username"]),
			getString(row["email"]),
			getString(row["action"]),
			getStringPtr(row["target_entity"]),
			getUint64Ptr(row["target_id"]),
			getStringPtr(row["details_str"]),
			getStringPtr(row["ip_address"]),
			getString(row["role"]),
			getTime(row["created_at"]),
		)
		if err != nil { log.Printf("Err: %v", err) }
	}
	return batchInsert.Send()
}

func flushUserAuditLogs(ctx context.Context, chConn driver.Conn, batch []map[string]interface{}) error {
	batchInsert, err := chConn.PrepareBatch(ctx, "INSERT INTO user_audit_logs")
	if err != nil {
		return err
	}
	for _, row := range batch {
		err := batchInsert.Append(
			getUint64(row["id"]),
			getUint64Ptr(row["user_id"]),
			getUint64Ptr(row["cart_id"]),
			getStringPtr(row["cart_uuid"]),
			getStringPtr(row["email"]),
			getUint64Ptr(row["order_id"]),
			getStringPtr(row["target_entity"]),
			getString(row["action"]),
			getStringPtr(row["details_str"]),
			getStringPtr(row["ip_address"]),
			getTime(row["created_at"]),
		)
		if err != nil { log.Printf("Err: %v", err) }
	}
	return batchInsert.Send()
}

func flushAffiliateClicks(ctx context.Context, chConn driver.Conn, batch []map[string]interface{}) error {
	batchInsert, err := chConn.PrepareBatch(ctx, "INSERT INTO affiliate_clicks")
	if err != nil {
		return err
	}
	for _, row := range batch {
		commission := 0.0
		if comm, ok := row["commission_earned"]; ok && comm != nil {
			switch v := comm.(type) {
			case float64: commission = v
			case []uint8: fmt.Sscanf(string(v), "%f", &commission)
			}
		}
		err := batchInsert.Append(
			getUint64(row["id"]),
			getUint64(row["affiliate_id"]),
			getUint64Ptr(row["affiliate_discount_id"]),
			getString(row["session_id"]),
			getStringPtr(row["ip_address"]),
			getStringPtr(row["user_agent"]),
			getStringPtr(row["referrer_url"]),
			getStringPtr(row["landing_page"]),
			getString(row["source"]),
			getUint64Ptr(row["country_id"]),
			getStringPtr(row["city"]),
			getStringPtr(row["device_type"]),
			getStringPtr(row["browser"]),
			getStringPtr(row["os"]),
			getStringPtr(row["utm_source"]),
			getStringPtr(row["utm_medium"]),
			getStringPtr(row["utm_campaign"]),
			getStringPtr(row["utm_content"]),
			getStringPtr(row["utm_term"]),
			getBool(row["converted"]),
			getTimePtr(row["conversion_date"]),
			getUint64Ptr(row["order_id"]),
			commission,
			getTime(row["created_at"]),
		)
		if err != nil { log.Printf("Err: %v", err) }
	}
	return batchInsert.Send()
}

// Helpers
func getStringPtr(v interface{}) *string {
	if v == nil { return nil }
	if s, ok := v.(string); ok { return &s }
	if b, ok := v.([]uint8); ok { s := string(b); return &s }
	return nil
}
func getString(v interface{}) string {
	if v == nil { return "" }
	if s, ok := v.(string); ok { return s }
	if b, ok := v.([]uint8); ok { return string(b) }
	return ""
}
func getUint64(v interface{}) uint64 {
	if v == nil { return 0 }
	if i, ok := v.(int64); ok { return uint64(i) }
	if i, ok := v.(int); ok { return uint64(i) }
	return 0
}
func getUint64Ptr(v interface{}) *uint64 {
	if v == nil { return nil }
	if i, ok := v.(int64); ok { u := uint64(i); return &u }
	if i, ok := v.(int); ok { u := uint64(i); return &u }
	return nil
}
func getBool(v interface{}) bool {
	if v == nil { return false }
	if b, ok := v.(bool); ok { return b }
	return false
}
func getTime(v interface{}) time.Time {
	if v == nil { return time.Time{} }
	if t, ok := v.(time.Time); ok { return t }
	return time.Time{}
}
func getTimePtr(v interface{}) *time.Time {
	if v == nil { return nil }
	if t, ok := v.(time.Time); ok { return &t }
	return nil
}
