package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

func TestArtemisMonitor(t *testing.T) {
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

	container, err := c.ServiceContainer(ctx, "artemis-monitor")
	assert.NoError(t, err, "ServiceContainer()")

	endpoint, err := container.PortEndpoint(ctx, "8080", "")
	assert.NoError(t, err, "PortEndpoint()")

	splits := strings.Split(endpoint, ":")
	randomHostPort := splits[1]

	var lastCount float64

	// Wait until messages consumed total >= 10
	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://%s:%s/actuator/prometheus",
			hostname, randomHostPort))
		if err != nil {
			return false
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}

		re := regexp.MustCompile(`monitor_messages_consumed_total\{application="artemis-monitor"\}\s+([0-9.]+)`)
		matches := re.FindStringSubmatch(string(body))
		if len(matches) != 2 {
			return false
		}

		lastCount, err = strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return false
		}

		return lastCount >= 10
	}, 60*time.Second, 1*time.Second, "metric did not reach required value")

	assert.GreaterOrEqual(t, lastCount, float64(10))

	cancel()
}
