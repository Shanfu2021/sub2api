package service

const (
	DefaultPricingDiscountFactor = 1.0
	MinPricingDiscountFactor     = 0.01

	PromoDiscountScopeAll          = "all"
	PromoDiscountScopeBalance      = "balance"
	PromoDiscountScopeSubscription = "subscription"
)

func normalizePricingDiscountFactor(v float64) float64 {
	if v < MinPricingDiscountFactor {
		return DefaultPricingDiscountFactor
	}
	return v
}

func NormalizePricingDiscountFactorForRepo(v float64) float64 {
	return normalizePricingDiscountFactor(v)
}

func NormalizePromoDiscountScope(scope string) string {
	switch scope {
	case PromoDiscountScopeBalance, PromoDiscountScopeSubscription:
		return scope
	default:
		return PromoDiscountScopeAll
	}
}

func PromoDiscountAppliesToGroup(scope string, group *Group) bool {
	normalizedScope := NormalizePromoDiscountScope(scope)
	isSubscription := group != nil && group.IsSubscriptionType()
	switch normalizedScope {
	case PromoDiscountScopeBalance:
		return !isSubscription
	case PromoDiscountScopeSubscription:
		return isSubscription
	default:
		return true
	}
}

func applyPricingDiscountFactor(baseMultiplier, discountFactor float64) float64 {
	base := baseMultiplier
	if base < 0 {
		base = 0
	}
	factor := normalizePricingDiscountFactor(discountFactor)
	return base * factor
}
