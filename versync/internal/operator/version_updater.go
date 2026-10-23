package operator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go/v3"
)

// VersionUpdater updates the versions in the JSON file
func VersionUpdater(git *gitlab.Client, projectPath string, branch string, artemisID string, pfActivemqID string) error {
	JSONFile := "data/platform_data.json"

	file, _, err := git.RepositoryFiles.GetFile(projectPath, JSONFile, &gitlab.GetFileOptions{
		Ref: new(branch),
	})
	if err != nil {
		return fmt.Errorf("API error to get %s: %w", JSONFile, err)
	}

	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return fmt.Errorf("decoding error: %w", err)
	}

	var data ClusterData

	err = json.Unmarshal(content, &data)
	if err != nil {
		return fmt.Errorf("JSON parsing error: %w", err)
	}

	type ComponentType int

	const (
		ArtemisBroker ComponentType = iota
		PFActiveMQOperator
		ArtemisCloudOperator
	)

	updateJSON := false

	var updates []string

	for i := range data.Clusters {
		cluster := &data.Clusters[i]

		fmt.Printf("Cluster: %s\n", cluster.ID)

		components := []ComponentType{ArtemisBroker, PFActiveMQOperator, ArtemisCloudOperator}

		for _, component := range components {
			var actual, jsonVersion, name string

			var needsUpdate bool

			var err error

			switch component {
			case ArtemisBroker:
				actual, needsUpdate, err = VersionArtemisBroker(git, cluster, branch, pfActivemqID)
				jsonVersion = cluster.ArtemisVersion
				name = "Artemis Broker"

			case PFActiveMQOperator:
				actual, needsUpdate, err = VersionOperatorPFActiveMQ(git, cluster, branch, pfActivemqID)
				jsonVersion = cluster.PfActivemqOperatorVersion
				name = "PF ActiveMQ Operator"

			case ArtemisCloudOperator:
				actual, needsUpdate, err = VersionOperatorArtemis(git, cluster, branch, artemisID)
				jsonVersion = cluster.ArtemisCloudOperatorVersion
				name = "Artemis Cloud Operator"
			}

			if err != nil {
				fmt.Printf(" %s, %s: Error - %v\n", name, cluster.ID, err)
			} else if needsUpdate {
				fmt.Printf(" needs update by: %s, %s: %s to %s\n", name, cluster.ID, jsonVersion, actual)
				updates = append(updates, fmt.Sprintf("%s %s: %s -> %s", cluster.ID, name, jsonVersion, actual))

				switch component {
				case ArtemisBroker:
					data.Clusters[i].ArtemisVersion = actual
				case PFActiveMQOperator:
					data.Clusters[i].PfActivemqOperatorVersion = actual
				case ArtemisCloudOperator:
					data.Clusters[i].ArtemisCloudOperatorVersion = actual
				}

				updateJSON = true
			}
		}
	}

	if updateJSON {
		updatedJSON, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON encoding error: %w", err)
		}

		var commitMsgBuilder strings.Builder
		commitMsgBuilder.WriteString("chore: sync versions with kustomizes\n\n")

		for _, update := range updates {
			commitMsgBuilder.WriteString("- " + update + "\n")
		}

		_, _, updateErr := git.RepositoryFiles.UpdateFile(
			projectPath,
			JSONFile,
			&gitlab.UpdateFileOptions{
				Branch:        new(branch),
				Content:       new(string(updatedJSON)),
				CommitMessage: new(commitMsgBuilder.String()),
			},
		)
		if updateErr != nil {
			return fmt.Errorf("API error to update %s: %w", JSONFile, updateErr)
		}
	}

	return nil
}
