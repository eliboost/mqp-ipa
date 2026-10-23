// Package resource contains custom resource related functions
package resource

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
)

func Test_processQueuePermissions(t *testing.T) {
	q := v1alpha1.Queue{
		Name: "mqp.myappl.MY_Queue",
		Permission: v1alpha1.Permission{
			Send:    []string{"a-pf-auth-myappl", "a-pf-auth-otherappl"},
			Consume: []string{"a-pf-auth-myappl"},
			Manage:  []string{"mqp.appl_myappl"},
		},
	}

	permissions := processQueuePermissions(q)

	// 4 configured user permissions + 3 users x 2 derived consume permissions on dlq/expiry + 9 admin permissions
	assert.Len(t, permissions, 4+6+9)

	// admin always gets send, consume and browse on the queue, its dlq and its expiry address
	for _, operation := range []string{"send", "consume", "browse"} {
		for _, suffix := range []string{"", "_DLQ", "_EXPIRY"} {
			assert.Contains(t, permissions, fmt.Sprintf("securityRoles.%q.admin.%s=true", q.Name+suffix, operation))
		}
	}
}

func Test_applyOptionalParameters1(t *testing.T) {
	type args struct {
		props         []string
		optionalProps []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "existing-optional-property",
			args: args{
				props: []string{
					"gracefulShutdownEnabled=true",
					"testProperty=false",
					"journalBufferSize=501760",
				},
				optionalProps: []string{
					"testProperty=true",
				},
			},
			want: []string{
				"gracefulShutdownEnabled=true",
				"testProperty=true",
				"journalBufferSize=501760",
			},
		},
		{
			name: "additional-optional-property",
			args: args{
				props: []string{
					"gracefulShutdownEnabled=true",
					"testProperty=false",
					"journalBufferSize=501760",
				},
				optionalProps: []string{
					"additionalProperty=true",
				},
			},
			want: []string{
				"gracefulShutdownEnabled=true",
				"testProperty=false",
				"journalBufferSize=501760",
				"additionalProperty=true",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := applyOptionalParameters(tt.args.props, tt.args.optionalProps)
			assert.Equal(t, tt.want, props)
		})
	}
}
