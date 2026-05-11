package service

func resolveImageRateMultiplier(apiKey *APIKey, effectiveGroupMultiplier float64, discountFactor float64) float64 {
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.ImageRateIndependent {
		if apiKey.Group.ImageRateMultiplier < 0 {
			return 0
		}
		return applyPricingDiscountFactor(apiKey.Group.ImageRateMultiplier, discountFactor)
	}
	return effectiveGroupMultiplier
}
