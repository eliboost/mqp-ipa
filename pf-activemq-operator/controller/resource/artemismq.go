// Package resource contains custom resource related functions
package resource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	activemqartemisv1beta1 "github.com/arkmq-org/activemq-artemis-operator/v2/api/v1beta1"
	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dlqTemplate    = "%s_DLQ"
	expiryTemplate = "%s_EXPIRY"
	adminRole      = "admin"
)

// ArtemisConfig static configs to generate artemis resource
type ArtemisConfig struct {
	IngressDomain string
	Image         string
	InitImage     string
}

func (c ArtemisConfig) getEnv() string {
	return strings.Split(c.IngressDomain, "-")[0]
}

// NewArtemis returns a new instance of pfactivemqoperatorpnetchv1alpha1.PFActiveMQ
func NewArtemis(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ, cfg ArtemisConfig) *activemqartemisv1beta1.ActiveMQArtemis {
	l := log.FromContext(ctx)
	imageName := cfg.Image
	initImageName := cfg.InitImage
	// Get version from artemis image tag
	reFullVersion := regexp.MustCompile(`\d+\.\d+\.\d+`)
	artemisVersion := reFullVersion.FindString(imageName)
	// Compare major and minor version of images
	reMajorMinorVersion := regexp.MustCompile(`\d+\.\d+\.`)
	if reMajorMinorVersion.FindString(initImageName) != reMajorMinorVersion.FindString(imageName) {
		l.Error(errors.New("image versions do not match"), "artemis image and init image version do not match")
		os.Exit(1)
	}

	storageClass := "vsphere-csi"
	affinityConfig := activemqartemisv1beta1.AffinityConfig{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"ActiveMQArtemis": pfCR.Name,
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}

	if isLocal, ok := pfCR.Labels[pfactivemqoperatorpnetchv1alpha1.PFActiveMQRunLocalLabelKey]; ok && isLocal == pfactivemqoperatorpnetchv1alpha1.LabelValueTrue {
		imageName = cfg.Image
		initImageName = cfg.InitImage
		storageClass = "local-path"
		affinityConfig = activemqartemisv1beta1.AffinityConfig{}
	}

	messageMigration := new(true)
	podDisruptionBudget := &policyv1.PodDisruptionBudgetSpec{
		MinAvailable: &intstr.IntOrString{IntVal: 1},
	}

	if pfCR.Spec.Instance.Replicas < 2 {
		// Disable messageMigration if it is a single broker instance
		messageMigration = new(false)
		// Disable PodDisruptionBudget (PDB) if it is a single broker instance
		podDisruptionBudget = nil
	}

	sslSecretName := fmt.Sprintf(DefaultAcceptorMappingSecretTpl, pfCR.Name)

	additionalJavaArgs := ""

	if len(pfCR.Spec.Instance.JavaArgs) > 0 {
		additionalJavaArgs = " " + strings.Join(pfCR.Spec.Instance.JavaArgs, " ")
	}

	artemis := &activemqartemisv1beta1.ActiveMQArtemis{
		Name:      pfCR.Name,
		Namespace: pfCR.Spec.Instance.Namespace,
		Labels: map[string]string{
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQManagedByOperatorLabelKey:        "true",
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNamespaceLabelKey: pfCR.Namespace,
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNameLabelKey:      pfCR.Name,
		},
		Spec: activemqartemisv1beta1.ActiveMQArtemisSpec{
			DeploymentPlan: activemqartemisv1beta1.DeploymentPlanType{
				Image:     imageName,
				InitImage: initImageName,
				//ImagePullSecrets:   nil,
				Size:               new(pfCR.Spec.Instance.Replicas),
				RequireLogin:       false,
				PersistenceEnabled: true,
				JournalType:        "aio",
				MessageMigration:   messageMigration,
				Resources:          pfCR.Spec.Instance.Resources,
				Storage: activemqartemisv1beta1.StorageType{
					Size:             pfCR.Spec.Instance.Storage,
					StorageClassName: storageClass,
				},
				// TopologySpreadConstraints: nil,
				JolokiaAgentEnabled: true,
				// ManagementRBACEnabled:     false,
				ExtraMounts: activemqartemisv1beta1.ExtraMountsType{
					Secrets: []string{
						fmt.Sprintf(DefaultAcceptorCertmgrOutSecretTpl, pfCR.Name),
						fmt.Sprintf(DefaultAuthSecretTpl, pfCR.Name),
						TrustStoreSecretName,
					},
					ConfigMaps: []string{
						fmt.Sprintf(DefaultLoggingConfigMapTpl, pfCR.Name),
					},
				},
				//Clustered:           nil,
				PodSecurity:  activemqartemisv1beta1.PodSecurityType{},
				StartupProbe: nil,
				LivenessProbe: &corev1.Probe{
					FailureThreshold: 10,
				},
				ReadinessProbe: &corev1.Probe{
					FailureThreshold: 10,
				},
				EnableMetricsPlugin: new(false),
				// Tolerations:         nil,
				Labels: map[string]string{
					"env":      strings.ToUpper(cfg.getEnv()),
					"platform": strings.Split(pfCR.Name, "-")[0],
					"app":      "activemq",
				},
				// NodeSelector: nil,
				Affinity: affinityConfig,
				PodSecurityContext: &corev1.PodSecurityContext{
					FSGroup:   ptr.To[int64](1001),
					RunAsUser: ptr.To[int64](1001),
				},
				Annotations: map[string]string{
					"prometheus.io/scheme":     "http",
					"prometheus.io/scrape":     "true",
					"prometheus.io/port":       "8175", // custom metrics binding => used to scrape without clientAuth
					"collectord.io/output":     "splunk",
					"collectord.io/logs-index": pfCR.Spec.Instance.SplunkIndex,
				},
				PodDisruptionBudget: podDisruptionBudget,
				// RevisionHistoryLimit:      nil,
				// ContainerSecurityContext:  nil,
				// ExtraVolumes:              nil,
				// ExtraVolumeMounts:         nil,
				// ExtraVolumeClaimTemplates: nil,
			},
			Acceptors: []activemqartemisv1beta1.AcceptorType{
				{
					Name:       DefaultAcceptorName,
					Port:       61617,
					Protocols:  "CORE,AMQP",
					SSLEnabled: true,
					SSLSecret:  fmt.Sprintf(DefaultAcceptorMappingSecretTpl, pfCR.Name),
					// EnabledCipherSuites:               "",
					// EnabledProtocols:                  "",
					NeedClientAuth: true,
					// WantClientAuth:                    false,
					VerifyHost: false,
					// SSLProvider:                       "",
					// SNIHost:                           "",
					Expose:     true,
					ExposeMode: &activemqartemisv1beta1.ExposeModes.Ingress,
					// AnycastPrefix:                     "",
					// MulticastPrefix:                   "",
					// ConnectionsAllowed:                0,
					// AMQPMinLargeMessageSize:           0,
					// SupportAdvisory:                   nil,
					// SuppressInternalManagementObjects: nil,
					BindToAllInterfaces: new(true),
					// KeyStoreProvider:                  "",
					// TrustStoreType:                    "",
					// TrustStoreProvider:                "",
					// It supports the following variables: $(CR_NAME), $(CR_NAMESPACE), $(BROKER_ORDINAL), $(ITEM_NAME), $(RES_NAME) and $(INGRESS_DOMAIN)
					IngressHost: "$(CR_NAME)-$(ITEM_NAME)-$(BROKER_ORDINAL).$(CR_NAMESPACE).$(INGRESS_DOMAIN)",
					// TrustSecret: nil,
				},
			},
			// Connectors:        nil,
			Console: activemqartemisv1beta1.ConsoleType{
				Name:          "console",
				Expose:        true,
				ExposeMode:    &activemqartemisv1beta1.ExposeModes.Ingress,
				SSLEnabled:    true,
				SSLSecret:     sslSecretName,
				UseClientAuth: true,
				IngressHost:   "$(CR_NAME)-$(ITEM_NAME)-$(BROKER_ORDINAL).$(CR_NAMESPACE).$(INGRESS_DOMAIN)",
				TrustSecret:   &sslSecretName,
			},
			Version:          artemisVersion,
			BrokerProperties: processBrokerProperties(pfCR),
			Env: []corev1.EnvVar{
				{
					Name:  "AMQ_EXTRA_ARGS",
					Value: "--relax-jolokia",
				},
				{
					Name:  "JAVA_ARGS_APPEND",
					Value: "-Dwebconfig.bindings.artemis.sslAutoReload=true -XX:MaxRAMPercentage=60" + additionalJavaArgs, // https://gitlab.pnet.ch/middleware/activemq/architecture/-/issues/38
				},
				{
					Name:  "TZ",
					Value: "Europe/Zurich",
				},
				{
					Name:  "LANG",
					Value: "en_US.UTF-8",
				},
			},
			IngressDomain: cfg.IngressDomain,
			// ResourceTemplates: nil,
		},
	}

	return artemis
}

