package handler

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	maxMerchantCallbackBodySize = 1 << 20
	defaultMerchantFaceValue    = 10.0
	defaultMerchantUSDRatio     = 10.0
)

var (
	errMerchantTokenRequired = infraerrors.BadRequest("MERCHANT_TOKEN_REQUIRED", "merchant callback token is required")
	errMerchantTokenInvalid  = infraerrors.Unauthorized("MERCHANT_TOKEN_INVALID", "invalid merchant callback token")
	errMerchantOrderRequired = infraerrors.BadRequest("MERCHANT_ORDER_REQUIRED", "merchant order id is required")
	errMerchantFaceValueBad  = infraerrors.BadRequest("MERCHANT_FACE_VALUE_INVALID", "merchant face value is invalid")
	errMerchantStatusMissing = infraerrors.BadRequest("MERCHANT_STATUS_REQUIRED", "merchant payment status is required")
)

// RedeemHandler handles redeem code-related requests
type RedeemHandler struct {
	redeemService     *service.RedeemService
	settingService    *service.SettingService
	promoService      *service.PromoService
	enterpriseService *service.EnterpriseService
}

// NewRedeemHandler creates a new RedeemHandler
func NewRedeemHandler(redeemService *service.RedeemService, settingService *service.SettingService, promoService *service.PromoService, enterpriseService *service.EnterpriseService) *RedeemHandler {
	return &RedeemHandler{
		redeemService:    redeemService,
		settingService:   settingService,
		promoService:     promoService,
		enterpriseService: enterpriseService,
	}
}

// RedeemRequest represents the redeem code request payload
type RedeemRequest struct {
	Code string `json:"code" binding:"required"`
}

// RedeemResponse represents the redeem response
type RedeemResponse struct {
	Message        string   `json:"message"`
	Type           string   `json:"type"`
	Value          float64  `json:"value"`
	NewBalance     *float64 `json:"new_balance,omitempty"`
	NewConcurrency *int     `json:"new_concurrency,omitempty"`
}

type ApplyPromoCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

type ApplyPromoCodeResponse struct {
	Message        string   `json:"message"`
	BonusAmount    float64  `json:"bonus_amount,omitempty"`
	DiscountFactor float64  `json:"discount_factor,omitempty"`
	DiscountLabel  string   `json:"discount_label,omitempty"`
	DiscountScope  string   `json:"discount_scope,omitempty"`
	NewBalance     *float64 `json:"new_balance,omitempty"`
}

type merchantCallbackPayload struct {
	OrderID        string
	FaceValueRaw   string
	PaymentStatus  string
	PaymentStatus2 string
	ProductName    string
	Buyer          string
}

type merchantCreateCardResponse struct {
	OrderID      string  `json:"order_id"`
	RedeemCode   string  `json:"redeem_code"`
	Type         string  `json:"type"`
	FaceValue    float64 `json:"face_value"`
	BalanceValue float64 `json:"balance_value"`
	Message      string  `json:"message"`
}

// Redeem handles redeeming a code
// POST /api/v1/redeem
func (h *RedeemHandler) Redeem(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req RedeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if h.enterpriseService != nil {
		enterprise, err := h.enterpriseService.GetUserEnterpriseContext(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if enterprise != nil && enterprise.SelfRedeemBlocked {
			response.ErrorFrom(c, service.ErrEnterpriseSelfRedeemForbidden)
			return
		}
	}

	result, err := h.redeemService.Redeem(c.Request.Context(), subject.UserID, req.Code)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.RedeemCodeFromService(result))
}

// GetHistory returns the user's redemption history
// GET /api/v1/redeem/history
func (h *RedeemHandler) GetHistory(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Default limit is 25
	limit := 25

	codes, err := h.redeemService.GetUserHistory(c.Request.Context(), subject.UserID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.RedeemCode, 0, len(codes))
	for i := range codes {
		out = append(out, *dto.RedeemCodeFromService(&codes[i]))
	}
	response.Success(c, out)
}

