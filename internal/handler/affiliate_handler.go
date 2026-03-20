package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"ecom-analytics-go/internal/models"
	r "ecom-analytics-go/internal/repository"

	"github.com/go-chi/chi/v5"
)

type AffiliateHandler struct {
	pgRepo r.AffiliateRepository
	chRepo r.ClickHouseRepository
}

func NewAffiliateHandler(pgRepo r.AffiliateRepository, chRepo r.ClickHouseRepository) *AffiliateHandler {
	return &AffiliateHandler{
		pgRepo: pgRepo,
		chRepo: chRepo,
	}
}

func (h *AffiliateHandler) TrackClick(w http.ResponseWriter, r *http.Request) {
	var click models.AffiliateClick
	if err := json.NewDecoder(r.Body).Decode(&click); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Resolve affiliate-related IDs
	_ = h.resolveAffiliate(r.Context(), &click)

	if err := h.chRepo.TrackClick(r.Context(), &click); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(click)
}

func (h *AffiliateHandler) resolveAffiliate(ctx context.Context, click *models.AffiliateClick) error {
	if click.AffiliateCode != "" {
		profile, err := h.pgRepo.GetProfileByCode(ctx, click.AffiliateCode)
		if err == nil && profile != nil {
			click.AffiliateID = profile.ID
			return nil
		}
	}

	coupon := click.CouponCode
	if coupon == "" && click.AffiliateCode != "" {
		coupon = click.AffiliateCode
	}

	if coupon != "" {
		profile, discountID, err := h.pgRepo.GetProfileByDiscountCode(ctx, coupon)
		if err == nil && profile != nil {
			click.AffiliateID = profile.ID
			click.AffiliateDiscountID = discountID
			return nil
		}
	}

	return fmt.Errorf("could not resolve affiliate")
}

