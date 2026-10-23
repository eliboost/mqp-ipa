package v1alpha1

import (
	"reflect"
	"unicode"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="The state of the resource"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="The age of the resource"

// PFActiveMQ is the Schema for the pfactivemqs API
type PFActiveMQ struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PFActiveMQSpec   `json:"spec,omitempty"`
	Status PFActiveMQStatus `json:"status,omitempty"`
}

// PFActiveMQStatus defines the observed state of PFActiveMQ
type PFActiveMQStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions represents the observations of a PFActiveMQ's current state.
	// Known .status.conditions.type are:
	//    Ready
	//    LoggingConfigReady
	//    CertResourcesReady
	//    CertSecretResourceReady
	//    ActiveMQBrokersReady
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	LastAppliedStorage string             `json:"lastAppliedStorage,omitempty"`
}

// +kubebuilder:object:root=true

// PFActiveMQList contains a list of PFActiveMQ
type PFActiveMQList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PFActiveMQ `json:"items"`
}

// PFActiveMQSpec defines the desired state of PFActiveMQ
type PFActiveMQSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// TeamMail email address from responsible team
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._%+-]+@postfinance.ch$`
	TeamMail string `json:"teamMail"`
	// Roles defines the owner and operators of this Instance
	// +kubebuilder:validation:Required
	Roles Roles `json:"roles"`
	// Instance abstracts the activemq artemis Instance
	// +kubebuilder:validation:Required
	Instance Instance `json:"instance"`
}

// Roles type matches the activemq artemis Roles
type Roles struct {
	// Owner of the mqp instance
	// +kubebuilder:validation:Required
	Owner string `json:"owner"`
	// Operators defines the roles that are allowed to manage the instance
	// +kubebuilder:validation:Optional
	Operators []string `json:"operators,omitempty"`
}

// Instance type matches the activemq artemis Instance
type Instance struct {
	// Namespace target namespace of the instance
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
	// Replicas number of brokers
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=2
	Replicas int32 `json:"replicas,omitempty"`
	// Storage size of the block storage of the instance - should match defined size of the block storage defined for the namespace
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^\d+(\.\d+)?[KMGT]i$`
	Storage string `json:"storage,omitempty"`
	// Resources kubernetes resources definition
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Queues definition of the queues
	// +kubebuilder:validation:Optional
	Queues []Queue `json:"queues,omitempty"`
	// SplunkIndex application index the log should be forwarded to
	// +kubebuilder:validation:Required
	SplunkIndex string `json:"splunkIndex"`
	// LogLevel of the activemq artemis broker e.g. INFO, DEBUG, TRACE
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation::Enum=INFO;DEBUG;TRACE
	LogLevel string `json:"logLevel,omitempty"`
	// Properties broker properties for experimental usage
	// +kubebuilder:validation:Optional
	Properties []string `json:"properties,omitempty"`
	// JavaArgs java arguments for experimental usage
	// +kubebuilder:validation:Optional
	JavaArgs []string `json:"javaArgs,omitempty"`
}

