package repository

import (
	"context"
	"database/sql"
	"ecom-analytics-go/internal/models"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
)

type PostgresAffiliateRepository struct {
	db *sqlx.DB
}

func NewPostgresAffiliateRepository(db *sqlx.DB) *PostgresAffiliateRepository {
	return &PostgresAffiliateRepository{db: db}
}

// GetProfileByID retrieves an affiliate profile by its ID
func (r *PostgresAffiliateRepository) GetProfileByID(ctx context.Context, id uint64) (*models.AffiliateProfile, error) {
	var profile models.AffiliateProfile
	query := `
		SELECT p.*, u.first_name as "user.firstName", u.last_name as "user.lastName", u.email as "user.email", u.phone as "user.phone"
		FROM affiliate_profiles p
		LEFT JOIN users u ON p.user_id = u.id
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &profile, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

// GetProfileByCode retrieves an affiliate profile by its affiliate code
func (r *PostgresAffiliateRepository) GetProfileByCode(ctx context.Context, code string) (*models.AffiliateProfile, error) {
	var profile models.AffiliateProfile
	query := `
		SELECT p.*, u.first_name as "user.firstName", u.last_name as "user.lastName", u.email as "user.email", u.phone as "user.phone"
		FROM affiliate_profiles p
		LEFT JOIN users u ON p.user_id = u.id
		WHERE p.affiliate_code = $1 AND p.deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &profile, query, code)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

// GetProfileByDiscountCode retrieves an affiliate profile and the specific affiliate_discount_id by a discount code
func (r *PostgresAffiliateRepository) GetProfileByDiscountCode(ctx context.Context, code string) (*models.AffiliateProfile, *uint64, error) {
	var result struct {
		models.AffiliateProfile
		AffiliateDiscountID uint64 `db:"affiliate_discount_id"`
	}

	query := `
		SELECT p.*, ad.id as affiliate_discount_id, 
		       u.first_name as "user.firstName", u.last_name as "user.lastName", 
		       u.email as "user.email", u.phone as "user.phone"
		FROM affiliate_profiles p
		JOIN affiliate_discounts ad ON p.id = ad.affiliate_id
		JOIN discounts d ON ad.discount_id = d.id
		LEFT JOIN users u ON p.user_id = u.id
		WHERE d.coupon = $1 AND p.deleted_at IS NULL AND ad.deleted_at IS NULL AND d.is_active = true
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &result, query, code)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	return &result.AffiliateProfile, &result.AffiliateDiscountID, nil
}

// GetProfileByUserID retrieves an affiliate profile by user ID
func (r *PostgresAffiliateRepository) GetProfileByUserID(ctx context.Context, userID uint64) (*models.AffiliateProfile, error) {
	var profile models.AffiliateProfile
	query := `SELECT * FROM affiliate_profiles WHERE user_id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &profile, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

// ListProfiles lists affiliate profiles with filtering and pagination
func (r *PostgresAffiliateRepository) ListProfiles(ctx context.Context, filters map[string]interface{}, page, limit int) ([]models.AffiliateProfile, uint64, error) {
	var profiles []models.AffiliateProfile
	var total uint64

	where := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if status, ok := filters["status"].(string); ok && status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		where = append(where, fmt.Sprintf("(affiliate_code ILIKE $%d OR bio ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")
	
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM affiliate_profiles WHERE %s", whereClause)
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query := fmt.Sprintf(`
		SELECT * FROM affiliate_profiles 
		WHERE %s 
		ORDER BY created_at DESC 
		LIMIT %d OFFSET %d
	`, whereClause, limit, offset)

	err = r.db.SelectContext(ctx, &profiles, query, args...)
	return profiles, total, err
}

// CreateProfile creates a new affiliate profile
func (r *PostgresAffiliateRepository) CreateProfile(ctx context.Context, profile *models.AffiliateProfile) error {
	query := `
		INSERT INTO affiliate_profiles (
			user_id, affiliate_code, country_id, status, commission_type, 
			commission_rate, payment_method, payment_details, bio, website_url, 
			social_media, created_at, updated_at
		) VALUES (
			:user_id, :affiliate_code, :country_id, :status, :commission_type, 
			:commission_rate, :payment_method, :payment_details, :bio, :website_url, 
			:social_media, NOW(), NOW()
		) RETURNING id
	`
	rows, err := r.db.NamedQueryContext(ctx, query, profile)
	if err != nil {
		return err
	}
	if rows.Next() {
		rows.Scan(&profile.ID)
	}
	return rows.Close()
}

// UpdateProfile updates an existing affiliate profile
func (r *PostgresAffiliateRepository) UpdateProfile(ctx context.Context, id uint64, data map[string]interface{}) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for k, v := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, argIdx))
		args = append(args, v)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE affiliate_profiles SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// SoftDeleteProfile soft deletes an affiliate profile