// ApplyPromoCode applies a registration promo code for an already registered user.
// POST /api/v1/redeem/promo
func (h *RedeemHandler) ApplyPromoCode(c *gin.Context) {
	if h == nil || h.promoService == nil {
		response.InternalError(c, "promo service not configured")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req ApplyPromoCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if h.enterpriseService != nil {
		enterprise, err := h.enterpriseService.GetUserEnterpriseContext(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if enterprise != nil && enterprise.SelfRedeemBlocked {
			response.ErrorFrom(c, service.ErrEnterpriseSelfRedeemForbidden)
			return
		}
	}

	result, err := h.promoService.ApplyPromoCodeDetailed(c.Request.Context(), subject.UserID, req.Code)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	resp := ApplyPromoCodeResponse{
		Message:        "Promo code applied successfully",
		BonusAmount:    result.BonusAmount,
		DiscountFactor: service.NormalizePricingDiscountFactorForRepo(result.DiscountFactor),
		DiscountLabel:  result.DiscountLabel,
		DiscountScope:  service.NormalizePromoDiscountScope(result.DiscountScope),
	}

	response.Success(c, resp)
}

// CreateMerchantBalanceCard creates a balance redeem code from an external merchant callback.
// It never fulfills balance/subscription directly; users still redeem the code manually.
// POST /api/v1/merchant/callbacks/balance-card?token=...
func (h *RedeemHandler) CreateMerchantBalanceCard(c *gin.Context) {
	if h == nil || h.redeemService == nil {
		response.InternalError(c, "merchant callback service not configured")
		return
	}

	callbackToken := strings.TrimSpace(c.Query("token"))
	if callbackToken == "" {
		response.ErrorFrom(c, errMerchantTokenRequired)
		return
	}
	expectedToken := ""
	if h.settingService != nil {
		expectedToken, _ = h.settingService.GetRawSetting(c.Request.Context(), service.SettingKeyMerchantCallbackToken)
		expectedToken = strings.TrimSpace(expectedToken)
	}
	if !merchantTokenMatches(callbackToken, expectedToken) {
		response.ErrorFrom(c, errMerchantTokenInvalid)
		return
	}

	providedToken := strings.TrimSpace(firstNonEmpty(
		c.GetHeader("x-merchant-token"),
		c.GetHeader("x-callback-token"),
		c.GetHeader("authorization"),
	))
	if strings.HasPrefix(strings.ToLower(providedToken), "bearer ") {
		providedToken = strings.TrimSpace(providedToken[7:])
	}
	if providedToken != "" && !merchantTokenMatches(providedToken, expectedToken) {
		response.ErrorFrom(c, errMerchantTokenInvalid)
		return
	}

	payload, err := parseMerchantCallbackPayload(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if payload.OrderID == "" {
		response.ErrorFrom(c, errMerchantOrderRequired)
		return
	}
	if merchantPaymentStatus(payload) == "" {
		response.ErrorFrom(c, errMerchantStatusMissing)
		return
	}
	if !merchantStatusLooksSuccessful(payload) {
		writeMerchantCallbackResponse(c, merchantCreateCardResponse{
			OrderID: payload.OrderID,
			Message: "ignored: payment status is not successful",
		})
		return
	}

	faceValue, err := resolveMerchantFaceValue(payload)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	balanceValue := faceValue * defaultMerchantUSDRatio
	code := buildMerchantRedeemCode(payload.OrderID)
	notes := buildMerchantRedeemNotes(payload, faceValue, balanceValue)

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = "merchant-balance-card:" + payload.OrderID
		c.Request.Header.Set("Idempotency-Key", idempotencyKey)
	}

	executeMerchantIdempotent(
		c,
		"merchant.callbacks.balance_card",
		struct {
			OrderID    string  `json:"order_id"`
			FaceValue  float64 `json:"face_value"`
			BalanceUSD float64 `json:"balance_usd"`
		}{
			OrderID:    payload.OrderID,
			FaceValue:  faceValue,
			BalanceUSD: balanceValue,
		},
		service.DefaultWriteIdempotencyTTL(),
		func(ctx context.Context) (any, error) {
			existing, lookupErr := h.redeemService.GetByCode(ctx, code)
			if lookupErr == nil {
				return merchantCreateCardResponse{
					OrderID:      payload.OrderID,
					RedeemCode:   existing.Code,
					Type:         existing.Type,
					FaceValue:    faceValue,
					BalanceValue: existing.Value,
					Message:      "redeem code already exists",
				}, nil
			}
			if !errors.Is(lookupErr, service.ErrRedeemCodeNotFound) {
				return nil, lookupErr
			}

			createErr := h.redeemService.CreateCode(ctx, &service.RedeemCode{
				Code:   code,
				Type:   service.RedeemTypeBalance,
				Value:  balanceValue,
				Status: service.StatusUnused,
				Notes:  notes,
			})
			if createErr != nil {
				existingAfterCreateErr, getErr := h.redeemService.GetByCode(ctx, code)
				if getErr == nil {
					return merchantCreateCardResponse{
						OrderID:      payload.OrderID,
						RedeemCode:   existingAfterCreateErr.Code,
						Type:         existingAfterCreateErr.Type,
						FaceValue:    faceValue,
						BalanceValue: existingAfterCreateErr.Value,
						Message:      "redeem code already exists",
					}, nil
				}
				return nil, createErr
			}

			return merchantCreateCardResponse{
				OrderID:      payload.OrderID,
				RedeemCode:   code,
				Type:         service.RedeemTypeBalance,
				FaceValue:    faceValue,
				BalanceValue: balanceValue,
				Message:      "redeem code created",
			}, nil
		},
	)
}

func parseMerchantCallbackPayload(c *gin.Context) (*merchantCallbackPayload, error) {
	data := make(map[string]string)

	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			data[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(values[0])
		}
	}

	bodyBytes := []byte(nil)
	if c.Request.Body != nil {
		readBody, err := io.ReadAll(io.LimitReader(c.Request.Body, maxMerchantCallbackBodySize))
		if err != nil {
			return nil, fmt.Errorf("read merchant callback body: %w", err)
		}
		bodyBytes = readBody
		c.Request.Body = io.NopCloser(strings.NewReader(string(readBody)))
	}

	if err := c.Request.ParseForm(); err == nil {
		for key, values := range c.Request.PostForm {
			if len(values) > 0 {
				data[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(values[0])
			}
		}
	}

	if len(bodyBytes) > 0 {
		trimmed := strings.TrimSpace(string(bodyBytes))
		if strings.HasPrefix(trimmed, "{") {
			var payload map[string]any
			if err := json.Unmarshal(bodyBytes, &payload); err == nil {
				for key, value := range payload {
					data[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(fmt.Sprint(value))
				}
			}
		}
	}

	return &merchantCallbackPayload{
		OrderID:        firstNonEmptyMap(data, "order_id", "out_trade_no", "trade_no", "merchant_order_id", "platform_order_id", "orderid"),
		FaceValueRaw:   firstNonEmptyMap(data, "face_value", "card_value", "goods_value", "product_value", "amount_face", "denomination"),
		PaymentStatus:  firstNonEmptyMap(data, "status", "pay_status", "trade_status", "order_status"),
		PaymentStatus2: firstNonEmptyMap(data, "state", "paystate", "payment_status"),
		ProductName:    firstNonEmptyMap(data, "product_name", "goods_name", "name", "title"),
		Buyer:          firstNonEmptyMap(data, "buyer", "buyer_id", "uid", "user", "email"),
	}, nil
}

func resolveMerchantFaceValue(payload *merchantCallbackPayload) (float64, error) {
	raw := strings.TrimSpace(payload.FaceValueRaw)
	if raw == "" {
		return defaultMerchantFaceValue, nil
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return 0, errMerchantFaceValueBad
	}
	return value, nil
}

func merchantStatusLooksSuccessful(payload *merchantCallbackPayload) bool {
	status := merchantPaymentStatus(payload)
	for _, ok := range []string{"success", "paid", "pay_success", "trade_success", "completed", "complete", "1", "true"} {
		if status == ok {
			return true
		}
	}
	return false
}

func merchantPaymentStatus(payload *merchantCallbackPayload) string {
	if payload == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(firstNonEmpty(payload.PaymentStatus, payload.PaymentStatus2)))
}

func merchantTokenMatches(provided, expected string) bool {
	provided = strings.TrimSpace(provided)
	expected = strings.TrimSpace(expected)
	if provided == "" || expected == "" || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func buildMerchantRedeemCode(orderID string) string {
	orderID = strings.TrimSpace(orderID)
	sum := sha256.Sum256([]byte(orderID))
	return "SHOP-" + strings.ToUpper(hex.EncodeToString(sum[:12]))
}

func buildMerchantRedeemNotes(payload *merchantCallbackPayload, faceValue, balanceValue float64) string {
	parts := []string{
		fmt.Sprintf("merchant order: %s", payload.OrderID),
		fmt.Sprintf("face value: %.2f", faceValue),
		fmt.Sprintf("balance usd: %.2f", balanceValue),
	}
	if payload.ProductName != "" {
		parts = append(parts, "product: "+payload.ProductName)
	}
	if payload.Buyer != "" {
		parts = append(parts, "buyer: "+payload.Buyer)
	}
	return strings.Join(parts, "; ")
}

func writeMerchantCallbackResponse(c *gin.Context, resp merchantCreateCardResponse) {
	mode := strings.ToLower(strings.TrimSpace(c.Query("response_mode")))
	switch mode {
	case "text", "plain":
		c.String(200, resp.RedeemCode)
	default:
		response.Success(c, resp)
	}
}

func executeMerchantIdempotent(
	c *gin.Context,
	scope string,
	payload any,
	ttl time.Duration,
	execute func(context.Context) (any, error),
) {
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		data, err := execute(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		writeMerchantCallbackAnyResponse(c, data)
		return
	}

	result, err := coordinator.Execute(c.Request.Context(), service.IdempotencyExecuteOptions{
		Scope:          scope,
		ActorScope:     "merchant:callback",
		Method:         c.Request.Method,
		Route:          c.FullPath(),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Payload:        payload,
		RequireKey:     true,
		TTL:            ttl,
	}, execute)
	if err != nil {
		if retryAfter := service.RetryAfterSecondsFromError(err); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		response.ErrorFrom(c, err)
		return
	}
	if result != nil && result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	if result == nil {
		writeMerchantCallbackAnyResponse(c, nil)
		return
	}
	writeMerchantCallbackAnyResponse(c, result.Data)
}

func writeMerchantCallbackAnyResponse(c *gin.Context, data any) {
	switch v := data.(type) {
	case merchantCreateCardResponse:
		writeMerchantCallbackResponse(c, v)
	case *merchantCreateCardResponse:
		if v == nil {
			response.Success(c, gin.H{})
			return
		}
		writeMerchantCallbackResponse(c, *v)
	case map[string]any:
		resp := merchantCreateCardResponse{
			OrderID:      toString(v["order_id"]),
			RedeemCode:   toString(v["redeem_code"]),
			Type:         toString(v["type"]),
			FaceValue:    toFloat64(v["face_value"]),
			BalanceValue: toFloat64(v["balance_value"]),
			Message:      toString(v["message"]),
		}
		if resp.OrderID != "" || resp.RedeemCode != "" {
			writeMerchantCallbackResponse(c, resp)
			return
		}
		response.Success(c, data)
	default:
		response.Success(c, data)
	}
}

func firstNonEmptyMap(data map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(data[strings.ToLower(strings.TrimSpace(key))]); value != "" {
			return value
		}
	}
	return ""
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func toFloat64(v any) float64 {
	switch value := v.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		f, _ := value.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return f
	default:
		f, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(v)), 64)
		return f
	}
}
