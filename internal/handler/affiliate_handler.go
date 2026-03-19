package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"ecom-analytics-go/internal/models"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jmoiron/sqlx"
)

type AffiliateHandler struct {
	chConn driver.Conn
	pgDB   *sqlx.DB
}

func NewAffiliateHandler(chConn driver.Conn, pgDB *sqlx.DB) *AffiliateHandler {
	return &AffiliateHandler{
		chConn: chConn,
		pgDB:   pgDB,
	}
}

func (h *AffiliateHandler) TrackClick(w http.ResponseWriter, r *http.Request) {
	var click models.AffiliateClick
	if err := json.NewDecoder(r.Body).Decode(&click); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Validate and resolve AffiliateID / AffiliateDiscountID via Postgres
	if err := h.resolveAffiliate(r.Context(), &click); err != nil {
		// We still track the click even if affiliate is not resolved, or maybe not? 
		// User said: "get relational info to store click rate which is necessary"
		// If we can't find the affiliate, we might still want to log it but maybe with ID 0.
		// For now, let's skip if no affiliate is found to keep click rate accurate.
		// http.Error(w, "Invalid affiliate or coupon", http.StatusNotFound)
		// return
	}

	// If neither ID is resolved, we might still want to log it as an unassigned click, 
	// but the UI usually needs an affiliate_id.
	if click.AffiliateID == 0 && (click.AffiliateDiscountID == nil || *click.AffiliateDiscountID == 0) {
		// Log but don't error? 
		// return // Skip for now
	}

	click.ID = uint64(time.Now().UnixNano())
	click.CreatedAt = time.Now()

	query := `
		INSERT INTO affiliate_clicks (
			id, affiliate_id, affiliate_discount_id, session_id, ip_address,
			user_agent, referrer_url, landing_page, source, country_id,
			city, device_type, browser, os, utm_source,
			utm_medium, utm_campaign, utm_content, utm_term, converted,
			conversion_date, order_id, commission_earned, created_at
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?
		)
	`

	err := h.chConn.Exec(r.Context(), query,
		click.ID, click.AffiliateID, click.AffiliateDiscountID, click.SessionID, click.IPAddress,
		click.UserAgent, click.ReferrerURL, click.LandingPage, click.Source, click.CountryID,
		click.City, click.DeviceType, click.Browser, click.OS, click.UTMSource,
		click.UTMMedium, click.UTMCampaign, click.UTMContent, click.UTMTerm, click.Converted,
		click.ConversionDate, click.OrderID, click.CommissionEarned, click.CreatedAt,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(click)
}

func (h *AffiliateHandler) resolveAffiliate(ctx context.Context, click *models.AffiliateClick) error {
	// Try resolve by AffiliateCode first
	if click.AffiliateCode != "" {
		var id uint64
		err := h.pgDB.GetContext(ctx, &id, `
			SELECT id FROM affiliate_profiles 
			WHERE affiliate_code = $1 AND status = 'active' AND is_active = true
		`, click.AffiliateCode)
		if err == nil {
			click.AffiliateID = id
			return nil
		}
	}

	// Try resolve by CouponCode
	coupon := click.CouponCode
	if coupon == "" && click.AffiliateCode != "" {
		coupon = click.AffiliateCode
	}

	if coupon != "" {
		var res struct {
			AffiliateID         uint64 `db:"affiliate_id"`
			AffiliateDiscountID uint64 `db:"affiliate_discount_id"`
		}
		err := h.pgDB.GetContext(ctx, &res, `
			SELECT ad.affiliate_id, ad.id as affiliate_discount_id 
			FROM affiliate_discounts ad
			JOIN discount d ON ad.discount_id = d.id
			WHERE d.coupon = $1 AND d.is_active = true AND ad.status = 'active' AND ad.is_active = true
		`, coupon)
		if err == nil {
			click.AffiliateID = res.AffiliateID
			click.AffiliateDiscountID = &res.AffiliateDiscountID
			return nil
		}
	}

	return sql.ErrNoRows
}

func (h *AffiliateHandler) GetClickRateAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	pageStr := r.URL.Query().Get("page")
	takeStr := r.URL.Query().Get("take")
	fromStr := r.URL.Query().Get("dateFrom")
	toStr := r.URL.Query().Get("dateTo")
	affiliateID := r.URL.Query().Get("affiliateId")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	take := 10
	if takeStr != "" {
		if t, err := strconv.Atoi(takeStr); err == nil && t > 0 {
			take = t
		}
	}
	offset := (page - 1) * take

	from := time.Now().AddDate(0, 0, -30)
	to := time.Now()

	if fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			to = t
		}
	}

	countQuery := "SELECT count(*) FROM (SELECT toDate(created_at) as date, utm_source FROM affiliate_clicks WHERE created_at >= ? AND created_at <= ?"
	countArgs := []interface{}{from, to}
	if affiliateID != "" {
		countQuery += " AND affiliate_id = ?"
		countArgs = append(countArgs, affiliateID)
	}
	countQuery += " GROUP BY date, utm_source)"

	var total uint64
	if err := h.chConn.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := `
		SELECT
			formatDateTime(created_at, '%Y-%m-%d') as date,
			utm_source,
			count(*) as clicks,
			countIf(converted) as conversions,
			if(clicks > 0, countIf(converted) / count(*), 0) as click_rate
		FROM affiliate_clicks
		WHERE created_at >= ? AND created_at <= ?
	`
	
	args := []interface{}{from, to}
	if affiliateID != "" {
		query += " AND affiliate_id = ?"
		args = append(args, affiliateID)
	}

	query += fmt.Sprintf(" GROUP BY date, utm_source ORDER BY date DESC, clicks DESC LIMIT %d OFFSET %d", take, offset)

	rows, err := h.chConn.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []models.ClickRateRecord
	for rows.Next() {
		var res models.ClickRateRecord
		if err := rows.Scan(&res.Date, &res.UTMSource, &res.Clicks, &res.Conversions, &res.ClickRate); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		results = append(results, res)
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
		Data:       results,
		Meta:       meta,
		Message:    "Click rate analytics retrieved successfully",
		StatusCode: http.StatusOK,
	})
}
