package security

import (
	"context"
	"fmt"
	"slices"
)

type SimpleGuard struct{}

func NewSimpleGuard() *SimpleGuard {
	return &SimpleGuard{}
}

func (g *SimpleGuard) Check(_ context.Context, in CheckInput) error {
	sec := in.Tool.Security
	if sec == nil {
		return nil
	}

	if sec.RequiredRole != "" {
		if in.User == nil {
			return fmt.Errorf("%w: role %q required", ErrAccessDenied, sec.RequiredRole)
		}
		if in.User.Role != sec.RequiredRole && !slices.Contains(in.User.Roles, sec.RequiredRole) {
			return fmt.Errorf("%w: role %q required", ErrAccessDenied, sec.RequiredRole)
		}
	}

	if len(sec.RequiredAnyRole) > 0 {
		if in.User == nil {
			return fmt.Errorf("%w: one of roles is required", ErrAccessDenied)
		}

		for _, role := range sec.RequiredAnyRole {
			if in.User.Role == role || slices.Contains(in.User.Roles, role) {
				return nil
			}
		}
		return fmt.Errorf("%w: none of required roles matched", ErrAccessDenied)
	}

	return nil
}
