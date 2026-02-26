package auth

import (
	"context"
	"strings"

	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iampol"
	"github.com/zhulik/d3/pkg/s3actions"
	"github.com/zhulik/d3/pkg/wld"
)

const s3ResourcePrefix = "arn:aws:s3:::"

// Authorizer decides if a user is allowed to perform an S3 action on a resource.
type Authorizer struct {
	ManagementBackend core.ManagementBackend
}

// IsAllowed returns whether the user is allowed to perform the action on the resource.
// key is the S3 resource identifier: bucket name for bucket ops, or "bucket/key" for object ops.
func (a *Authorizer) IsAllowed(
	ctx context.Context, user *core.User, action s3actions.Action, resource string,
) (bool, error) {
	if user == nil {
		// TODO: anonymous access to public buckets
		return false, nil
	}

	if user.Name == "admin" {
		return true, nil
	}

	bindings, err := a.ManagementBackend.GetBindingsByUser(ctx, user.Name)
	if err != nil {
		return false, err
	}

	// First pass: any Deny that matches overrides
	for _, binding := range bindings {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		policy, err := a.ManagementBackend.GetPolicyByID(ctx, binding.PolicyID)
		if err != nil {
			return false, err
		}

		for _, stmt := range policy.Statement {
			if stmt.Effect != iampol.EffectDeny {
				continue
			}

			if a.statementMatches(stmt, action, resource) {
				return false, nil
			}
		}
	}

	// Second pass: any Allow that matches grants access
	for _, binding := range bindings {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		policy, err := a.ManagementBackend.GetPolicyByID(ctx, binding.PolicyID)
		if err != nil {
			return false, err
		}

		for _, stmt := range policy.Statement {
			if stmt.Effect != iampol.EffectAllow {
				continue
			}

			if a.statementMatches(stmt, action, resource) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (a *Authorizer) statementMatches(stmt iampol.Statement, action s3actions.Action, resourceSuffix string) bool {
	// Policy statement's s3:* (All) matches any requested action; otherwise require explicit match
	actionMatches := lo.Contains(stmt.Action, s3actions.All) || lo.Contains(stmt.Action, action)
	if !actionMatches {
		return false
	}

	return lo.ContainsBy(stmt.Resource, func(res string) bool {
		pattern, ok := strings.CutPrefix(res, s3ResourcePrefix)
		if !ok {
			return false
		}

		return wld.Match(pattern, resourceSuffix)
	})
}
