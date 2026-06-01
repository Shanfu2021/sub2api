package service

func EffectiveAPIUsageCapacity(user *User) (concurrency int, rpm int) {
	if user == nil {
		return 0, 0
	}
	concurrency = user.Concurrency
	rpm = user.RPMLimit
	if user.ParentUserID != nil && !isAgentManagerRole(user.Role) || user.Role == RoleEnterprise {
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