func processBrokerProperties(pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) []string {
	core := []string{
		"globalMaxSizePercentOfJvmMaxMemory=40", // we reduce the default value of 50 so we can survive message redistribution with almost completely used global-max-size (only applied after restart) https://activemq.apache.org/components/artemis/documentation/latest/perf-tuning.html#tuning-the-vm / https://issues.apache.org/jira/browse/ARTEMIS-1131
		"gracefulShutdownEnabled=true",          // https://github.com/arkmq-org/activemq-artemis-operator/discussions/814
		"gracefulShutdownTimeout=20000",         // default k8s shutdown 30s
		// "journalBufferSize=501760",         // default 490KB -> kann weg? -> currently not available as property - leads to apply error in artemis operator
		// "journalBufferTimeout=500000",      // default 500000 -> kann weg? -> currently not available as property - leads to apply error in artemis operator
		"journalFileSize=10485760", // default 10MB -> kann weg?
		// "journalMaxIo=4096",                // default 4096 bei ASYNCIO -> kann weg? -> currently not available as property - leads to apply error in artemis operator
		"journalMinFiles=2",                // default 2 -> kann weg?
		"journalPoolFiles=4",               // default -1 -> we use atm 4, will maybe change later https://github.com/apache/activemq-artemis/commit/974bdd622eabba83b467745ad5176c6cbf8eae71
		"journalSyncNonTransactional=true", // default true -> kann weg?
		"journalSyncTransactional=true",    // default true -> kann weg?
		"maxDiskUsage=60",                  // https://gitlab.pnet.ch/middleware/activemq/architecture/-/issues/38
		"threadPoolMaxSize=30",             // default 30 -> kann weg?
		"transactionTimeout=300000",        // default 300000 -> kann weg?
	}

	props := slices.Concat(
		core,
		processAcceptorConfigs(),
		processAddressAndQueues(pfCR.Spec.Instance.Queues),
		processSecurityRoles(pfCR.Spec.Instance),
		processBrokerPluginConfig(),
	)

	props = applyOptionalParameters(props, pfCR.Spec.Instance.Properties)

	return props
}

