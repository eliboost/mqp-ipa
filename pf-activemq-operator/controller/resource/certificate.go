// Package resource contains custom resource related functions
package resource

import (
	"fmt"
	"strings"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCert returns configured certmanagerv1.Certificate resource
func NewCert(pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ, dnsNames []string, ingressDomain string) *certmanagerv1.Certificate {
	env, _, _ := strings.Cut(ingressDomain, "-")

	issuerName := "tls-rsa-nonprod-vault"

	if env == "p1" {
		issuerName = "tls-rsa-prod-vault"
	}

	return &certmanagerv1.Certificate{
		Name:      fmt.Sprintf(DefaultAcceptorCertmgrOutSecretTpl, pfCR.Name),
		Namespace: pfCR.Spec.Instance.Namespace,
		Labels: map[string]string{
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQManagedByOperatorLabelKey:        pfactivemqoperatorpnetchv1alpha1.LabelValueTrue,
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNamespaceLabelKey: pfCR.Namespace,
			pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNameLabelKey:      pfCR.Name,
		},
		Spec: certmanagerv1.CertificateSpec{
			CommonName:            fmt.Sprintf("%s.%s", pfCR.Name, ingressDomain),
			DNSNames:              dnsNames,
			Duration:              &metav1.Duration{Duration: 6 * 30 * 24 * time.Hour},
			EncodeUsagesInRequest: new(true),
			IssuerRef: certmanagermetav1.IssuerReference{
				Name:  issuerName,
				Kind:  "ClusterIssuer",
				Group: "cert-manager.io",
			},
			PrivateKey: &certmanagerv1.CertificatePrivateKey{
				RotationPolicy: certmanagerv1.RotationPolicyAlways,
				Encoding:       certmanagerv1.PKCS1,
				Algorithm:      certmanagerv1.RSAKeyAlgorithm,
				Size:           4096,
			},
			SecretName: fmt.Sprintf(DefaultAcceptorCertmgrOutSecretTpl, pfCR.Name),
			Subject: &certmanagerv1.X509Subject{
				Organizations:       []string{"PostFinance AG"},
				Countries:           []string{"CH"},
				OrganizationalUnits: []string{"PostFinance AG"},
				Localities:          []string{"Bern"},
				Provinces:           []string{"Bern"},
			},
			Keystores: &certmanagerv1.CertificateKeystores{
				JKS: &certmanagerv1.JKSKeystore{
					Create: true,
					PasswordSecretRef: certmanagermetav1.SecretKeySelector{
						LocalObjectReference: certmanagermetav1.LocalObjectReference{
							Name: fmt.Sprintf(DefaultAcceptorMappingSecretTpl, pfCR.Name),
						},
						Key: "keyStorePassword",
					},
				},
			},
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageServerAuth,
				certmanagerv1.UsageClientAuth,
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
			},
		},
	}
}