func (r *PostgresAffiliateRepository) SoftDeleteProfile(ctx context.Context, id uint64) error {
	query := `UPDATE affiliate_profiles SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetDiscountByID retrieves an affiliate discount by its ID
func (r *PostgresAffiliateRepository) GetDiscountByID(ctx context.Context, id uint64) (*models.AffiliateDiscount, error) {
	var discount models.AffiliateDiscount
	query := `SELECT * FROM affiliate_discounts WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &discount, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &discount, err
}

// ListDiscounts lists all active discounts for an affiliate
func (r *PostgresAffiliateRepository) ListDiscounts(ctx context.Context, affiliateID uint64) ([]models.AffiliateDiscount, error) {
	var discounts []models.AffiliateDiscount
	query := `SELECT * FROM affiliate_discounts WHERE affiliate_id = $1 AND deleted_at IS NULL`
	err := r.db.SelectContext(ctx, &discounts, query, affiliateID)
	return discounts, err
}

// CreateDiscount creates a new affiliate discount mapping
func (r *PostgresAffiliateRepository) CreateDiscount(ctx context.Context, discount *models.AffiliateDiscount) error {
	query := `
		INSERT INTO affiliate_discounts (
			affiliate_id, discount_id, status, max_usage, commission_type, 
			commission_rate, start_date, end_date, notes, created_at, updated_at
		) VALUES (
			:affiliate_id, :discount_id, :status, :max_usage, :commission_type, 
			:commission_rate, :start_date, :end_date, :notes, NOW(), NOW()
		) RETURNING id
	`
	rows, err := r.db.NamedQueryContext(ctx, query, discount)
	if err != nil {
		return err
	}
	if rows.Next() {
		rows.Scan(&discount.ID)
	}
	return rows.Close()
}

// UpdateDiscount updates an existing affiliate discount mapping
func (r *PostgresAffiliateRepository) UpdateDiscount(ctx context.Context, id uint64, data map[string]interface{}) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for k, v := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, argIdx))
		args = append(args, v)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE affiliate_discounts SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// SoftDeleteDiscount soft deletes an affiliate discount mapping