func processBrokerPluginConfig() []string {
	res := []string{
		"brokerPlugins.\"ch.postfinance.activemq.artemis.logging.plugin.PFActiveMQLoggingPlugin.class\"" +
			".init=" +
			"LOG_ALL_EVENTS=false," +
			"LOG_DETAILS=false," +
			"LOG_CONNECTION_EVENTS=false," +
			"LOG_SESSION_EVENTS=false," +
			"LOG_ROUTE_EVENTS=false," +
			"LOG_CONSUMER_EVENTS=true," +
			"LOG_DELIVERING_EVENTS=true," +
			"LOG_SENDING_EVENTS=true," +
			"LOG_INTERNAL_EVENTS=false",
	}

	return res
}

func processSecurityRoles(pfCR pfactivemqoperatorpnetchv1alpha1.Instance) []string {
	//nolint:prealloc // do not preallocate
	res := []string{
		// allow cli queue stat
		// "securityRoles.\"activemq.management.#\".myrole.manage=true",
		// "securityRoles.\"activemq.management.#\".myrole.createNonDurableQueue=true",
		// "securityRoles.\"activemq.management.#\".myrole.createAddress=true",
		// "securityRoles.\"activemq.management.#\".myrole.consume=true",
		// "securityRoles.\"activemq.management.#\".myrole.send=true",
	}

	for i := range pfCR.Queues {
		res = append(res, processQueuePermissions(pfCR.Queues[i])...)
	}

	return res
}

