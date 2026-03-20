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
	"ecom-analytics-go/internal/repository"
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

	// Repositories
	pgRepo := repository.NewPostgresAffiliateRepository(pgDB)
	chRepo := repository.NewClickHouseAffiliateRepository(chConn)

	logsHandler := handler.NewLogsHandler(chConn)
	affiliateHandler := handler.NewAffiliateHandler(pgRepo, chRepo)

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

		// Affiliate Tracking & Analytics
		r.Post("/affiliate/track-click", affiliateHandler.TrackClick)
		r.Get("/affiliate/click-rate", affiliateHandler.GetClickRateAnalytics)
		r.Get("/affiliate/stats", affiliateHandler.GetAffiliateStats)
		r.Post("/affiliate/bulk-stats", affiliateHandler.GetBulkAffiliateStats)
		r.Get("/affiliate/click-info", affiliateHandler.GetClickInfo)
		r.Get("/affiliate/lookup", affiliateHandler.LookupAffiliateByIP)
		r.Post("/affiliate/convert", affiliateHandler.TrackConversion)
		r.Post("/affiliate/track-conversion", affiliateHandler.TrackConversion)
		r.Post("/affiliate/generate-tracking-url", affiliateHandler.GenerateTrackingURL)
		r.Get("/affiliate/time-based-stats", affiliateHandler.GetTimeBasedStats)
		r.Post("/affiliate/analytics/clicks", affiliateHandler.GetComplexAnalytics)

		// Affiliate Profiles CRUD
		r.Post("/affiliate/profiles", affiliateHandler.CreateAffiliate)
		r.Get("/affiliate/profiles", affiliateHandler.ListAffiliates)
		r.Get("/affiliate/profiles/{id}", affiliateHandler.GetAffiliate)
		r.Get("/affiliate/profiles/code/{code}", affiliateHandler.GetAffiliateByCode)
		r.Get("/affiliate/profiles/user/{userId}", affiliateHandler.GetAffiliateByUserID)
		r.Put("/affiliate/profiles/{id}", affiliateHandler.UpdateAffiliate)
		r.Delete("/affiliate/profiles/{id}", affiliateHandler.DeleteProfile)

		// Affiliate Applications
		r.Get("/affiliate/applications", affiliateHandler.ListApplications)
		r.Post("/affiliate/applications/{id}/approve", affiliateHandler.ApproveApplication)
		r.Post("/affiliate/applications/{id}/reject", affiliateHandler.RejectApplication)

		// Affiliate Commissions
		r.Get("/affiliate/commissions", affiliateHandler.ListCommissions)
		r.Post("/affiliate/commissions/{id}/approve", affiliateHandler.ApproveCommission)
		r.Post("/affiliate/commissions/{id}/reject", affiliateHandler.RejectCommission)
		r.Post("/affiliate/commissions/{id}/pay", affiliateHandler.MarkCommissionPaid)
		r.Get("/affiliate/commissions/preview", affiliateHandler.GetCartCommissionPreview)
		r.Post("/affiliate/commissions/calculate", affiliateHandler.CalculateOrderCommission)

		// Affiliate Discounts
		r.Get("/affiliate/discounts", affiliateHandler.ListDiscountsForAffiliate)
		r.Post("/affiliate/discounts", affiliateHandler.AssignDiscount)
		r.Put("/affiliate/discounts/{id}", affiliateHandler.UpdateDiscountAssignment)
		r.Delete("/affiliate/discounts/{id}", affiliateHandler.RemoveDiscountFromAffiliate)

		// Reports
		r.Get("/affiliate/reports/performance", affiliateHandler.GetPerformanceReports)
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