func (r *PostgresAffiliateRepository) SoftDeleteDiscount(ctx context.Context, id uint64) error {
	query := `UPDATE affiliate_discounts SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// CreateCommission records a new affiliate commission
func (r *PostgresAffiliateRepository) CreateCommission(ctx context.Context, commission *models.AffiliateCommission) error {
	query := `
		INSERT INTO affiliate_commissions (
			affiliate_id, affiliate_discount_id, order_id, discount_id, click_id, 
			source, status, order_amount, commission_rate, commission_amount, 
			currency_code, customer_email, customer_phone, coupon_code, created_at, updated_at
		) VALUES (
			:affiliate_id, :affiliate_discount_id, :order_id, :discount_id, :click_id, 
			:source, :status, :order_amount, :commission_rate, :commission_amount, 
			:currency_code, :customer_email, :customer_phone, :coupon_code, NOW(), NOW()
		) RETURNING id
	`
	rows, err := r.db.NamedQueryContext(ctx, query, commission)
	if err != nil {
		return err
	}
	if rows.Next() {
		rows.Scan(&commission.ID)
	}
	return rows.Close()
}

// CreateConversion records a new affiliate conversion
func (r *PostgresAffiliateRepository) CreateConversion(ctx context.Context, conversion *models.AffiliateConversion) error {
	query := `
		INSERT INTO affiliate_conversions (
			affiliate_id, affiliate_discount_id, click_id, order_id, type, 
			conversion_value, customer_email, session_id, ip_address, created_at, updated_at
		) VALUES (
			:affiliate_id, :affiliate_discount_id, :click_id, :order_id, :type, 
			:conversion_value, :customer_email, :session_id, :ip_address, NOW(), NOW()
		) RETURNING id
	`
	rows, err := r.db.NamedQueryContext(ctx, query, conversion)
	if err != nil {
		return err
	}
	if rows.Next() {
		rows.Scan(&conversion.ID)
	}
	return rows.Close()
}

// CreateDiscountUsage records a new affiliate discount usage
func (r *PostgresAffiliateRepository) CreateDiscountUsage(ctx context.Context, usage *models.AffiliateDiscountUsage) error {
	query := `
		INSERT INTO affiliate_discount_usages (
			affiliate_discount_id, discount_usage_id, order_id, customer_email, 
			order_amount, discount_amount, commission_rate, commission_amount, 
			session_id, ip_address, created_at, updated_at, affiliate_id
		) VALUES (
			:affiliate_discount_id, :discount_usage_id, :order_id, :customer_email, 
			:order_amount, :discount_amount, :commission_rate, :commission_amount, 
			:session_id, :ip_address, NOW(), NOW(), :affiliate_id
		) RETURNING id
	`
	rows, err := r.db.NamedQueryContext(ctx, query, usage)
	if err != nil {
		return err
	}
	if rows.Next() {
		rows.Scan(&usage.ID)
	}
	return rows.Close()
}

// GetCommissionStats retrieves aggregated commission stats for an affiliate from PostgreSQL
func (r *PostgresAffiliateRepository) GetCommissionStats(ctx context.Context, affiliateID uint64) (*models.AffiliateStats, error) {
	var stats models.AffiliateStats
	query := `
		SELECT 
			affiliate_id,
			COUNT(id) as total_conversions,
			SUM(commission_amount) as total_earnings,
			SUM(order_amount) as total_revenue
		FROM affiliate_commissions
		WHERE affiliate_id = $1 AND status != 'cancelled'
		GROUP BY affiliate_id
	`
	err := r.db.GetContext(ctx, &stats, query, affiliateID)
	if err == sql.ErrNoRows {
		return &models.AffiliateStats{AffiliateID: affiliateID}, nil
	}
	return &stats, err
}

// GetTimeBasedStats retrieves time-based analytics data from PostgreSQL
func (r *PostgresAffiliateRepository) GetTimeBasedStats(ctx context.Context, affiliateID uint64, startTime, endTime string) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			DATE(created_at) as date,
			COUNT(id) as usages,
			SUM(order_amount) as revenue,
			SUM(commission_amount) as commission
		FROM affiliate_discount_usages
		WHERE affiliate_id = $1 AND created_at BETWEEN $2 AND $3
		GROUP BY DATE(created_at)
		ORDER BY DATE(created_at) ASC
	`
	rows, err := r.db.QueryxContext(ctx, query, affiliateID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		result := make(map[string]interface{})
		err := rows.MapScan(result)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// ListApplications lists affiliate applications (profiles with pending status)
func (r *PostgresAffiliateRepository) ListApplications(ctx context.Context, status string, page, limit int) ([]models.AffiliateProfile, uint64, error) {
	filters := map[string]interface{}{"status": status}
	return r.ListProfiles(ctx, filters, page, limit)
}

// ApproveApplication approves an affiliate application
func (r *PostgresAffiliateRepository) ApproveApplication(ctx context.Context, id uint64, data map[string]interface{}) error {
	data["status"] = "active"
	data["approved_at"] = "NOW()" // This might need adjustment for sqlx
	return r.UpdateProfile(ctx, id, data)
}

// RejectApplication rejects an affiliate application
func (r *PostgresAffiliateRepository) RejectApplication(ctx context.Context, id uint64, reason string) error {
	data := map[string]interface{}{
		"status":           "rejected",
		"rejected_at":     "NOW()",
		"rejection_reason": reason,
	}
	return r.UpdateProfile(ctx, id, data)
}

// ListCommissions lists affiliate commissions with filtering and pagination
func (r *PostgresAffiliateRepository) ListCommissions(ctx context.Context, filters map[string]interface{}, page, limit int) ([]models.AffiliateCommission, uint64, error) {
	var commissions []models.AffiliateCommission
	var total uint64

	where := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if status, ok := filters["status"].(string); ok && status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if affID, ok := filters["affiliateId"].(uint64); ok && affID > 0 {
		where = append(where, fmt.Sprintf("affiliate_id = $%d", argIdx))
		args = append(args, affID)
		argIdx++
	}

	if dateFrom, ok := filters["dateFrom"].(string); ok && dateFrom != "" {
		where = append(where, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, dateFrom)
		argIdx++
	}

	if dateTo, ok := filters["dateTo"].(string); ok && dateTo != "" {
		where = append(where, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, dateTo)
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")
	
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM affiliate_commissions WHERE %s", whereClause)
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query := fmt.Sprintf(`
		SELECT * FROM affiliate_commissions 
		WHERE %s 
		ORDER BY created_at DESC 
		LIMIT %d OFFSET %d
	`, whereClause, limit, offset)

	err = r.db.SelectContext(ctx, &commissions, query, args...)
	return commissions, total, err
}

// GetCommissionByID retrieves a commission by ID
func (r *PostgresAffiliateRepository) GetCommissionByID(ctx context.Context, id uint64) (*models.AffiliateCommission, error) {
	var commission models.AffiliateCommission
	query := `SELECT * FROM affiliate_commissions WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &commission, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &commission, err
}

// UpdateCommissionStatus updates the status of a commission
func (r *PostgresAffiliateRepository) UpdateCommissionStatus(ctx context.Context, id uint64, status string, notes string) error {
	query := `UPDATE affiliate_commissions SET status = $1, notes = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, status, notes, id)
	return err
}

// MarkCommissionPaid marks a commission as paid
func (r *PostgresAffiliateRepository) MarkCommissionPaid(ctx context.Context, id uint64, paymentData map[string]interface{}) error {
	setClauses := []string{"status = 'paid'", "paid_at = NOW()", "updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

	if ref, ok := paymentData["payment_reference"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("payment_reference = $%d", argIdx))
		args = append(args, ref)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE affiliate_commissions SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetPerformanceReports retrieves performance reports
func (r *PostgresAffiliateRepository) GetPerformanceReports(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	// Simplified performance report example
	query := `
		SELECT 
			affiliate_id,
			COUNT(id) as conversions,
			SUM(order_amount) as revenue,
			SUM(commission_amount) as earnings
		FROM affiliate_commissions
		WHERE status != 'cancelled' AND deleted_at IS NULL
		GROUP BY affiliate_id
	`
	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		result := make(map[string]interface{})
		err := rows.MapScan(result)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}