func processQueuePermissions(q pfactivemqoperatorpnetchv1alpha1.Queue) []string {
	res := []string{}
	dlqName := fmt.Sprintf(dlqTemplate, q.Name)
	expiryName := fmt.Sprintf(expiryTemplate, q.Name)
	allUsers := make(map[string]bool)

	for operation, users := range q.GetPermissions() {
		for _, user := range users {
			// add permission for configured users
			res = append(res, fmt.Sprintf("securityRoles.%q.%s.%s=true", q.Name, user, operation))
			allUsers[user] = true
		}
	}

	// admin users always get these permissions on the queue, its DLQ and its expiry address
	for _, adminOperation := range []string{"send", "consume", "browse"} {
		res = append(res,
			fmt.Sprintf("securityRoles.%q.%s.%s=true", q.Name, adminRole, adminOperation),
			fmt.Sprintf("securityRoles.%q.%s.%s=true", dlqName, adminRole, adminOperation),
			fmt.Sprintf("securityRoles.%q.%s.%s=true", expiryName, adminRole, adminOperation),
		)
	}

	// add consume permissions on DLQ for all users who have any permission on the main queue
	for user := range allUsers {
		res = append(res, fmt.Sprintf("securityRoles.%q.%s.%s=true", dlqName, user, "consume"), fmt.Sprintf("securityRoles.%q.%s.%s=true", expiryName, user, "consume"))
	}

	sort.Strings(res)

	return res
}

func processAcceptorConfigs() []string {
	return []string{
		// enable cert auto reload on acceptor
		fmt.Sprintf("acceptorConfigurations.%s.params.sslAutoReload=true", DefaultAcceptorName),
	}
}

