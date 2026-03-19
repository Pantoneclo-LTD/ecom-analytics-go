package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"ecom-analytics-go/internal/db"
	"ecom-analytics-go/internal/handler"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, relying on environment variables: %v", err)
	}

	// ClickHouse
	chConn, err := db.Connect()
	if err != nil {
		log.Fatalf("ClickHouse connection failed: %v", err)
	}
	defer chConn.Close()

	if err := db.InitSchema(chConn); err != nil {
		log.Fatalf("ClickHouse schema initialization failed: %v", err)
	}

	// Postgres
	pgDB, err := db.ConnectPostgres()
	if err != nil {
		log.Fatalf("Postgres connection failed: %v", err)
	}
	defer pgDB.Close()

	logsHandler := handler.NewLogsHandler(chConn)
	affiliateHandler := handler.NewAffiliateHandler(chConn, pgDB)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		// System Audit Logs (matches NestJS @Controller('audit-logs'))
		r.Post("/logs", logsHandler.CreateSystemLog)
		r.Get("/logs/findAllDataByPagination", logsHandler.GetSystemLogs)

		// User Audit Logs (matches NestJS @Post('web/createUserAuditLog'))
		r.Post("/logs/web/createUserAuditLog", logsHandler.CreateUserLog)
		r.Get("/logs/user/web/findAllDataByPagination", logsHandler.GetUserLogs)

		// Affiliate Tracking
		r.Post("/affiliate/track-click", affiliateHandler.TrackClick)
		r.Get("/affiliate/click-rate", affiliateHandler.GetClickRateAnalytics)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Starting analytics server on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
