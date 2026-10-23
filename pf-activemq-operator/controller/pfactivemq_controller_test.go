package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	activemqartemisv1beta1 "github.com/arkmq-org/activemq-artemis-operator/v2/api/v1beta1"
	"github.com/stretchr/testify/require"
	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/controller/condition"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	testNamespace       = "appl-activemq-iscsi-test-e1"
	testIngressDomain   = "t1-nucleus-a.pnet.ch"
	testT2IngressDomain = "t2-nucleus-a.pnet.ch"
)

// TestControllerIntegration is e e2e test for the pfactivemq controller.
func TestControllerIntegration(t *testing.T) {
	t.Parallel()

	log.SetLogger(zap.New(zap.UseDevMode(true)))

	singleBroker := &pfactivemqoperatorpnetchv1alpha1.PFActiveMQ{
		Namespace: "default", Name: "single-broker",
		Spec: pfactivemqoperatorpnetchv1alpha1.PFActiveMQSpec{
			TeamMail: "test@postfinance.ch",
			Roles: pfactivemqoperatorpnetchv1alpha1.Roles{
				Owner:     "bancs/appl_xyz",
				Operators: []string{"bancs.mqp"},
			},
			Instance: pfactivemqoperatorpnetchv1alpha1.Instance{
				Namespace:   testNamespace,
				Replicas:    1,
				SplunkIndex: "midwitd",
				Storage:     "4Gi",
				Queues: []pfactivemqoperatorpnetchv1alpha1.Queue{
					{
						Name: "mqp.test.In_Queue",
						Permission: pfactivemqoperatorpnetchv1alpha1.Permission{
							Send:    []string{"userA"},
							Consume: []string{"userB"},
						},
						Config: &pfactivemqoperatorpnetchv1alpha1.Config{
							MaxSizeBytes:          10485760,
							PageLimitBytes:        10485760,
							PageFullMessagePolicy: "FAIL",
							ExpiryDelay:           2,
						},
					},
				},
			},
		},
	}

	multiBroker := &pfactivemqoperatorpnetchv1alpha1.PFActiveMQ{
		Namespace: "default", Name: "multi-broker",
		Spec: pfactivemqoperatorpnetchv1alpha1.PFActiveMQSpec{
			TeamMail: "test@postfinance.ch",
			Roles: pfactivemqoperatorpnetchv1alpha1.Roles{
				Owner:     "bancs/appl_xyz",
				Operators: []string{"bancs.mqp"},
			},
			Instance: pfactivemqoperatorpnetchv1alpha1.Instance{
				Namespace:   testNamespace,
				Replicas:    2,
				Storage:     "2Gi",
				SplunkIndex: "midwitd",
				Queues: []pfactivemqoperatorpnetchv1alpha1.Queue{
					{
						Name: "mqp.test.In_Queue",
						Permission: pfactivemqoperatorpnetchv1alpha1.Permission{
							Send:    []string{"userA"},
							Consume: []string{"userB"},
						},
					},
				},
			},
		},
	}

	result, teardown := setup(t)

	t.Cleanup(func() {
		teardown()
	})

	tests := []struct {
		name         string
		pfActiveMQ   *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ
		testTemplate string
		assertions   func(t *testing.T, brokerProperties []string)
	}{
		{
			name:         "single-broker",
			pfActiveMQ:   singleBroker,
			testTemplate: "single.yaml",
			assertions: func(t *testing.T, brokerProperties []string) {
				t.Helper()
				require.Contains(t, brokerProperties, `addressSettings."mqp.test.In_Queue".maxSizeBytes=10485760`)
				require.Contains(t, brokerProperties, `addressSettings."mqp.test.In_Queue".pageLimitBytes=10485760`)
				require.Contains(t, brokerProperties, `addressSettings."mqp.test.In_Queue".pageFullMessagePolicy=FAIL`)
				require.Contains(t, brokerProperties, `addressSettings."mqp.test.In_Queue".expiryDelay=2`)
			},
		},
		{
			name:         "multi-broker",
			pfActiveMQ:   multiBroker,
			testTemplate: "multi.yaml",
			assertions: func(t *testing.T, brokerProperties []string) {
				t.Helper()
				require.Contains(t, brokerProperties, `addressSettings."mqp.test.In_Queue".maxSizeBytes=-1`)
				require.NotContains(t, brokerProperties, `addressSettings."mqp.test.In_Queue".pageLimitBytes`)
				require.NotContains(t, brokerProperties, `addressSettings."mqp.test.In_Queue".pageFullMessagePolicy`)
				require.NotContains(t, brokerProperties, `addressSettings."mqp.test.In_Queue".expiryDelay`)
			},
		},
		{
			name:         "multi-broker expiry queue",
			pfActiveMQ:   multiBroker,
			testTemplate: "multi.yaml",
			assertions: func(t *testing.T, brokerProperties []string) {
				t.Helper()
				require.Contains(t, brokerProperties, `addressSettings."mqp.test.In_Queue".expiryAddress=mqp.test.In_Queue_EXPIRY`)
				require.Contains(t, brokerProperties, `addressConfigurations."mqp.test.In_Queue_EXPIRY".routingTypes=ANYCAST`)
				require.Contains(t, brokerProperties, `addressConfigurations."mqp.test.In_Queue_EXPIRY".queueConfigs."mqp.test.In_Queue_EXPIRY".address=mqp.test.In_Queue_EXPIRY`)
				require.Contains(t, brokerProperties, `addressConfigurations."mqp.test.In_Queue_EXPIRY".queueConfigs."mqp.test.In_Queue_EXPIRY".routingType=ANYCAST`)
				require.Contains(t, brokerProperties, `securityRoles."mqp.test.In_Queue_EXPIRY".admin.send=true`)
				require.Contains(t, brokerProperties, `securityRoles."mqp.test.In_Queue_EXPIRY".admin.consume=true`)
				require.Contains(t, brokerProperties, `securityRoles."mqp.test.In_Queue_EXPIRY".userA.consume=true`)
			},
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			name := fmt.Sprintf("%d-pfactivemq-sample-schmidomin", i+1)

			err := installRequiredTestRunFiles(result.admin, name, tc.testTemplate)
			require.NoError(t, err)

			p := tc.pfActiveMQ.DeepCopy()
			p.Name = name

			err = result.k8sClient.Create(context.TODO(), p)
			require.NoError(t, err)

			t.Cleanup(func() {
				err := result.k8sClient.Delete(context.TODO(), p.DeepCopy())
				if err != nil && !apierrors.IsNotFound(err) {
					require.NoError(t, err)
				}

				require.Eventually(t, func() bool {
					err := result.k8sClient.Get(context.TODO(), client.ObjectKey{Namespace: p.Namespace, Name: name}, &pfactivemqoperatorpnetchv1alpha1.PFActiveMQ{})
					return apierrors.IsNotFound(err)
				}, 60*time.Second, time.Second)
			})

			fakeSetArtemisStatusToReady(t, result.k8sClient, name, p.Spec.Instance.Namespace)
			fakeSetPVCsToReady(t, result.k8sClient, name, p.Spec.Instance.Namespace)

			require.Eventually(t, condition.HasConditionFunc(context.TODO(), result.k8sClient, p.DeepCopy(), metav1.Condition{
				Type:   pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeReady,
				Status: metav1.ConditionTrue,
			}), 300*time.Second, time.Second)

			artemis := &activemqartemisv1beta1.ActiveMQArtemis{}
			err = result.k8sClient.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: p.Spec.Instance.Namespace}, artemis)
			require.NoError(t, err)

			tc.assertions(t, artemis.Spec.BrokerProperties)

			// PVC Increase

			pvcList := &v1.PersistentVolumeClaimList{}
			err = result.k8sClient.List(context.TODO(), pvcList, client.InNamespace(p.Spec.Instance.Namespace), client.MatchingLabels{
				pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNamespaceLabelKey: p.Spec.Instance.Namespace,
				pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNameLabelKey:      name,
			})
			require.NoError(t, err)

			prefix := fmt.Sprintf("%s-%s-ss-", name, name)
			for _, pvc := range pvcList.Items {
				if strings.HasPrefix(pvc.Name, prefix) {
					require.Equal(t, p.Spec.Instance.Storage, pvc.Spec.Resources.Requests.Storage().String())
				}
			}

			freshP := p.DeepCopy()
			require.Eventually(t, func() bool {
				if err := result.k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(freshP), freshP); err != nil {
					return false
				}
				return freshP.Status.LastAppliedStorage == freshP.Spec.Instance.Storage
			}, 60*time.Second, time.Second, "Status.LastAppliedStorage should be updated in status after PVC increase")
		})
	}

}

