package ldap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_HasAuthorRole(t *testing.T) {
	username, _ := os.LookupEnv("LDAP_USERNAME")
	password, _ := os.LookupEnv("LDAP_PASSWORD")

	if username == "" || password == "" {
		t.Skip("Please set LDAP_USERNAME and LDAP_PASSWORD to run this test")
	}

	c := New("https://pfdir2.pnet.ch:2636", basicAuth{username, password})

	ok, err := c.HasAuthorRole("middleware", "mqp_other", "schmidomin")
	assert.NoError(t, err)
	assert.True(t, ok)

	ok2, err2 := c.HasAuthorRole("ces", "admin", "abc")
	assert.NoError(t, err2)
	assert.False(t, ok2)

}
