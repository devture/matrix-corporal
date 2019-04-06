package policycheck

import (
	"context"
	"devture-matrix-corporal/corporal/policy"
	"net/http"
)

type PolicyCheckFunc func(*http.Request, context.Context, policy.Policy, policy.Checker) PolicyCheckResponse

type PolicyCheckResponse struct {
	Allow bool

	ErrorCode    string
	ErrorMessage string
}
