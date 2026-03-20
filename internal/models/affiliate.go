package models

import (
	"time"
)

type AffiliateClick struct {
	ID                  uint64     `json:"id" clickhouse:"id"`
	AffiliateID         uint64     `json:"affiliate_id" clickhouse:"affiliate_id"`
	AffiliateDiscountID *uint64    `json:"affiliate_discount_id" clickhouse:"affiliate_discount_id"`
	SessionID           string     `json:"session_id" clickhouse:"session_id"`
	IPAddress           *string    `json:"ip_address" clickhouse:"ip_address"`
	UserAgent           *string    `json:"user_agent" clickhouse:"user_agent"`
	ReferrerURL         *string    `json:"referrer_url" clickhouse:"referrer_url"`
	LandingPage         *string    `json:"landing_page" clickhouse:"landing_page"`
	Source              string     `json:"source" clickhouse:"source"`
	CountryID           *uint64    `json:"country_id" clickhouse:"country_id"`
	City                *string    `json:"city" clickhouse:"city"`
	DeviceType          *string    `json:"device_type" clickhouse:"device_type"`
	Browser             *string    `json:"browser" clickhouse:"browser"`
	OS                  *string    `json:"os" clickhouse:"os"`
	UTMSource           *string    `json:"utm_source" clickhouse:"utm_source"`
	UTMMedium           *string    `json:"utm_medium" clickhouse:"utm_medium"`
	UTMCampaign         *string    `json:"utm_campaign" clickhouse:"utm_campaign"`
	UTMContent          *string    `json:"utm_content" clickhouse:"utm_content"`
	UTMTerm             *string    `json:"utm_term" clickhouse:"utm_term"`
	Converted           bool       `json:"converted" clickhouse:"converted"`
	ConversionDate      *time.Time `json:"conversion_date" clickhouse:"conversion_date"`
	OrderID             *uint64    `json:"order_id" clickhouse:"order_id"`
	CommissionEarned    float64    `json:"commission_earned" clickhouse:"commission_earned"`
	CreatedAt           time.Time  `json:"created_at" clickhouse:"created_at"`

	// Input only fields (for validation)
	AffiliateCode string `json:"affiliate_code" clickhouse:"-"`
	CouponCode    string `json:"coupon_code" clickhouse:"-"`
}

type User struct {
	ID        uint64 `json:"id" db:"id"`
	FirstName string `json:"firstName" db:"first_name"`
	LastName  string `json:"lastName" db:"last_name"`
	Email     string `json:"email" db:"email"`
	Phone     string `json:"phone" db:"phone"`
}

type Country struct {
	ID   uint64 `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
	Code string `json:"code" db:"code"`
}

type Discount struct {
	ID           uint64  `json:"id" db:"id"`
	Title        string  `json:"title" db:"title"`
	Coupon       string  `json:"coupon" db:"coupon"`
	DiscountType string  `json:"discount_type" db:"discount_type"`
	AmountType   string  `json:"amount_type" db:"amount_type"`
	IsActive     bool    `json:"is_active" db:"is_active"`
	IsAutomatic  bool    `json:"is_automatic" db:"is_automatic"`
}

type AffiliateStats struct {
	AffiliateID      uint64  `json:"affiliate_id"`
	TotalClicks      uint64  `json:"totalClicks"`
	TotalConversions uint64  `json:"totalConversions"`
	TotalEarnings    float64 `json:"totalEarnings"`
	TotalRevenue     float64 `json:"totalRevenue"`
	ConversionRate   float64 `json:"conversionRate"`
}

type BulkStatsRequest struct {
	DateFrom     string   `json:"dateFrom"`
	DateTo       string   `json:"dateTo"`
	AffiliateIDs []uint64 `json:"affiliateIds"`
}

type ApproveApplicationRequest struct {
	CommissionRate float64 `json:"commissionRate"`
	Notes          string  `json:"notes"`
}

type RejectApplicationRequest struct {
	Reason string `json:"reason"`
}

type MarkCommissionPaidRequest struct {
	PaymentReference string `json:"paymentReference"`
}

type AffiliateProfile struct {
	ID               uint64     `json:"id" db:"id"`
	UserID           uint64     `json:"user_id" db:"user_id"`
	AffiliateCode    string     `json:"affiliate_code" db:"affiliate_code"`
	CountryID        *uint64    `json:"country_id" db:"country_id"`
	Status           string     `json:"status" db:"status"`
	CommissionType   string     `json:"commission_type" db:"commission_type"`
	CommissionRate   float64    `json:"commission_rate" db:"commission_rate"`
	TotalEarnings    float64    `json:"total_earnings" db:"total_earnings"`
	PendingEarnings  float64    `json:"pending_earnings" db:"pending_earnings"`
	PaidEarnings     float64    `json:"paid_earnings" db:"paid_earnings"`
	TotalClicks      uint64     `json:"total_clicks" db:"total_clicks"`
	TotalConversions uint64     `json:"total_conversions" db:"total_conversions"`
	TotalOrders      uint64     `json:"total_orders" db:"total_orders"`
	ConversionRate   float64    `json:"conversion_rate" db:"conversion_rate"`
	PaymentMethod    *string    `json:"payment_method" db:"payment_method"`
	PaymentDetails   *string    `json:"payment_details" db:"payment_details"`
	Bio              *string    `json:"bio" db:"bio"`
	WebsiteURL       *string    `json:"website_url" db:"website_url"`
	SocialMedia      *string    `json:"social_media" db:"social_media"`
	ApprovedAt       *time.Time `json:"approved_at" db:"approved_at"`
	RejectedAt       *time.Time `json:"rejected_at" db:"rejected_at"`
	RejectionReason  *string    `json:"rejection_reason" db:"rejection_reason"`
	Notes            *string    `json:"notes" db:"notes"`
	IsActive         bool       `json:"is_active" db:"is_active"`
	Remarks          *string    `json:"remarks" db:"remarks"`
	CreatedBy        *uint64    `json:"created_by" db:"created_by"`
	UpdatedBy        *uint64    `json:"updated_by" db:"updated_by"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at" db:"deleted_at"`

	// Relations
	User               *User               `json:"user,omitempty"`
	Country            *Country            `json:"country,omitempty"`
	AffiliateDiscounts []AffiliateDiscount `json:"affiliateDiscounts,omitempty"`
}

