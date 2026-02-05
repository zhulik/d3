package iampol

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/zhulik/d3/pkg/json"
	"github.com/zhulik/d3/pkg/s3actions"
)

var (
	ErrInvalidPolicy = errors.New("invalid policy")
)

type Effect string

const (
	EffectAllow Effect = "Allow"
	EffectDeny  Effect = "Deny"
)

var (
	effects = []Effect{ //nolint:gochecknoglobals
		EffectAllow,
		EffectDeny,
	}
)

type IAMPolicy struct {
	ID        string      `json:"Id"`
	Statement []Statement `json:"Statement"`
}

// Statement represents a single statement in an IAM policy.
// it only implements a subset of the full IAM policy statement structure,
// for instance it only supports arrays of Actions and Resources,
// and does not support Conditions or Principals.
type Statement struct {
	Effect   Effect   `json:"Effect"`
	Action   []string `json:"Action"`
	Resource []string `json:"Resource"`
}

func Parse(policyBytes []byte) (*IAMPolicy, error) {
	policy, err := json.Unmarshal[IAMPolicy](policyBytes)
	if err != nil {
		return nil, err
	}

	if policy.ID == "" {
		return nil, fmt.Errorf("%w: missing Id", ErrInvalidPolicy)
	}

	if len(policy.Statement) == 0 {
		return nil, fmt.Errorf("%w: missing Statement", ErrInvalidPolicy)
	}

	for i, stmt := range policy.Statement {
		if !lo.Contains(effects, stmt.Effect) {
			return nil, fmt.Errorf("%w: invalid Effect in Statement %d", ErrInvalidPolicy, i)
		}

		if len(stmt.Action) == 0 {
			return nil, fmt.Errorf("%w: missing Action in Statement %d", ErrInvalidPolicy, i)
		}

		for _, action := range stmt.Action {
			if !lo.Contains(s3actions.Actions, s3actions.Action(action)) {
				return nil, fmt.Errorf("%w: invalid Action in Statement %d, Action %s", ErrInvalidPolicy, i, action)
			}
		}

		if len(stmt.Resource) == 0 {
			return nil, fmt.Errorf("%w: missing Resource in Statement %d", ErrInvalidPolicy, i)
		}

		for _, resource := range stmt.Resource {
			if _, ok := strings.CutPrefix(resource, "arn:aws:s3:::"); !ok {
				return nil, fmt.Errorf("%w: invalid Resource in Statement %d, Resource %s", ErrInvalidPolicy, i, resource)
			}
		}
	}

	return &policy, nil
}
