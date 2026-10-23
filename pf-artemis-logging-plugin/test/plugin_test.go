package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

func TestSchemaRegistryPlugin(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	hostIP := os.Getenv("HOST_IP")
	hostname := "localhost"

	if hostIP != "" {
		hostname = hostIP
	}

	c, err := compose.NewDockerCompose("docker-compose.yml")
	assert.NoError(t, err, "NewDockerComposeAPI()")

	t.Cleanup(func() {
		assert.NoError(t, c.Down(context.Background(), compose.RemoveOrphans(true), compose.RemoveImagesLocal), "c.Down()")
	})

	err = c.Up(ctx, compose.Wait(true))
	assert.NoError(t, err, "Up()")

	container, err := c.ServiceContainer(ctx, "activemq-artemis")
	assert.NoError(t, err, "ServiceContainer()")

	endpoint, err := container.PortEndpoint(ctx, "8161", "")
	assert.NoError(t, err, "PortEndpoint()")

	splits := strings.Split(endpoint, ":")

	srRandomHostPort := splits[1]

	listSubjects := fmt.Sprintf("http://%s:%s/console", hostname, srRandomHostPort)

	checkGetRequest(t, listSubjects, http.StatusOK)

	_, _, err = container.Exec(ctx, []string{"bin/artemis", "producer", "--user=myuser", "--password=mypassword", "--message-count=1"})
	assert.NoError(t, err, "ProduceMessage()")

	logs, err := container.Logs(ctx)
	require.NoError(t, err)

	allLogs, err := io.ReadAll(logs)
	require.NoError(t, err)

	assert.Equal(t, true, strings.Contains(string(allLogs), "[PFEventLogging] Message produced"))

	cancel()
}

func checkGetRequest(t *testing.T, url string, httpStatus int) {
	res, err := http.Get(url)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, httpStatus, res.StatusCode)
}
