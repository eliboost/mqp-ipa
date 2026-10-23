// Package ldap contains ldap related policy checks
package ldap

import (
	"errors"

	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/policy"
)

// All namespaces reasons.
const (
	ReasonInvalidRole = "InvalidRole"
	ReasonNotOwner    = "NotOwner"
)

// AppRole definition
type AppRole struct {
	App  string
	Role string
}

// AppRoleGetter returns a list of the app roles which are validated
type AppRoleGetter[T any] func(T) ([]AppRole, error)

// ChangeChain creates a policy.ChangeChainProvider for the instance merger.
func ChangeChain[T any](c Config[T]) (policy.ChangeChain[T], error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	cli := New(c.BaseURL, basicAuth{
		username: c.Username,
		password: c.Password,
	})

	return policy.ChangeChain[T]{
		CheckNewRolesAreValid(c.AppRoleGetter),
		CheckAuthorHasOneRole(cli, c.AppRoleGetter),
	}, nil
}

// Config configures the ldap policies
type Config[T any] struct {
	Username      string
	Password      string
	BaseURL       string
	AppRoleGetter AppRoleGetter[T]
}

// Validate validates filepath config.
func (c Config[T]) Validate() error {
	if c.Username == "" || c.Password == "" {
		return errors.New("username and password must not be empty")
	}

	return nil
}