// TestCalcDNSNames covers the dns names put into the certificate, including the
// t2 alias that exists for the t1 cluster.
func TestCalcDNSNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		instanceName  string
		ingressDomain string
		spec          activemqartemisv1beta1.ActiveMQArtemisSpec
		expected      []string
	}{
		{
			name:          "no-brokers",
			instanceName:  "t1-broker",
			ingressDomain: testIngressDomain,
			spec:          newArtemisSpec(0, false),
			expected: []string{
				"t1-broker." + testIngressDomain,
			},
		},
		{
			name:          "two-brokers",
			instanceName:  "t1-broker",
			ingressDomain: testIngressDomain,
			spec:          newArtemisSpec(2, true, sslAcceptor("testAcceptor")),
			expected: []string{
				"t1-broker-testAcceptor-0." + testNamespace + "." + testIngressDomain,
				"t1-broker-testAcceptor-0-svc." + testNamespace + ".svc.cluster.local",
				"t1-broker-ss-0.t1-broker-hdls-svc." + testNamespace + ".svc.cluster.local",
				"t1-broker-console-0." + testNamespace + "." + testIngressDomain,
				"t1-broker-testAcceptor-1." + testNamespace + "." + testIngressDomain,
				"t1-broker-testAcceptor-1-svc." + testNamespace + ".svc.cluster.local",
				"t1-broker-ss-1.t1-broker-hdls-svc." + testNamespace + ".svc.cluster.local",
				"t1-broker-console-1." + testNamespace + "." + testIngressDomain,
			},
		},
		{
			name:          "t2-instance",
			instanceName:  "t2-broker",
			ingressDomain: testIngressDomain,
			spec:          newArtemisSpec(1, true, sslAcceptor("testAcceptor")),
			expected: []string{
				"t2-broker-testAcceptor-0." + testNamespace + "." + testIngressDomain,
				"t2-broker-testAcceptor-0-svc." + testNamespace + ".svc.cluster.local",
				"t2-broker-ss-0.t2-broker-hdls-svc." + testNamespace + ".svc.cluster.local",
				"t2-broker-testAcceptor-0." + testNamespace + "." + testT2IngressDomain,
				"t2-broker-console-0." + testNamespace + "." + testIngressDomain,
				"t2-broker-console-0." + testNamespace + "." + testT2IngressDomain,
			},
		},
		{
			name:          "t2-instance-no-brokers",
			instanceName:  "t2-broker",
			ingressDomain: testIngressDomain,
			spec:          newArtemisSpec(0, false),
			expected: []string{
				"t2-broker." + testIngressDomain,
				"t2-broker." + testT2IngressDomain,
			},
		},
		{
			name:          "t2-instance-on-t2-cluster",
			instanceName:  "t2-broker",
			ingressDomain: testT2IngressDomain,
			spec:          newArtemisSpec(1, false, sslAcceptor("testAcceptor")),
			expected: []string{
				"t2-broker-testAcceptor-0." + testNamespace + "." + testT2IngressDomain,
				"t2-broker-testAcceptor-0-svc." + testNamespace + ".svc.cluster.local",
				"t2-broker-ss-0.t2-broker-hdls-svc." + testNamespace + ".svc.cluster.local",
			},
		},
		{
			name:          "t1-instance-on-t1-cluster",
			instanceName:  "t1-broker",
			ingressDomain: testIngressDomain,
			spec:          newArtemisSpec(1, false, sslAcceptor("testAcceptor")),
			expected: []string{
				"t1-broker-testAcceptor-0." + testNamespace + "." + testIngressDomain,
				"t1-broker-testAcceptor-0-svc." + testNamespace + ".svc.cluster.local",
				"t1-broker-ss-0.t1-broker-hdls-svc." + testNamespace + ".svc.cluster.local",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := calcDNSNames(tc.spec, types.NamespacedName{
				Name:      tc.instanceName,
				Namespace: testNamespace,
			}, tc.ingressDomain)

			require.ElementsMatch(t, tc.expected, got)
		})
	}
}

