// Package resource contains custom resource related functions
package resource

//nolint:gosec // G101: it's not a secret
const (
	// DefaultAcceptorName acceptor name
	DefaultAcceptorName = "ssl"
	// DefaultAcceptorMappingSecretTpl secret tpl
	DefaultAcceptorMappingSecretTpl = "%s-mapping-ssl-cert"
	// DefaultAcceptorCertmgrOutSecretTpl ssl cert
	DefaultAcceptorCertmgrOutSecretTpl = "%s-ssl-cert"
	// DefaultLoggingConfigMapTpl logging config
	DefaultLoggingConfigMapTpl = "%s-logging-config"
	// DefaultAuthSecretTpl auth secret
	DefaultAuthSecretTpl = "%s-jaas-config"
	// TrustStoreSecretName name
	TrustStoreSecretName = "ces-data-p1-shared-pki-all-store"
)