func (h *AffiliateHandler) GetAffiliateStats(w http.ResponseWriter, r *http.Request) {
	affiliateIDStr := r.URL.Query().Get("affiliateId")
	affiliateID, _ := strconv.ParseUint(affiliateIDStr, 10, 64)

	// Combine stats from both PG and ClickHouse
	pgStats, _ := h.pgRepo.GetCommissionStats(r.Context(), affiliateID)
	clicks, _, _ := h.chRepo.GetClickStats(r.Context(), affiliateID)

	stats := models.AffiliateStats{
		AffiliateID:      affiliateID,
		TotalClicks:      clicks,
		TotalConversions: pgStats.TotalConversions,
		TotalEarnings:    pgStats.TotalEarnings,
		TotalRevenue:     pgStats.TotalRevenue,
	}

	if stats.TotalClicks > 0 {
		stats.ConversionRate = float64(stats.TotalConversions) / float64(stats.TotalClicks) * 100
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *AffiliateHandler) GetClickRateAnalytics(w http.ResponseWriter, r *http.Request) {
	affiliateIDStr := r.URL.Query().Get("affiliateId")
	affiliateID, _ := strconv.ParseUint(affiliateIDStr, 10, 64)

	clicks, _, err := h.chRepo.GetClickStats(r.Context(), affiliateID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pgStats, _ := h.pgRepo.GetCommissionStats(r.Context(), affiliateID)

	rate := 0.0
	if clicks > 0 {
		rate = float64(pgStats.TotalConversions) / float64(clicks) * 100
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"affiliate_id":    affiliateID,
		"click_count":     clicks,
		"conversion_rate": rate,
	})
}

func (h *AffiliateHandler) GetBulkAffiliateStats(w http.ResponseWriter, r *http.Request) {
	var req models.BulkStatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stats, err := h.chRepo.GetBulkClickStats(r.Context(), req.AffiliateIDs, req.DateFrom, req.DateTo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *AffiliateHandler) GetClickInfo(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	click, err := h.chRepo.GetClickInfo(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "Click info not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(click)
}

func (h *AffiliateHandler) LookupAffiliateByIP(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ipAddress")
	ua := r.URL.Query().Get("userAgent")

	click, err := h.chRepo.LookupAffiliateByIP(r.Context(), ip, ua, 24)
	if err != nil {
		http.Error(w, "No recent click found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(click)
}

func (h *AffiliateHandler) TrackConversion(w http.ResponseWriter, r *http.Request) {
	var conv models.AffiliateConversion
	if err := json.NewDecoder(r.Body).Decode(&conv); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Create conversion record in PG
	if err := h.pgRepo.CreateConversion(r.Context(), &conv); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Mark click as converted in ClickHouse if click_id is present
	if conv.ClickID != nil {
		clickIDStr := fmt.Sprintf("%d", *conv.ClickID)
		_ = h.chRepo.MarkConverted(r.Context(), clickIDStr, *conv.OrderID, conv.CommissionEarned)
	}

	// 3. Create commission record if applicable
	if conv.CommissionEarned > 0 {
		commission := models.AffiliateCommission{
			AffiliateID:      conv.AffiliateID,
			OrderID:          conv.OrderID,
			CommissionAmount: conv.CommissionEarned,
			Status:           "pending",
			Source:           conv.Type,
			CurrencyCode:     "USD", // Default
		}
		if conv.AffiliateDiscountID != nil {
			commission.AffiliateDiscountID = *conv.AffiliateDiscountID
		}
		_ = h.pgRepo.CreateCommission(r.Context(), &commission)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conv)
}

func (h *AffiliateHandler) GenerateTrackingURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BaseURL       string            `json:"baseUrl"`
		AffiliateCode string            `json:"affiliateCode"`
		UTM           map[string]string `json:"utm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Simple URL generation logic
	trackingURL := fmt.Sprintf("%s?ref=%s", req.BaseURL, req.AffiliateCode)
	for k, v := range req.UTM {
		if v != "" {
			trackingURL += fmt.Sprintf("&%s=%s", k, v)
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"trackingUrl": trackingURL})
}

func (h *AffiliateHandler) GetTimeBasedStats(w http.ResponseWriter, r *http.Request) {
	affiliateIDStr := r.URL.Query().Get("affiliateId")
	affiliateID, _ := strconv.ParseUint(affiliateIDStr, 10, 64)
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	stats, err := h.pgRepo.GetTimeBasedStats(r.Context(), affiliateID, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *AffiliateHandler) GetComplexAnalytics(w http.ResponseWriter, r *http.Request) {
	// Placeholder for more advanced analytics
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "placeholder",
		"message": "Complex analytics engine to be implemented",
	})
}

// --- CRUD for Profiles ---

func (h *AffiliateHandler) CreateAffiliate(w http.ResponseWriter, r *http.Request) {
	var profile models.AffiliateProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.pgRepo.CreateProfile(r.Context(), &profile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

func (h *AffiliateHandler) GetAffiliate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	profile, err := h.pgRepo.GetProfileByID(r.Context(), id)
	if err != nil || profile == nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *AffiliateHandler) GetAffiliateByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	profile, err := h.pgRepo.GetProfileByCode(r.Context(), code)
	if err != nil || profile == nil {
		http.Error(w, "Affiliate not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *AffiliateHandler) UpdateAffiliate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.pgRepo.UpdateProfile(r.Context(), id, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AffiliateHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err := h.pgRepo.SoftDeleteProfile(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AffiliateHandler) ListAffiliates(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filters := map[string]interface{}{
		"status": query.Get("status"),
		"search": query.Get("search"),
	}
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 {
		limit = 10
	}

	profiles, total, err := h.pgRepo.ListProfiles(r.Context(), filters, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  profiles,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AffiliateHandler) GetAffiliateByUserID(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseUint(chi.URLParam(r, "userId"), 10, 64)
	profile, err := h.pgRepo.GetProfileByUserID(r.Context(), userID)
	if err != nil || profile == nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// --- Application Management ---

func (h *AffiliateHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 { page = 1 }
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 { limit = 10 }

	apps, total, err := h.pgRepo.ListApplications(r.Context(), status, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  apps,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AffiliateHandler) ApproveApplication(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var req models.ApproveApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data := map[string]interface{}{
		"commission_rate": req.CommissionRate,
		"remarks":         req.Notes,
	}

	if err := h.pgRepo.ApproveApplication(r.Context(), id, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AffiliateHandler) RejectApplication(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var req models.RejectApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.pgRepo.RejectApplication(r.Context(), id, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// --- Commission Management ---

func (h *AffiliateHandler) ListCommissions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filters := map[string]interface{}{
		"status":      query.Get("status"),
		"dateFrom":    query.Get("dateFrom"),
		"dateTo":      query.Get("dateTo"),
	}
	if affIDStr := query.Get("affiliateId"); affIDStr != "" {
		if id, err := strconv.ParseUint(affIDStr, 10, 64); err == nil {
			filters["affiliateId"] = id
		}
	}

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 { page = 1 }
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 { limit = 10 }

	commissions, total, err := h.pgRepo.ListCommissions(r.Context(), filters, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  commissions,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AffiliateHandler) ApproveCommission(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err := h.pgRepo.UpdateCommissionStatus(r.Context(), id, "approved", "Approved by admin"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AffiliateHandler) RejectCommission(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var req struct { Reason string `json:"reason"` }
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.pgRepo.UpdateCommissionStatus(r.Context(), id, "rejected", req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func (h *AffiliateHandler) GetCartCommissionPreview(w http.ResponseWriter, r *http.Request) {
	cartIDStr := r.URL.Query().Get("cartId")
	couponCode := r.URL.Query().Get("couponCode")
	
	// Basic implementation for now: find affiliate by coupon and return their rate
	profile, _, err := h.pgRepo.GetProfileByDiscountCode(r.Context(), couponCode)
	if err != nil || profile == nil {
		http.Error(w, "Affiliate not found for coupon", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"cartId":         cartIDStr,
		"couponCode":      couponCode,
		"affiliateId":     profile.ID,
		"commissionRate":  profile.CommissionRate,
		"commissionType":  profile.CommissionType,
		"status":          "preview",
	})
}

func (h *AffiliateHandler) CalculateOrderCommission(w http.ResponseWriter, r *http.Request) {
	var req struct { OrderID uint64 `json:"orderId"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// This would typically involve fetching order details and applying affiliate logic
	// For now, return a placeholder or success message
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orderId": req.OrderID,
		"status":  "calculation_requested",
		"message": "Order commission calculation trigger received",
	})
}

func (h *AffiliateHandler) MarkCommissionPaid(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var req models.MarkCommissionPaidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data := map[string]interface{}{
		"payment_reference": req.PaymentReference,
	}

	if err := h.pgRepo.MarkCommissionPaid(r.Context(), id, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AffiliateHandler) GetPerformanceReports(w http.ResponseWriter, r *http.Request) {
	filters := make(map[string]interface{})
	reports, err := h.pgRepo.GetPerformanceReports(r.Context(), filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func (h *AffiliateHandler) ListDiscountsForAffiliate(w http.ResponseWriter, r *http.Request) {
	affiliateIDStr := chi.URLParam(r, "affiliateId")
	affiliateID, _ := strconv.ParseUint(affiliateIDStr, 10, 64)

	discounts, err := h.pgRepo.ListDiscounts(r.Context(), affiliateID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(discounts)
}

func (h *AffiliateHandler) AssignDiscount(w http.ResponseWriter, r *http.Request) {
	var discount models.AffiliateDiscount
	if err := json.NewDecoder(r.Body).Decode(&discount); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.pgRepo.CreateDiscount(r.Context(), &discount); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(discount)
}

func (h *AffiliateHandler) UpdateDiscountAssignment(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.pgRepo.UpdateDiscount(r.Context(), id, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AffiliateHandler) RemoveDiscountFromAffiliate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err := h.pgRepo.SoftDeleteDiscount(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