type AffiliateDiscount struct {
	ID                    uint64     `json:"id" db:"id"`
	AffiliateID           uint64     `json:"affiliate_id" db:"affiliate_id"`
	DiscountID            uint64     `json:"discount_id" db:"discount_id"`
	Status                string     `json:"status" db:"status"`
	UsageCount            uint64     `json:"usage_count" db:"usage_count"`
	MaxUsage              *uint64    `json:"max_usage" db:"max_usage"`
	TotalCommissionEarned float64    `json:"total_commission_earned" db:"total_commission_earned"`
	StartDate             *time.Time `json:"start_date" db:"start_date"`
	EndDate               *time.Time `json:"end_date" db:"end_date"`
	Notes                 *string    `json:"notes" db:"notes"`
	CommissionType        *string    `json:"commission_type" db:"commission_type"`
	CommissionRate        *float64   `json:"commission_rate" db:"commission_rate"`
	IsActive              bool       `json:"is_active" db:"is_active"`
	Remarks               *string    `json:"remarks" db:"remarks"`
	CreatedBy             *uint64    `json:"created_by" db:"created_by"`
	UpdatedBy             *uint64    `json:"updated_by" db:"updated_by"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt             *time.Time `json:"deleted_at" db:"deleted_at"`

	// Relations
	Discount *Discount `json:"discount,omitempty"`
}

type AffiliateCommission struct {
	ID                  uint64     `json:"id" db:"id"`
	AffiliateID         uint64     `json:"affiliate_id" db:"affiliate_id"`
	AffiliateDiscountID uint64     `json:"affiliate_discount_id" db:"affiliate_discount_id"`
	OrderID             *uint64    `json:"order_id" db:"order_id"`
	DiscountID          *uint64    `json:"discount_id" db:"discount_id"`
	ClickID             *uint64    `json:"click_id" db:"click_id"`
	Source              string     `json:"source" db:"source"`
	Status              string     `json:"status" db:"status"`
	OrderAmount         float64    `json:"order_amount" db:"order_amount"`
	CommissionRate      float64    `json:"commission_rate" db:"commission_rate"`
	CommissionAmount    float64    `json:"commission_amount" db:"commission_amount"`
	CurrencyCode        string     `json:"currency_code" db:"currency_code"`
	CustomerEmail       *string    `json:"customer_email" db:"customer_email"`
	CustomerPhone       *string    `json:"customer_phone" db:"customer_phone"`
	CouponCode          *string    `json:"coupon_code" db:"coupon_code"`
	DiscountAmount      *float64   `json:"discount_amount" db:"discount_amount"`
	ApprovedAt          *time.Time `json:"approved_at" db:"approved_at"`
	PaidAt              *time.Time `json:"paid_at" db:"paid_at"`
	PaymentReference    *string    `json:"payment_reference" db:"payment_reference"`
	Notes               *string    `json:"notes" db:"notes"`
	RefundReason        *string    `json:"refund_reason" db:"refund_reason"`
	RefundedAt          *time.Time `json:"refunded_at" db:"refunded_at"`
	CountryID           *uint64    `json:"country_id" db:"country_id"`
	UUID                *string    `json:"uuid" db:"uuid"`
	InvoiceNo           *string    `json:"invoice_no" db:"invoice_no"`
	CommissionType      *string    `json:"commission_type" db:"commission_type"`
	IsActive            bool       `json:"is_active" db:"is_active"`
	Remarks             *string    `json:"remarks" db:"remarks"`
	CreatedBy           *uint64    `json:"created_by" db:"created_by"`
	UpdatedBy           *uint64    `json:"updated_by" db:"updated_by"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at" db:"deleted_at"`
}

