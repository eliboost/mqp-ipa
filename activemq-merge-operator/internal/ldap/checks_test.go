package ldap

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/merge"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/policy"
)

func TestCheckNewRolesAreValid(t *testing.T) {
	type args[T any] struct {
		roleGetter AppRoleGetter[T]
		change     merge.ParsedChange[T]
	}
	type testCase[T any] struct {
		name string
		args args[T]
		want policy.Results
	}
	tests := []testCase[string]{
		{
			name: "delete_file_case",
			args: args[string]{
				roleGetter: func(_ string) ([]AppRole, error) {
					return nil, nil
				},
				change: merge.ParsedChange[string]{},
			},
			want: policy.Results{},
		},
		{
			name: "valid",
			args: args[string]{
				roleGetter: func(_ string) ([]AppRole, error) {
					return []AppRole{{App: "ces", Role: "linux"}}, nil
				},
				change: merge.ParsedChange[string]{},
			},
			want: policy.Results{},
		},
		{
			name: "invalid",
			args: args[string]{
				roleGetter: func(_ string) ([]AppRole, error) {
					return []AppRole{}, fmt.Errorf("invalid role abc")
				},
				change: merge.ParsedChange[string]{
					NewObj: new(""),
				},
			},
			want: createResult(false, ReasonInvalidRole, "some roles are invalid: invalid role abc"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _ := CheckNewRolesAreValid(tt.args.roleGetter)(context.TODO(), nil, &tt.args.change)
			assert.Equalf(t, tt.want, res, "CheckNewRolesAreValid(%v)(%v)", tt.args.roleGetter, tt.args.change)
		})
	}
}

func createResult(isClose bool, reason string, messages ...string) policy.Results {
	results := policy.Results{}

	for _, msg := range messages {
		results = append(results, policy.Result{
			Reason:  reason,
			Close:   isClose,
			Message: msg,
		})
	}

	return results
}

type checker struct {
	isOk bool
	err  error
}

func (c checker) HasAuthorRole(application, role, username string) (bool, error) {
	return c.isOk, c.err
}

func TestCheckAuthorHasOneRole(t *testing.T) {
	nokChecker := checker{
		isOk: false,
		err:  nil,
	}

	okChecker := checker{
		isOk: true,
		err:  nil,
	}

	errChecker := checker{
		isOk: false,
		err:  fmt.Errorf("failed to get role"),
	}

	type args[T any] struct {
		cli        RoleChecker
		roleGetter AppRoleGetter[T]
		change     merge.ParsedChange[T]
		mr         *merge.Request
	}
	type testCase[T any] struct {
		name    string
		args    args[T]
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}
	tests := []testCase[string]{
		{
			name: "valid",
			args: args[string]{
				cli: okChecker,
				roleGetter: func(s string) ([]AppRole, error) {
					return []AppRole{{App: "", Role: ""}}, nil
				},
				change: merge.ParsedChange[string]{
					NewObj: new(""),
				},
				mr: &merge.Request{Author: ""},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid",
			args: args[string]{
				cli: nokChecker,
				roleGetter: func(s string) ([]AppRole, error) {
					return []AppRole{{App: "app1", Role: "role1"}, {App: "app2", Role: "role1"}}, nil
				},
				change: merge.ParsedChange[string]{
					NewObj: new(""),
				},
				mr: &merge.Request{Author: ""},
			},
			want:    createResult(true, ReasonNotOwner, "author does not have one of the owner roles: `[{App:app1 Role:role1} {App:app2 Role:role1}]`"),
			wantErr: assert.NoError,
		},
		{
			name: "error",
			args: args[string]{
				cli: errChecker,
				roleGetter: func(s string) ([]AppRole, error) {
					return []AppRole{{App: "app1", Role: "role1"}}, nil
				},
				change: merge.ParsedChange[string]{
					NewObj: new(""),
				},
				mr: &merge.Request{Author: ""},
			},
			want:    policy.Results{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := CheckAuthorHasOneRole(tt.args.cli, tt.args.roleGetter)(context.TODO(), tt.args.mr, &tt.args.change)
			tt.wantErr(t, err)

			assert.Equalf(t, tt.want, results, "CheckAuthorHasOneRole(%v, %v)(%v)", tt.args.cli, tt.args.roleGetter, tt.args.change)
		})
	}
}
