// Package resource contains custom resource related functions
package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
)

func Test_getUniqueUsers(t *testing.T) {
	queues := []pfactivemqoperatorpnetchv1alpha1.Queue{
		{
			Name: "a",
			Permission: pfactivemqoperatorpnetchv1alpha1.Permission{
				Send: []string{
					"userB",
					"userA",
				},
				Consume: []string{
					"userB",
					"userA",
				},
				Browse: []string{
					"userC",
				},
			},
		},
		{
			Name: "b",
			Permission: pfactivemqoperatorpnetchv1alpha1.Permission{
				Consume: []string{
					"userD",
					"userA",
				},
				CreateDurableQueue: []string{
					"userC",
				},
			},
		},
	}

	users := getUniqueUsers(queues)

	assert.Equal(t, []string{"userA", "userB", "userC", "userD"}, users)
}
