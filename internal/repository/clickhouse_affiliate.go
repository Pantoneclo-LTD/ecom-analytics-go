package repository

import (
	"context"
	"ecom-analytics-go/internal/models"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"time"
)

type ClickHouseRepository interface {
	TrackClick(ctx context.Context, click *models.AffiliateClick) error
	GetClickStats(ctx context.Context, affiliateID uint64) (uint64, uint64, error)
	GetBulkClickStats(ctx context.Context, affiliateIDs []uint64, startTime, endTime string) ([]map[string]interface{}, error)
	MarkConverted(ctx context.Context, clickID string, orderID uint64, commissionAmount float64) error
	GetClickInfo(ctx context.Context, sessionID string) (*models.AffiliateClick, error)
	LookupAffiliateByIP(ctx context.Context, ip, ua string, windowHours int) (*models.AffiliateClick, error)
}

type ClickHouseAffiliateRepository struct {
	conn driver.Conn
}

func NewClickHouseAffiliateRepository(conn driver.Conn) *ClickHouseAffiliateRepository {
	return &ClickHouseAffiliateRepository{conn: conn}
}

func (r *ClickHouseAffiliateRepository) TrackClick(ctx context.Context, click *models.AffiliateClick) error {
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

	return r.conn.Exec(ctx, query,
		click.ID, click.AffiliateID, click.AffiliateDiscountID, click.SessionID, click.IPAddress,
		click.UserAgent, click.ReferrerURL, click.LandingPage, click.Source, click.CountryID,
		click.City, click.DeviceType, click.Browser, click.OS, click.UTMSource,
		click.UTMMedium, click.UTMCampaign, click.UTMContent, click.UTMTerm, click.Converted,
		click.ConversionDate, click.OrderID, click.CommissionEarned, click.CreatedAt,
	)
}

func (r *ClickHouseAffiliateRepository) GetClickStats(ctx context.Context, affiliateID uint64) (uint64, uint64, error) {
	var totalClicks, uniqueVisitors uint64
	
	query := `
		SELECT 
			count(*) as total_clicks,
			uniq(ip_address) as unique_visitors
		FROM affiliate_clicks
		WHERE affiliate_id = ?
	`
	
	err := r.conn.QueryRow(ctx, query, affiliateID).Scan(&totalClicks, &uniqueVisitors)
	return totalClicks, uniqueVisitors, err
}

func (r *ClickHouseAffiliateRepository) GetBulkClickStats(ctx context.Context, affiliateIDs []uint64, startTime, endTime string) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			affiliate_id,
			count(*) as total_clicks,
			countIf(converted = 1) as total_conversions
		FROM affiliate_clicks
		WHERE affiliate_id IN (?) AND created_at BETWEEN ? AND ?
		GROUP BY affiliate_id
	`
	
	rows, err := r.conn.Query(ctx, query, affiliateIDs, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var affID uint64
		var clicks, conversions uint64
		if err := rows.Scan(&affID, &clicks, &conversions); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"affiliate_id": affID,
			"total_clicks": clicks,
			"total_conversions": conversions,
		})
	}
	return results, nil
}

func (r *ClickHouseAffiliateRepository) MarkConverted(ctx context.Context, clickID string, orderID uint64, commissionAmount float64) error {
	query := `
		ALTER TABLE affiliate_clicks UPDATE 
			converted = 1,
			conversion_date = now(),
			order_id = ?,
			commission_earned = ?
		WHERE id = ?
	`
	return r.conn.Exec(ctx, query, orderID, commissionAmount, clickID)
}

func (r *ClickHouseAffiliateRepository) GetClickInfo(ctx context.Context, sessionID string) (*models.AffiliateClick, error) {
	query := `
		SELECT id, affiliate_id, affiliate_discount_id, utm_source, utm_medium, utm_campaign
		FROM affiliate_clicks
		WHERE session_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	var click models.AffiliateClick
	err := r.conn.QueryRow(ctx, query, sessionID).Scan(
		&click.ID, &click.AffiliateID, &click.AffiliateDiscountID, 
		&click.UTMSource, &click.UTMMedium, &click.UTMCampaign,
	)
	if err != nil {
		return nil, err
	}
	return &click, nil
}

func (r *ClickHouseAffiliateRepository) LookupAffiliateByIP(ctx context.Context, ip, ua string, windowHours int) (*models.AffiliateClick, error) {
	query := fmt.Sprintf(`
		SELECT id, affiliate_id, affiliate_discount_id
		FROM affiliate_clicks
		WHERE ip_address = ? AND user_agent = ? AND converted = 0
		AND created_at >= subtractHours(now(), %d)
		ORDER BY created_at DESC
		LIMIT 1
	`, windowHours)
	
	var click models.AffiliateClick
	err := r.conn.QueryRow(ctx, query, ip, ua).Scan(&click.ID, &click.AffiliateID, &click.AffiliateDiscountID)
	if err != nil {
		return nil, err
	}
	return &click, nil
}
