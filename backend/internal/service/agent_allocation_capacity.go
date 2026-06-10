package service

func EffectiveAPIUsageCapacity(user *User) (concurrency int, rpm int) {
	if user == nil {
		return 0, 0
	}
	concurrency = user.Concurrency
	rpm = user.RPMLimit
	if user.Role == RoleEnterprise {
		if concurrency < 0 {
			concurrency = 0
		}
		return concurrency, rpm
	}
	if user.ParentUserID != nil && !isAgentManagerRole(user.Role) {
		if user.AllocatedConcurrency > 0 {
			concurrency = user.AllocatedConcurrency
		}
		if user.AllocatedRPM > 0 {
			rpm = user.AllocatedRPM
		}
	}
	return concurrency, rpm
}

func effectiveUserAPIUsageCapacity(user *User) (concurrency int, rpm int) {
	return EffectiveAPIUsageCapacity(user)
}
