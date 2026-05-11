package service

const (
	DefaultPricingDiscountFactor = 1.0
	MinPricingDiscountFactor     = 0.01
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

func applyPricingDiscountFactor(baseMultiplier, discountFactor float64) float64 {
	base := baseMultiplier
	if base < 0 {
		base = 0
	}
	factor := normalizePricingDiscountFactor(discountFactor)
	return base * factor
}
