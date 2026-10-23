// Package config contains some globally use values
package config

const (
	// GitlabTokenEnvKey env variable key for gitlab token
	GitlabTokenEnvKey = "GITLAB_TOKEN"
	// GitlabAPIURL base url for Gitlab api
	GitlabAPIURL = "https://gitlab.pnet.ch/api/v4"

	// ArtemisPath project id for artemis operator
	ArtemisPath = "middleware/activemq/kube-artemis-operator"
	// PfActivemqPath project id for pf activemq operator
	PfActivemqPath = "middleware/activemq/kube-pf-activemq-operator"
	// DocumentationPath project id for documentation
	DocumentationPath = "middleware/activemq/documentation"

	// BranchName env variable key for branch
	BranchName = "BRANCH_NAME"

	// EnvsRootPath root path for environments
	EnvsRootPath = "envs"
	// OverlaysRootPath root path for overlays
	OverlaysRootPath = "overlays"
)
