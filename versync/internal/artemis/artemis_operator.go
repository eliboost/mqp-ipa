// Package artemis contains functions to manage Artemis operator versions
package artemis

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go/v3"
)

// GetResourceURL reads the kustomization.yaml and extracts only the resource URL
func GetResourceURL(git *gitlab.Client, projectPath, envPath, branch string) (string, error) {
	kustomizationFile := envPath + "/kustomization.yaml"

	file, _, err := git.RepositoryFiles.GetFile(
		projectPath,
		kustomizationFile,
		&gitlab.GetFileOptions{
			Ref: new(branch),
		},
	)
	if err != nil {
		return "", fmt.Errorf("API error: %w", err)
	}

	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return "", fmt.Errorf("decoding error: %w", err)
	}

	contentStr := string(content)

	resourceRegex := regexp.MustCompile(`resources:\s*\n\s*-\s*(https?://[^\s]+)`)
	matches := resourceRegex.FindStringSubmatch(contentStr)

	if len(matches) < 2 {
		return "", errors.New("no resource URL found")
	}

	resourceURL := matches[1]

	return resourceURL, nil
}

// GetOperatorVersion extracts the operator version from the resource URL
func GetOperatorVersion(git *gitlab.Client, projectPath, envPath, branch string) (string, error) {
	resourceURL, err := GetResourceURL(git, projectPath, envPath, branch)
	if err != nil {
		return "", fmt.Errorf("error reading resource URL: %w", err)
	}

	versionRegex := regexp.MustCompile(`/releases/download/([^/]+)/`)
	matches := versionRegex.FindStringSubmatch(resourceURL)

	if len(matches) < 2 {
		return "", errors.New("couldn't extract version from URL")
	}

	version := strings.TrimPrefix(matches[1], "v")

	return version, nil
}

// ProcessEnvironmentFiles processes the environment files
func ProcessEnvironmentFiles(git *gitlab.Client, projectPath, envPath string, branch string) error {
	_, err := GetOperatorVersion(git, projectPath, envPath, branch)
	if err != nil {
		return fmt.Errorf("error getting operator version: %w", err)
	}

	return nil
}
