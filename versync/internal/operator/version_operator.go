// Package operator contains functions to manage operator versions
package operator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go/v3"
	"gitlab.pnet.ch/middleware/activemq/versync/internal/activemqpf"
	"gitlab.pnet.ch/middleware/activemq/versync/internal/artemis"
)

// ClusterData represents the structure of the JSON data containing cluster information
type ClusterData struct {
	Clusters []Cluster `json:"clusters"`
}

// Cluster represents a single cluster configuration
type Cluster struct {
	ID                          string `json:"id"`
	ArtemisVersion              string `json:"artemisVersion"`
	ArtemisCloudOperatorVersion string `json:"artemisCloudOperatorVersion"`
	PfActivemqOperatorVersion   string `json:"pfActivemqOperatorVersion"`
}

// GetRepoVersions retrieves the versions from the repository
func GetRepoVersions(git *gitlab.Client, projectPath string, branch string, artemisID string, pfActivemqID string) error {
	JSONFile := "data/platform_data.json"

	fmt.Printf("read JSON file: %s\n", JSONFile)

	fmt.Printf("check versions from branch: %s\n", branch)

	file, _, err := git.RepositoryFiles.GetFile(
		projectPath,
		JSONFile,
		&gitlab.GetFileOptions{
			Ref: new(branch),
		},
	)
	if err != nil {
		return fmt.Errorf("API error: %w", err)
	}

	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return fmt.Errorf("decoding error: %w", err)
	}

	contentStr := string(content)

	var data ClusterData

	err = json.Unmarshal([]byte(contentStr), &data)
	if err != nil {
		return fmt.Errorf("JSON parsing error: %w", err)
	}

	for i := range data.Clusters {
		cluster := &data.Clusters[i]

		if _, _, err := VersionOperatorArtemis(git, cluster, branch, artemisID); err != nil {
			return fmt.Errorf("failed to check Artemis operator version for cluster %s: %w", cluster.ID, err)
		}

		if _, _, err := VersionOperatorPFActiveMQ(git, cluster, branch, pfActivemqID); err != nil {
			return fmt.Errorf("failed to check PF ActiveMQ operator version for cluster %s: %w", cluster.ID, err)
		}

		if _, _, err := VersionArtemisBroker(git, cluster, branch, pfActivemqID); err != nil {
			return fmt.Errorf("failed to check Artemis broker version for cluster %s: %w", cluster.ID, err)
		}
	}

	return nil
}

// VersionOperatorArtemis checks and updates the cluster if necessary
func VersionOperatorArtemis(git *gitlab.Client, cluster *Cluster, branch string, projectPath string) (string, bool, error) {
	envPath := "envs/" + cluster.ID

	operatorArtemisActualVersion, err := artemis.GetOperatorVersion(git, projectPath, envPath, branch)
	if err != nil {
		fmt.Printf(" couldn't get Artemis Cloud Operator version for cluster %s: %v\n", cluster.ID, err)
		return "", false, err
	}

	jsonVersion := strings.TrimSpace(cluster.ArtemisCloudOperatorVersion)
	operatorArtemisActualVersion = strings.TrimSpace(operatorArtemisActualVersion)

	needsUpdateOperatorArtemis := jsonVersion != operatorArtemisActualVersion

	return operatorArtemisActualVersion, needsUpdateOperatorArtemis, nil
}

// VersionOperatorPFActiveMQ checks and updates the cluster if necessary
func VersionOperatorPFActiveMQ(git *gitlab.Client, cluster *Cluster, branch string, projectPath string) (string, bool, error) {
	envPath := "overlays/" + cluster.ID

	operatorPFActiveMQActualVersion, err := activemqpf.GetOperatorVersion(git, projectPath, envPath, branch)
	if err != nil {
		fmt.Printf(" couldn't get PF ActiveMQ Operator version for cluster %s: %v\n", cluster.ID, err)
		return "", false, err
	}

	jsonVersion := strings.TrimSpace(cluster.PfActivemqOperatorVersion)
	operatorPFActiveMQActualVersion = strings.TrimSpace(operatorPFActiveMQActualVersion)

	needsUpdateOperatorPFActiveMQ := jsonVersion != operatorPFActiveMQActualVersion

	return operatorPFActiveMQActualVersion, needsUpdateOperatorPFActiveMQ, nil
}

// VersionArtemisBroker checks and updates the cluster if necessary
func VersionArtemisBroker(git *gitlab.Client, cluster *Cluster, branch string, projectPath string) (string, bool, error) {
	envPath := "overlays/" + cluster.ID

	artemisBrokerActualVersion, err := activemqpf.GetArtemisVersion(git, projectPath, envPath, branch)
	if err != nil {
		fmt.Printf(" couldn't get Artemis Broker version for cluster %s: %v\n", cluster.ID, err)
		return "", false, err
	}

	jsonVersion := strings.TrimSpace(cluster.ArtemisVersion)
	artemisBrokerActualVersion = strings.TrimSpace(artemisBrokerActualVersion)

	needsUpdateArtemisBroker := jsonVersion != artemisBrokerActualVersion

	return artemisBrokerActualVersion, needsUpdateArtemisBroker, nil
}
