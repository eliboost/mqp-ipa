package instance

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_makeClusterDefinition(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:     "valid_cluster_definition",
			filepath: "../../test/instances/e1-k8s-pfnet-a/e1-mqp-egw-a/valid-sample.yaml",
			wantErr:  assert.NoError,
		},
		{
			name:     "invalid_cluster_definition",
			filepath: "../../test/instances/e1-k8s-pfnet-a/e1-mqp-egw-a/invalid-sample.yaml",
			wantErr:  assert.Error,
		},
	}

	for _, test := range tests {
		data, err := os.ReadFile(filepath.Clean(test.filepath))

		if err != nil {
			t.Errorf("Failed to load file: %s", err.Error())
		}

		t.Run(test.name, func(t *testing.T) {
			_, err := makeInstance(test.filepath, data)
			if !test.wantErr(t, err, fmt.Sprintf("makeInstance(%v, %v)", test.filepath, data)) {
				return
			}

		})

	}
}