func newArtemisSpec(size int32, exposeConsole bool, acceptors ...activemqartemisv1beta1.AcceptorType) activemqartemisv1beta1.ActiveMQArtemisSpec {
	return newArtemisSpecWithSize(&size, exposeConsole, acceptors...)
}

func newArtemisSpecWithSize(size *int32, exposeConsole bool, acceptors ...activemqartemisv1beta1.AcceptorType) activemqartemisv1beta1.ActiveMQArtemisSpec {
	return activemqartemisv1beta1.ActiveMQArtemisSpec{
		DeploymentPlan: activemqartemisv1beta1.DeploymentPlanType{
			Size: size,
		},
		Acceptors: acceptors,
		Console: activemqartemisv1beta1.ConsoleType{
			Expose:     exposeConsole,
			SSLEnabled: exposeConsole,
		},
	}
}

func sslAcceptor(name string) activemqartemisv1beta1.AcceptorType {
	return activemqartemisv1beta1.AcceptorType{
		Name:       name,
		SSLEnabled: true,
	}
}

func fakeSetArtemisStatusToReady(t *testing.T, k8sClient client.Client, name, namespace string) {
	artemis := &activemqartemisv1beta1.ActiveMQArtemis{}

	err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: namespace}, artemis)
	require.NoError(t, err)

	condition.SetStatusCondition(context.TODO(), &artemis.Status.Conditions, metav1.Condition{
		Type:   activemqartemisv1beta1.ReadyConditionType,
		Reason: activemqartemisv1beta1.ReadyConditionReason,
		Status: metav1.ConditionTrue,
	})

	err = k8sClient.Status().Update(context.TODO(), artemis)
	require.NoError(t, err)
}

func fakeSetPVCsToReady(t *testing.T, k8sClient client.Client, name, namespace string) {
	pvcList := &v1.PersistentVolumeClaimList{}

	err := k8sClient.List(context.TODO(), pvcList, client.InNamespace(namespace), client.MatchingLabels{
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNamespaceLabelKey: namespace,
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNameLabelKey:      name,
	})
	require.NoError(t, err)

	err = k8sClient.List(context.TODO(), pvcList, client.InNamespace(namespace))
	if err != nil {
		return
	}

	prefix := fmt.Sprintf("%s-%s-ss-", name, name)
	for _, pvc := range pvcList.Items {

		if strings.HasPrefix(pvc.Name, prefix) {
			pvc.Status.Phase = v1.ClaimBound

			err = k8sClient.Status().Update(context.TODO(), &pvc)
			require.NoError(t, err)
		}
	}

}
