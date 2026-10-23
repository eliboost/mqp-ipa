// Package resource contains custom resource related functions
package resource

import (
	"fmt"
	"sort"
	"strings"

	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/controller/random"
	corev1 "k8s.io/api/core/v1"
)

// NewMappingSecret returns the corev1.Secret which used to map the key and truststore
func NewMappingSecret(pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) *corev1.Secret {
	keystoreName := fmt.Sprintf(DefaultAcceptorCertmgrOutSecretTpl, pfCR.Name)

	keystorePassword := random.New(32)

	return &corev1.Secret{
		Name:      fmt.Sprintf(DefaultAcceptorMappingSecretTpl, pfCR.Name),
		Namespace: pfCR.Spec.Instance.Namespace,
		Labels: map[string]string{
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQManagedByOperatorLabelKey:        pfactivemqoperatorpnetchv1alpha1.LabelValueTrue,
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNamespaceLabelKey: pfCR.Namespace,
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNameLabelKey:      pfCR.Name,
		},
		StringData: map[string]string{
			"keyStorePath":       fmt.Sprintf("/amq/extra/secrets/%s/keystore.jks", keystoreName),
			"keyStorePassword":   keystorePassword,
			"trustStorePath":     fmt.Sprintf("/amq/extra/secrets/%s/truststore.jks", TrustStoreSecretName),
			"trustStorePassword": "passphrase",
		},
		Type: corev1.SecretTypeOpaque,
	}
}

// NewAuthSecret returns configured v1.ConfigMap authentication login module config resource
func NewAuthSecret(pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ, operationCommonName string) *corev1.Secret {
	return &corev1.Secret{
		Name:      fmt.Sprintf(DefaultAuthSecretTpl, pfCR.Name),
		Namespace: pfCR.Spec.Instance.Namespace,
		Labels: map[string]string{
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQManagedByOperatorLabelKey:        pfactivemqoperatorpnetchv1alpha1.LabelValueTrue,
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNamespaceLabelKey: pfCR.Namespace,
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNameLabelKey:      pfCR.Name,
		},

		StringData: map[string]string{"login.config": `
// a full login.config with the default activemq realm
activemq {
		// ensure the operator can connect to the broker by referencing the existing properties config
		org.apache.activemq.artemis.spi.core.security.jaas.PropertiesLoginModule sufficient
			org.apache.activemq.jaas.properties.user="artemis-users.properties"
			org.apache.activemq.jaas.properties.role="artemis-roles.properties"
			baseDir="/home/jboss/amq-broker/etc";
		
        // a custom LoginModule that will reload from this secret
		org.apache.activemq.artemis.spi.core.security.jaas.TextFileCertificateLoginModule sufficient
			reload=true
			debug=true
			org.apache.activemq.jaas.textfiledn.user="cert-users.properties"
			org.apache.activemq.jaas.textfiledn.role="cert-roles.properties";
		
        // add any other supported LoginModule here
};
`,
			"cert-users.properties": strings.Join(getUserProps(pfCR.Spec, operationCommonName), "\n"),
			"cert-roles.properties": strings.Join(getRolesProps(pfCR.Spec, operationCommonName), "\n"),
		},
	}
}

func getUserProps(spec pfactivemqoperatorpnetchv1alpha1.PFActiveMQSpec, operationCommonName string) []string {
	users := getUniqueUsers(spec.Instance.Queues)

	if len(spec.Roles.Operators) > 0 {
		users = append(users, operationCommonName)
	}

	res := make([]string, 0, len(users))

	for i := range users {
		user := users[i]

		res = append(res, fmt.Sprintf("%s=/.*CN=%s.*/", user, user))
	}

	return res
}

func getRolesProps(spec pfactivemqoperatorpnetchv1alpha1.PFActiveMQSpec, operationCommonName string) []string {
	users := getUniqueUsers(spec.Instance.Queues)
	res := []string{}

	if len(spec.Roles.Operators) > 0 {
		res = append(res, "admin="+operationCommonName)
	}

	for i := range users {
		user := users[i]

		res = append(res, fmt.Sprintf("%s=%s", user, user))
	}

	return res
}

func getUniqueUsers(queues []pfactivemqoperatorpnetchv1alpha1.Queue) []string {
	res := map[string]bool{}

	for i := range queues {
		queue := queues[i]

		for _, users := range queue.GetPermissions() {
			for _, user := range users {
				res[user] = true
			}
		}
	}

	unique := make([]string, 0, len(res))
	for k := range res {
		unique = append(unique, k)
	}

	sort.Strings(unique)

	return unique
}
