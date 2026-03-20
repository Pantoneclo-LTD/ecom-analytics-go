package repository

import (
	"context"
	"ecom-analytics-go/internal/models"
)

type AffiliateRepository interface {
	// Affiliate Profile
	GetProfileByID(ctx context.Context, id uint64) (*models.AffiliateProfile, error)
	GetProfileByCode(ctx context.Context, code string) (*models.AffiliateProfile, error)
	GetProfileByUserID(ctx context.Context, userID uint64) (*models.AffiliateProfile, error)
	ListProfiles(ctx context.Context, filters map[string]interface{}, page, limit int) ([]models.AffiliateProfile, uint64, error)
	GetProfileByDiscountCode(ctx context.Context, code string) (*models.AffiliateProfile, *uint64, error)
	CreateProfile(ctx context.Context, profile *models.AffiliateProfile) error
	UpdateProfile(ctx context.Context, id uint64, data map[string]interface{}) error
	SoftDeleteProfile(ctx context.Context, id uint64) error

	// Applications
	ListApplications(ctx context.Context, status string, page, limit int) ([]models.AffiliateProfile, uint64, error)
	ApproveApplication(ctx context.Context, id uint64, data map[string]interface{}) error
	RejectApplication(ctx context.Context, id uint64, reason string) error

	// Affiliate Discount
	GetDiscountByID(ctx context.Context, id uint64) (*models.AffiliateDiscount, error)
	ListDiscounts(ctx context.Context, affiliateID uint64) ([]models.AffiliateDiscount, error)
	CreateDiscount(ctx context.Context, discount *models.AffiliateDiscount) error
	UpdateDiscount(ctx context.Context, id uint64, data map[string]interface{}) error
	SoftDeleteDiscount(ctx context.Context, id uint64) error

	// Tracking & Conversions
	CreateCommission(ctx context.Context, commission *models.AffiliateCommission) error
	CreateConversion(ctx context.Context, conversion *models.AffiliateConversion) error
	CreateDiscountUsage(ctx context.Context, usage *models.AffiliateDiscountUsage) error

	// Commission Management
	ListCommissions(ctx context.Context, filters map[string]interface{}, page, limit int) ([]models.AffiliateCommission, uint64, error)
	GetCommissionByID(ctx context.Context, id uint64) (*models.AffiliateCommission, error)
	UpdateCommissionStatus(ctx context.Context, id uint64, status string, notes string) error
	MarkCommissionPaid(ctx context.Context, id uint64, paymentData map[string]interface{}) error

	// Statistics & Analytics (PG side)
	GetCommissionStats(ctx context.Context, affiliateID uint64) (*models.AffiliateStats, error)
	GetTimeBasedStats(ctx context.Context, affiliateID uint64, startTime, endTime string) ([]map[string]interface{}, error)
	GetPerformanceReports(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error)
}
