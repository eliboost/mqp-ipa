// Package resource contains custom resource related functions
package resource

import (
	"fmt"

	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

// NewLoggingConfig returns configured v1.ConfigMap logging resource
func NewLoggingConfig(pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) *v1.ConfigMap {
	logLevel := "INFO"

	if pfCR.Spec.Instance.LogLevel != "" {
		logLevel = pfCR.Spec.Instance.LogLevel
	}

	return &v1.ConfigMap{
		Name:      fmt.Sprintf(DefaultLoggingConfigMapTpl, pfCR.Name),
		Namespace: pfCR.Spec.Instance.Namespace,
		Labels: map[string]string{
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQManagedByOperatorLabelKey:        pfactivemqoperatorpnetchv1alpha1.LabelValueTrue,
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNamespaceLabelKey: pfCR.Namespace,
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNameLabelKey:      pfCR.Name,
		},
		Data: map[string]string{"logging.properties": fmt.Sprintf(`
			# Log4J 2 configuration
			
			# Monitor config file every X seconds for updates
			monitorInterval=5
			
			rootLogger.level=INFO
			rootLogger.appenderRef.console.ref=console
			
			logger.activemq.name=org.apache.activemq
			logger.activemq.level=%s
			
			logger.artemis_journal.name=org.apache.activemq.artemis.journal
			logger.artemis_journal.level=INFO
			
			logger.artemis_utils.name=org.apache.activemq.artemis.utils
			logger.artemis_utils.level=INFO
			
			# CriticalAnalyzer: If you have issues with the CriticalAnalyzer, setting this to TRACE would give
			# you extra troubleshooting info, but do not use TRACE regularly as it would incur extra CPU usage.
			logger.critical_analyzer.name=org.apache.activemq.artemis.utils.critical
			logger.critical_analyzer.level=INFO
			
			# Audit loggers: to enable change levels from OFF to INFO
			logger.audit_base.name=org.apache.activemq.audit.base
			logger.audit_base.level=OFF
			
			logger.audit_resource.name=org.apache.activemq.audit.resource
			logger.audit_resource.level=OFF
			
			logger.audit_message.name=org.apache.activemq.audit.message
			logger.audit_message.level=OFF
			
			# Jetty logger levels
			logger.jetty.name=org.eclipse.jetty
			logger.jetty.level=WARN
			
			# web console authenticator too verbose for impatient client
			logger.authentication_filter.name=io.hawt.web.auth.AuthenticationFilter
			logger.authentication_filter.level=ERROR
			
			# Quorum related logger levels
			logger.curator.name=org.apache.curator
			logger.curator.level=WARN

			logger.zookeeper.name=org.apache.zookeeper
			logger.zookeeper.level=ERROR
			
			# Noisy loggers on DEBUG and TRACE
			logger.certificate_util.name=org.apache.activemq.artemis.core.remoting.CertificateUtil
			logger.certificate_util.level=INFO

			logger.security_store_impl.name=org.apache.activemq.artemis.core.security.impl.SecurityStoreImpl
			logger.security_store_impl.level=INFO

			logger.jchannel_wrapper.name=org.apache.activemq.artemis.api.core.jgroups.JChannelWrapper
			logger.jchannel_wrapper.level=INFO

			logger.jgroups_broadcast_endpoint.name=org.apache.activemq.artemis.api.core.JGroupsBroadcastEndpoint
			logger.jgroups_broadcast_endpoint.level=INFO

			logger.page_subscription_counter_impl.name=org.apache.activemq.artemis.core.paging.cursor.impl.PageSubscriptionCounterImpl
			logger.page_subscription_counter_impl.level=INFO

			# Console appender
			appender.console.type=Console
			appender.console.name=console
			appender.console.layout.type=PatternLayout
			appender.console.layout.pattern=%%d %%-5level [%%logger] %%msg%%n
	`, logLevel)},
	}
}