type AffiliateConversion struct {
	ID                  uint64     `json:"id" db:"id"`
	AffiliateID         uint64     `json:"affiliate_id" db:"affiliate_id"`
	AffiliateDiscountID *uint64    `json:"affiliate_discount_id" db:"affiliate_discount_id"`
	ClickID             *uint64    `json:"click_id" db:"click_id"`
	OrderID             *uint64    `json:"order_id" db:"order_id"`
	Type                string     `json:"type" db:"type"`
	ConversionValue     float64    `json:"conversion_value" db:"conversion_value"`
	CustomerEmail       *string    `json:"customer_email" db:"customer_email"`
	CustomerPhone       *string    `json:"customer_phone" db:"customer_phone"`
	SessionID           *string    `json:"session_id" db:"session_id"`
	IPAddress           *string    `json:"ip_address" db:"ip_address"`
	ConversionPage      *string    `json:"conversion_page" db:"conversion_page"`
	TimeToConversion    *int        `json:"time_to_conversion" db:"time_to_conversion"`
	CouponUsed          *string    `json:"coupon_used" db:"coupon_used"`
	DiscountApplied     *float64   `json:"discount_applied" db:"discount_applied"`
	CommissionEarned    float64    `json:"commission_earned" db:"commission_earned"`
	ConversionData      *string    `json:"conversion_data" db:"conversion_data"`
	IsActive            bool       `json:"is_active" db:"is_active"`
	Remarks             *string    `json:"remarks" db:"remarks"`
	CreatedBy           *uint64    `json:"created_by" db:"created_by"`
	UpdatedBy           *uint64    `json:"updated_by" db:"updated_by"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at" db:"deleted_at"`
}

type AffiliateDiscountUsage struct {
	ID                  uint64     `json:"id" db:"id"`
	AffiliateDiscountID *uint64    `json:"affiliate_discount_id" db:"affiliate_discount_id"`
	DiscountUsageID     *uint64    `json:"discount_usage_id" db:"discount_usage_id"`
	OrderID             uint64     `json:"order_id" db:"order_id"`
	CustomerEmail       string     `json:"customer_email" db:"customer_email"`
	CustomerPhone       *string    `json:"customer_phone" db:"customer_phone"`
	OrderAmount         float64    `json:"order_amount" db:"order_amount"`
	DiscountAmount      float64    `json:"discount_amount" db:"discount_amount"`
	CommissionRate      float64    `json:"commission_rate" db:"commission_rate"`
	CommissionAmount    float64    `json:"commission_amount" db:"commission_amount"`
	SessionID           *string    `json:"session_id" db:"session_id"`
	IPAddress           *string    `json:"ip_address" db:"ip_address"`
	UserAgent           *string    `json:"user_agent" db:"user_agent"`
	ReferrerURL         *string    `json:"referrer_url" db:"referrer_url"`
	UTMSource           *string    `json:"utm_source" db:"utm_source"`
	UTMMedium           *string    `json:"utm_medium" db:"utm_medium"`
	UTMCampaign         *string    `json:"utm_campaign" db:"utm_campaign"`
	AffiliateID         *uint64    `json:"affiliate_id" db:"affiliate_id"`
	IsActive            bool       `json:"is_active" db:"is_active"`
	Remarks             *string    `json:"remarks" db:"remarks"`
	CreatedBy           *uint64    `json:"created_by" db:"created_by"`
	UpdatedBy           *uint64    `json:"updated_by" db:"updated_by"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at" db:"deleted_at"`
}
