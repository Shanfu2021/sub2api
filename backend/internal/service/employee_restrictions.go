package service

import infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

var ErrEmployeeFeatureRestricted = infraerrors.Forbidden("EMPLOYEE_FEATURE_RESTRICTED", "employee accounts cannot use this feature")

func IsEmployeeRole(role string) bool {
	return role == RoleEmployee
}
