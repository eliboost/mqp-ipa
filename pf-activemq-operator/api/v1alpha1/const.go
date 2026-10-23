package v1alpha1

// Important condition types.
const (
	// PFActiveMQConditionTypeReady indicates that the activeMq artemis brokers are configured
	// correctly and all required resources are set up.
	PFActiveMQConditionTypeReady = "Ready"

	// PFActiveMQConditionTypeCertResourcesReady indicates that the ingress tls certs
	// are configured correctly.
	PFActiveMQConditionTypeCertResourcesReady = "CertResourcesReady"

	// PFActiveMQConditionTypeLoggingConfigReady indicates that the logging property config map is ready
	PFActiveMQConditionTypeLoggingConfigReady = "LoggingConfigReady"

	// PFActiveMQConditionTypeAuthSecretReady indicates that the auth login module secret is ready
	PFActiveMQConditionTypeAuthSecretReady = "AuthSecretReady"

	// PFActiveMQConditionTypeCertSecretResourcesReady indicates that the secret for cert
	// mapping exists.
	PFActiveMQConditionTypeCertSecretResourcesReady = "CertSecretResourcesReady"

	// PFActiveMQConditionTypeActiveMQBrokersReady indicates that the active mq brokers are
	// ready and healthy.
	PFActiveMQConditionTypeActiveMQBrokersReady = "ActiveMQBrokersReady"

	// PFActiveMQConditionTypeStorageReady indicates that the PVC storage has been
	// successfully resized to the desired size.
	PFActiveMQConditionTypeStorageReady = "StorageReady"
)

const (
	// PFActiveMQManagedByOperatorLabelKey used to watch operator created objects
	PFActiveMQManagedByOperatorLabelKey = "pf-activemq-operator.pnet.ch/operator-managed"
	// PFActiveMQPrimaryResourceNamespaceLabelKey used to identify created extra objects
	PFActiveMQPrimaryResourceNamespaceLabelKey = "pf-activemq-operator.pnet.ch/primary-resource-namespace"
	// PFActiveMQPrimaryResourceNameLabelKey used to identify created extra objects
	PFActiveMQPrimaryResourceNameLabelKey = "pf-activemq-operator.pnet.ch/primary-resource-name"

	// PFActiveMQRunLocalLabelKey only used for local testing, switch image and storage class
	PFActiveMQRunLocalLabelKey = "pf-activemq-operator.pnet.ch/run-local"

	// LabelValueTrue is the string value "true" used for label values
	LabelValueTrue = "true"
)

// All existing condition reasons.
const (
	ReasonReady       = "Ready"
	ReasonMismatch    = "Mismatch"
	ReasonMissing     = "Missing"
	ReasonTerminating = "Terminating"
	ReasonUnknown     = "Unknown"
)