//nolint:gocognit // Not really complex
func processAddressAndQueues(queues []pfactivemqoperatorpnetchv1alpha1.Queue) []string {
	res := []string{}

	// for each address starting with mqp. used for global custom defaults
	res = append(res, []string{
		"addressSettings.\"mqp.#\"configDeleteAddresses=FORCE",
		"addressSettings.\"mqp.#\"configDeleteQueues=FORCE",
		"addressSettings.\"mqp.#\".diskFullMessagePolicy=FAIL",
	}...)

	for i := range queues {
		queue := &queues[i]
		queueName := queue.Name
		dlqName := fmt.Sprintf(dlqTemplate, queueName)
		expiryName := fmt.Sprintf(expiryTemplate, queueName)

		// defaults for queue configs
		addressFullMessagesPolicy := "PAGE"
		maxSizeBytes := -1
		redeliveryDelay := 30000
		redeliveryMultiplier := "2.0"
		maxRedeliveryDelay := 600000
		maxDeliveryAttempts := 16
		redistributionDelay := 1000

		//nolint:nestif // Not really complex
		if queue.Config != nil {
			if queue.Config.AddressFullMessagePolicy != "" {
				addressFullMessagesPolicy = queue.Config.AddressFullMessagePolicy
			}

			if queue.Config.MaxSizeBytes != 0 {
				maxSizeBytes = queue.Config.MaxSizeBytes
			}

			if queue.Config.RedeliveryDelay != 0 {
				redeliveryDelay = queue.Config.RedeliveryDelay
			}

			if queue.Config.RedeliveryMultiplier != "" {
				redeliveryMultiplier = queue.Config.RedeliveryMultiplier
			}

			if queue.Config.MaxRedeliveryDelay != 0 {
				maxRedeliveryDelay = queue.Config.MaxRedeliveryDelay
			}

			if queue.Config.MaxDeliveryAttempts != 0 {
				maxDeliveryAttempts = queue.Config.MaxDeliveryAttempts
			}

			if queue.Config.RedistributionDelay != 0 {
				redistributionDelay = queue.Config.RedistributionDelay
			}
		}

		res = append(res, []string{
			fmt.Sprintf("addressSettings.%q.maxSizeBytes=%d", queueName, maxSizeBytes),
			fmt.Sprintf("addressSettings.%q.addressFullMessagePolicy=%s", queueName, addressFullMessagesPolicy),
			fmt.Sprintf("addressSettings.%q.deadLetterAddress=%s", queueName, dlqName),
			fmt.Sprintf("addressSettings.%q.expiryAddress=%s", queueName, expiryName),

			// redelivery / redistribution settings
			fmt.Sprintf("addressSettings.%q.redeliveryDelay=%d", queueName, redeliveryDelay),
			fmt.Sprintf("addressSettings.%q.redeliveryMultiplier=%s", queueName, redeliveryMultiplier),
			fmt.Sprintf("addressSettings.%q.maxRedeliveryDelay=%d", queueName, maxRedeliveryDelay),
			fmt.Sprintf("addressSettings.%q.maxDeliveryAttempts=%d", queueName, maxDeliveryAttempts),
			fmt.Sprintf("addressSettings.%q.redistributionDelay=%d", queueName, redistributionDelay),

			// address configs
			fmt.Sprintf("addressConfigurations.%q.routingTypes=ANYCAST", queueName),

			// address queueName configs
			fmt.Sprintf("addressConfigurations.%q.queueConfigs.%q.address=%s", queueName, queueName, queueName),
			fmt.Sprintf("addressConfigurations.%q.queueConfigs.%q.routingType=ANYCAST", queueName, queueName),

			// address dlq configs
			fmt.Sprintf("addressConfigurations.%q.routingTypes=ANYCAST", dlqName),
			fmt.Sprintf("addressConfigurations.%q.queueConfigs.%q.address=%s", dlqName, dlqName, dlqName),
			fmt.Sprintf("addressConfigurations.%q.queueConfigs.%q.routingType=ANYCAST", dlqName, dlqName),

			// address expiry configs
			fmt.Sprintf("addressConfigurations.%q.routingTypes=ANYCAST", expiryName),
			fmt.Sprintf("addressConfigurations.%q.queueConfigs.%q.address=%s", expiryName, expiryName, expiryName),
			fmt.Sprintf("addressConfigurations.%q.queueConfigs.%q.routingType=ANYCAST", expiryName, expiryName),
		}...)

		// settings without a default, only applied when explicitly configured
		if queue.Config != nil {
			if queue.Config.PageLimitBytes != 0 {
				res = append(res, fmt.Sprintf("addressSettings.%q.pageLimitBytes=%d", queueName, queue.Config.PageLimitBytes))
			}

			if queue.Config.PageFullMessagePolicy != "" {
				res = append(res, fmt.Sprintf("addressSettings.%q.pageFullMessagePolicy=%s", queueName, queue.Config.PageFullMessagePolicy))
			}

			if queue.Config.ExpiryDelay != 0 {
				res = append(res, fmt.Sprintf("addressSettings.%q.expiryDelay=%d", queueName, queue.Config.ExpiryDelay))
			}
		}
	}

	return res
}

func applyOptionalParameters(props, optionalProps []string) []string {
	for i := range optionalProps {
		propertyAlreadyDefined := false

		for j := range props {
			if strings.Split((props)[j], "=")[0] == strings.Split(optionalProps[i], "=")[0] {
				(props)[j] = optionalProps[i]
				propertyAlreadyDefined = true
			}
		}

		if !propertyAlreadyDefined {
			props = append(props, optionalProps[i])
		}
	}

	return props
}
