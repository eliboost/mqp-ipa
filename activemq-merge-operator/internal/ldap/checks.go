package ldap

import (
	"context"
	"fmt"

	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/merge"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/policy"
)

// CheckNewRolesAreValid is a policy to ensure that added roles are valid.
func CheckNewRolesAreValid[T any](roleGetter AppRoleGetter[T]) policy.ChangeCheckFunc[T] {
	return func(_ context.Context, _ *merge.Request, c *merge.ParsedChange[T]) (policy.Results, error) {
		if c.NewObj == nil {
			return policy.Results{}, nil
		}

		_, err := roleGetter(*c.NewObj)
		if err != nil {
			return policy.Results{
				{
					Reason:  ReasonInvalidRole,
					Message: fmt.Sprintf("some roles are invalid: %s", err),
					Close:   false,
					Path:    c.FilePath,
				},
			}, nil
		}

		return policy.Results{}, nil
	}
}

// CheckAuthorHasOneRole is a policy to ensure that only owners can change the configuration.
//
// Policy failure closes merge request.
func CheckAuthorHasOneRole[T any](cli RoleChecker, roleGetter AppRoleGetter[T]) policy.ChangeCheckFunc[T] {
	return func(_ context.Context, mr *merge.Request, c *merge.ParsedChange[T]) (policy.Results, error) {
		object := c.OldObj

		if c.Operation == merge.ChangeOperationCreate {
			object = c.NewObj
		}

		roles, _ := roleGetter(*object)

		for _, role := range roles {
			hasRole, err := cli.HasAuthorRole(role.App, role.Role, mr.Author)
			if err != nil {
				return policy.Results{}, err
			}

			if hasRole {
				return policy.Results{}, nil
			}
		}

		return policy.Results{
			{
				Reason:  ReasonNotOwner,
				Message: fmt.Sprintf("author does not have one of the owner roles: `%+v`", roles),
				Close:   true,
				Path:    c.FilePath,
			},
		}, nil
	}
}