// Queue defines the queues of the activemq broker
type Queue struct {
	// Name of the queue
	// Full-Format: <resource-type>.<application>.[<identifier>_]<subject>
	// For example: mqp.esbpf.In_Queue_1_Test, mqp.myapp.my-first-mqp-demo
	// <resource-type>: mqp
	// <application>: [a-z0-9-]+ letters, digits, hyphens
	// <identifier>: (In_Queue|Out_Queue) is optional
	// <subject>: [a-zA-Z0-9-]+(_[a-zA-Z0-9-]+)*
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^mqp\.[a-z0-9-]+\.[a-zA-Z0-9-]+(_[a-zA-Z0-9-]+)*$`
	Name string `json:"name"`
	// Permission used to set permission for (technical) users. See https://activemq.apache.org/components/artemis/documentation/latest/security.html#role-based-security-for-addresses
	// +kubebuilder:validation:Required
	Permission Permission `json:"permission"`
	// Config used to set queue specific configuration like maxSizeBytes, pageLimitBytes, pageFullMessagePolicy and expiryDelay. See https://activemq.apache.org/components/artemis/documentation/latest/address-model.html#address-settings
	Config *Config `json:"config,omitempty"`
}

type Config struct {
	// MaxSizeBytes defines the maximum size of the queue in bytes. When the limit is reached, the configured pageFullMessagePolicy will be applied. A value of -1 means no limit, this is the default value. See https://artemis.apache.org/components/artemis/documentation/latest/paging.html#paging-mode
	// +kubebuilder:validation:Optional
	MaxSizeBytes int `json:"maxSizeBytes,omitempty"`
	// AddressFullMessagePolicy defines the policy to apply when the maxSizeBytes limit is reached. FAIL will reject new messages, DROP will drop old messages. See https://artemis.apache.org/components/artemis/documentation/latest/paging.html#paging-mode
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=PAGE;FAIL;DROP;BLOCK
	AddressFullMessagePolicy string `json:"addressFullMessagePolicy,omitempty"`
	// PageLimitBytes defines the size of the in-memory buffer for messages. When the limit is reached, messages will be paged to disk. See https://artemis.apache.org/components/artemis/documentation/latest/paging.html#paging-mode
	// +kubebuilder:validation:Optional
	PageLimitBytes int `json:"pageLimitBytes,omitempty"`
	// PageFullMessagePolicy defines the policy to apply when the pageLimitBytes is reached. FAIL will reject new messages, DROP will drop old messages. See https://artemis.apache.org/components/artemis/documentation/latest/paging.html#paging-mode
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=FAIL;DROP
	PageFullMessagePolicy string `json:"pageFullMessagePolicy,omitempty"`
	// ExpiryDelay defines the time in milliseconds after which a message will expire. See https://artemis.apache.org/components/artemis/documentation/latest/message-expiry.html
	// +kubebuilder:validation:Optional
	ExpiryDelay int `json:"expiryDelay,omitempty"`
	// RedeliveryDelay defines the time in milliseconds the broker will wait before redelivering the messages.
	// +kubebuilder:validation:Optional
	RedeliveryDelay int `json:"redeliveryDelay,omitempty"`
	// RedeliveryMultiplier defines the multiplier the last delay time a message is redelivered will be multiplied with.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)?$`
	RedeliveryMultiplier string `json:"redeliveryMultiplier,omitempty"`
	// MaxRedeliveryDelay defines the max value the delay can become.
	// +kubebuilder:validation:Optional
	MaxRedeliveryDelay int `json:"maxRedeliveryDelay,omitempty"`
	// MaxDeliveryAttempts defines the max attempts to try a redelivery.
	// +kubebuilder:validation:Optional
	MaxDeliveryAttempts int `json:"maxDeliveryAttempts,omitempty"`
	// RedistributionDelay defines the delay in milliseconds between detecting the need for redistribution and actually attempting redistribution.
	// A delay of zero means the messages will be immediately redistributed. A value of -1 signifies that messages will never be redistributed.
	// +kubebuilder:validation:Optional
	RedistributionDelay int `json:"redistributionDelay,omitempty"`
}

// Permission type matches the activemq artemis permissions
// See https://activemq.apache.org/components/artemis/documentation/latest/security.html#role-based-security-for-addresses
type Permission struct {
	// List of users with createAddress permission on this queue
	CreateAddress []string `json:"createAddress,omitempty"`
	// List of users with deleteAddress permission on this queue
	DeleteAddress []string `json:"deleteAddress,omitempty"`
	// List of users with createDurableQueue permission on this queue
	CreateDurableQueue []string `json:"createDurableQueue,omitempty"`
	// List of users with deleteDurableQueue permission on this queue
	DeleteDurableQueue []string `json:"deleteDurableQueue,omitempty"`
	// List of users with createNonDurableQueue permission on this queue
	CreateNonDurableQueue []string `json:"createNonDurableQueue,omitempty"`
	// List of users with deleteNonDurableQueue permission on this queue
	DeleteNonDurableQueue []string `json:"deleteNonDurableQueue,omitempty"`
	// List of users with send permission on this queue
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Send []string `json:"send"`
	// List of users with consume permission on this queue
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Consume []string `json:"consume"`
	// List of users with browse permission on this queue
	Browse []string `json:"browse,omitempty"`
	// List of users with manage permission on this queue
	Manage []string `json:"manage,omitempty"`
	// List of users with view permission on this queue
	View []string `json:"view,omitempty"`
	// List of users with edit permission on this queue
	Edit []string `json:"edit,omitempty"`
}

// GetPermissions returns the permission struct as map. Struct fields (=operation) are used as key.
func (q *Queue) GetPermissions() map[string][]string {
	res := map[string][]string{}

	v := reflect.ValueOf(q.Permission)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.String {
			users := field.Interface().([]string)

			if len(users) == 0 {
				continue
			}

			for _, user := range users {
				operation := firstLetterToLower(v.Type().Field(i).Name)

				res[operation] = append(res[operation], user)
			}
		}
	}

	return res
}

func firstLetterToLower(s string) string {
	if s == "" {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])

	return string(r)
}

//nolint:gochecknoinits // created from Kubebuilder
func init() {
	SchemeBuilder.Register(&PFActiveMQ{}, &PFActiveMQList{})
}
