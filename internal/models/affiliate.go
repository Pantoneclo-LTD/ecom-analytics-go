package models

import (
	"time"

	"github.com/google/uuid"
)

type AffiliateClick struct {
	ID                  uuid.UUID  `json:"id" clickhouse:"id"`
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
