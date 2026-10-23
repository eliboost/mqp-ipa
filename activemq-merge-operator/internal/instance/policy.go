// Package instance showcase the implementation of a policy module
package instance

import (
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/policy"
	activemqv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
)

// All namespaces reasons.
const (
	ReasonInvalidInstanceDefinition     = "InvalidInstanceDefinition"
	ReasonInvalidNamespace              = "InvalidNamespace"
	ReasonInvalidQueueName              = "InvalidQueueName"
	ReasonDuplicateQueue                = "DuplicateQueue"
	ReasonBlockStorageAnnotationMissing = "MissingBlockStorageAnnotation"
	ReasonBlockStorageAnnotationInvalid = "InvalidBlockStorageAnnotation"
	ReasonFileRename                    = "FileRename"
	ReasonPropertiesChange              = "PropertiesChange"
	ReasonJavaArgsChange                = "JavaArgsChange"
	ReasonMismatchingCRName             = "MismatchingCRName"
	ReasonCertificateMissing            = "MissingCertificate"
	ReasonSplunkIndexMissing            = "MissingSplunkIndex"
	ReasonPlatformMismatchInNamespace   = "InvalidPlatformInNamespace"
	ReasonPlatformMismatchInPermission  = "InvalidPlatformInPermission"
	ReasonPlatformMismatchInSplunk      = "InvalidPlatformInSplunk"
	ReasonInvalidQueueConfig            = "InvalidQueueConfig"
	ReasonQueueRenamed                  = "QueueRenamed"
	ReasonMissingESBPFOperatorRole      = "MissingEBPFOperatorRole"
	ReasonInvalidPermission             = "InvalidPermission"
	ReasonInstanceDelete                = "InstanceDelete"
	ReasonMissingOperatorRole           = "MissingOperatorRole"
)

// ChangeChain creates a policy.ChangeChainProvider for the instance merger.
func ChangeChain(config Config) policy.ChangeChain[activemqv1alpha1.PFActiveMQ] {
	return policy.ChangeChain[activemqv1alpha1.PFActiveMQ]{
		CheckInstanceDelete,
		CheckFileRename,
		CheckPropertiesChange,
		CheckJavaArgsChange,
		CheckMismatchingCRName,
		CheckPlatformInNamespace,
		// CheckSplunk(K8sClient{}),
		CheckPlatformInPermissions,
		CheckTargetNamespace(GitlabClient{}),
		CheckIdentity(CesClient{endpoint: config.CesIdentityEndpoint}),
		CheckQueueRename,
		CheckQueueNaming,
		CheckQueueDuplicate,
		CheckESBPFOperator,
		CheckQueuePageFullMessagePolicyConfigChange,
		CheckQueueMaxSizeBytesConfigChange,
		CheckQueueExpiryDelayConfigChange,
		CheckQueuePageLimitBytesConfigChange,
		CheckAllowedPermissions,
		CheckOperatorRole,
	}
}

// Config configures the instance check policy module
type Config struct {
	ClusterInfoURL      string
	ClusterInfoInsecure bool
	CesIdentityEndpoint string
}
