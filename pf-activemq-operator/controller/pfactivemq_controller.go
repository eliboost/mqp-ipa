// Package controller contains kubernetes controller functionality
package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/controller/condition"
	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/controller/resource"
	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/internal/logger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	activemqartemisv1beta1 "github.com/arkmq-org/activemq-artemis-operator/v2/api/v1beta1"
	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// Config for the reconciler
type Config struct {
	Artemis             resource.ArtemisConfig
	OperationCommonName string
}

// PFActiveMQReconciler reconciles a PFActiveMQ object
type PFActiveMQReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config Config
}

const (
	finalizerName = "pf-activemq-operator.pnet.ch/finalizer"
)

// +kubebuilder:rbac:groups=pf-activemq-operator.pnet.ch,resources=pfactivemqs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pf-activemq-operator.pnet.ch,resources=pfactivemqs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pf-activemq-operator.pnet.ch,resources=pfactivemqs/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=broker.amq.io,resources=activemqartemises,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;patch;update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;patch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PFActiveMQReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	pfCustomResource := &pfactivemqoperatorpnetchv1alpha1.PFActiveMQ{}

	err := r.Get(ctx, req.NamespacedName, pfCustomResource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			l.Info("No customResource PFActiveMQ found")
			return ctrl.Result{}, nil
		}
	}

	// Check if the resource is being deleted
	if !pfCustomResource.DeletionTimestamp.IsZero() {
		return r.DeletionReconciler(ctx, pfCustomResource)
	}

	if !controllerutil.ContainsFinalizer(pfCustomResource, finalizerName) {
		controllerutil.AddFinalizer(pfCustomResource, finalizerName)

		if err := r.Update(ctx, pfCustomResource); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.updateStatusCondition(ctx, pfCustomResource); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.syncObjects(ctx, pfCustomResource); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create or update resources for %s: %w", pfCustomResource.Name, err)
	}

	_ = r.Status().Update(ctx, pfCustomResource)

	if !condition.HasConditionAndReason(pfCustomResource.Status.Conditions, metav1.Condition{
		Type:   pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeReady,
		Status: metav1.ConditionTrue,
		Reason: pfactivemqoperatorpnetchv1alpha1.ReasonReady,
	}) {
		return ctrl.Result{
			RequeueAfter: time.Second * 5,
		}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PFActiveMQReconciler) SetupWithManager(mgr ctrl.Manager) error {
	labelPredicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{MatchLabels: map[string]string{pfactivemqoperatorpnetchv1alpha1.PFActiveMQManagedByOperatorLabelKey: "true"}})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&pfactivemqoperatorpnetchv1alpha1.PFActiveMQ{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.findPFActiveMQForObject), builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, labelPredicate)).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(r.findPFActiveMQForObject), builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, labelPredicate)).
		Watches(&certmanagerv1.Certificate{}, handler.EnqueueRequestsFromMapFunc(r.findPFActiveMQForObject), builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, labelPredicate)).
		Watches(&activemqartemisv1beta1.ActiveMQArtemis{}, handler.EnqueueRequestsFromMapFunc(r.findPFActiveMQForObject), builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, labelPredicate)).
		Complete(r)
}

// DeletionReconciler delete
func (r *PFActiveMQReconciler) DeletionReconciler(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("enter deletion reconciler")

	condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
		Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeReady,
		Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonTerminating,
		Message: "PFActiveMQ is terminating.",
		Status:  metav1.ConditionFalse,
	})

	objects, err := resource.ExtraObjects(ctx, r.Client, pfCR)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list objects: %w", err)
	}

	for i := range objects {
		o := objects[i]

		if err := r.Delete(ctx, o); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete object: %w", err)
		}
	}

	controllerutil.RemoveFinalizer(pfCR, finalizerName)

	if err := r.Update(ctx, pfCR); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// syncObjects creates and updates additional resources as cert-manager cert and artemis cloud operator resources
func (r *PFActiveMQReconciler) syncObjects(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) error {
	l := log.FromContext(ctx)

	targetNamespace := types.NamespacedName{
		Name:      pfCR.Name,
		Namespace: pfCR.Spec.Instance.Namespace,
	}

	artemisCustomResource := resource.NewArtemis(ctx, pfCR, r.Config.Artemis)

	dnsNames := calcDNSNames(artemisCustomResource.Spec, targetNamespace, r.Config.Artemis.IngressDomain)
	certResource := resource.NewCert(pfCR, dnsNames, r.Config.Artemis.IngressDomain)
	loggingConfig := resource.NewLoggingConfig(pfCR)
	mappingSecret := resource.NewMappingSecret(pfCR)
	authSecret := resource.NewAuthSecret(pfCR, r.Config.OperationCommonName)

	objects := map[string]client.Object{
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeCertSecretResourcesReady: mappingSecret.DeepCopy(),
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeCertResourcesReady:       certResource.DeepCopy(),
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeLoggingConfigReady:       loggingConfig.DeepCopy(),
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeAuthSecretReady:          authSecret.DeepCopy(),
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeActiveMQBrokersReady:     artemisCustomResource.DeepCopy(),
	}

	secrets := map[string]map[string]string{
		mappingSecret.Name: mappingSecret.StringData,
		authSecret.Name:    authSecret.StringData,
	}

	if !reflect.DeepEqual(pfCR.Status.LastAppliedStorage, pfCR.Spec.Instance.Storage) {
		if err := r.syncPVCs(ctx, pfCR); err != nil {
			return err
		}
	}

	for conditionType, object := range objects {
		if condition.HasCondition(pfCR.Status.Conditions, metav1.Condition{
			Type:   conditionType,
			Status: metav1.ConditionTrue,
		}) {
			l.V(4).Info("no sync necessary", logger.Type(conditionType), logger.Reason(pfactivemqoperatorpnetchv1alpha1.ReasonReady))

			continue
		}

		result, err := createOrPatch(ctx, r.Client, object, func() error {
			if isManaged, ok := object.GetLabels()[pfactivemqoperatorpnetchv1alpha1.PFActiveMQManagedByOperatorLabelKey]; !ok || isManaged != "true" {
				l.Info("skipped unmanaged resource",
					"object", fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
				)

				return nil
			}

			if typed, ok := object.(*activemqartemisv1beta1.ActiveMQArtemis); ok {
				typed.Spec = artemisCustomResource.Spec

				return nil
			}

			if typed, ok := object.(*certmanagerv1.Certificate); ok {
				typed.Spec = certResource.Spec

				return nil
			}

			if typed, ok := object.(*corev1.ConfigMap); ok {
				typed.Data = loggingConfig.Data

				return nil
			}

			if typed, ok := object.(*corev1.Secret); ok {
				data := secrets[typed.Name]
				typed.StringData = data

				return nil
			}

			return nil
		})
		if err != nil {
			return err
		}

		if result != controllerutil.OperationResultNone {
			l.Info("create or update object",
				"object", fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
				"type", conditionType,
				"operation", result,
			)
		}
	}

	return nil
}

func calcDNSNames(artemisSpec activemqartemisv1beta1.ActiveMQArtemisSpec, name types.NamespacedName, ingressDomain string) []string {
	var (
		dnsNames []string
		t2Domain string
	)

	// a t2 instance on the t1 cluster is also reachable via the t2 alias
	if strings.HasPrefix(name.Name, "t2") && !strings.HasPrefix(ingressDomain, "t2") {
		t2Domain = strings.Replace(ingressDomain, "t1", "t2", 1)
	}

	if artemisSpec.DeploymentPlan.Size == nil || *artemisSpec.DeploymentPlan.Size == 0 {
		// add placeholder name until we add brokers
		dnsNames = append(dnsNames, fmt.Sprintf("%s.%s", name.Name, ingressDomain))

		if t2Domain != "" {
			dnsNames = append(dnsNames, fmt.Sprintf("%s.%s", name.Name, t2Domain))
		}

		return dnsNames
	}

	for i := range artemisSpec.Acceptors {
		acceptor := artemisSpec.Acceptors[i]

		if !acceptor.SSLEnabled {
			continue
		}

		for ordinal := range *artemisSpec.DeploymentPlan.Size {
			ingressName := fmt.Sprintf("%s-%s-%d.%s.%s", name.Name, acceptor.Name, ordinal, name.Namespace, ingressDomain)                 // must match IngressHost
			serviceName := fmt.Sprintf("%s-%s-%d-svc.%s.svc.cluster.local", name.Name, acceptor.Name, ordinal, name.Namespace)             // <cr-name>-<acceptor>-<ordinal>-svc.<namespace>.svc.cluster.local
			headlessServiceName := fmt.Sprintf("%s-ss-%d.%s-hdls-svc.%s.svc.cluster.local", name.Name, ordinal, name.Name, name.Namespace) // <cr-name>-ss-<ordinal>.<cr-name>-hdls-svc.<namespace>.svc.cluster.local

			dnsNames = append(dnsNames, ingressName, serviceName, headlessServiceName)

			if t2Domain != "" {
				dnsNames = append(dnsNames, fmt.Sprintf("%s-%s-%d.%s.%s", name.Name, acceptor.Name, ordinal, name.Namespace, t2Domain))
			}

			if artemisSpec.Console.Expose && artemisSpec.Console.SSLEnabled {
				dnsNames = append(dnsNames, fmt.Sprintf("%s-console-%d.%s.%s", name.Name, ordinal, name.Namespace, ingressDomain))

				if t2Domain != "" {
					dnsNames = append(dnsNames, fmt.Sprintf("%s-console-%d.%s.%s", name.Name, ordinal, name.Namespace, t2Domain))
				}
			}
		}
	}

	return dnsNames
}

func (r *PFActiveMQReconciler) updateStatusCondition(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) error {
	if found := condition.FindStatusCondition(pfCR.Status.Conditions, pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeReady); found == nil {
		condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
			Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeReady,
			Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonUnknown,
			Message: "The reconciler checks all the resources...",
			Status:  metav1.ConditionUnknown,
		})
	}

	targetNamespace := types.NamespacedName{
		Name:      pfCR.Name,
		Namespace: pfCR.Spec.Instance.Namespace,
	}

	desiredMappingSecretName, err := r.setCertResourcesReady(ctx, pfCR, targetNamespace)
	if err != nil {
		return err
	}

	if err := r.setCertSecretResourcesReady(ctx, pfCR, *desiredMappingSecretName); err != nil {
		return err
	}

	if err := r.setLoggingConfigResourcesReady(ctx, pfCR); err != nil {
		return err
	}

	if err := r.setAuthSecretResourcesReady(ctx, pfCR); err != nil {
		return err
	}

	if err := r.setActiveMQBrokersReady(ctx, pfCR); err != nil {
		return err
	}

	if !allConditionsReady(pfCR) {
		condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
			Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeReady,
			Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonReady,
			Message: "Not all resources are ready",
			Status:  metav1.ConditionFalse,
		})

		return nil
	}

	condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
		Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeReady,
		Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonReady,
		Message: "All resources are created and ready",
		Status:  metav1.ConditionTrue,
	})

	return nil
}

func allConditionsReady(pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) bool {
	stats := []string{
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeCertResourcesReady,
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeLoggingConfigReady,
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeCertSecretResourcesReady,
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeActiveMQBrokersReady,
	}

	for _, stat := range stats {
		if !condition.HasCondition(pfCR.Status.Conditions,
			metav1.Condition{
				Type:   stat,
				Status: metav1.ConditionTrue,
			},
		) {
			return false
		}
	}

	return true
}

func (r *PFActiveMQReconciler) setActiveMQBrokersReady(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) error {
	activeMQ := &activemqartemisv1beta1.ActiveMQArtemis{}
	desiredActiveMQ := resource.NewArtemis(ctx, pfCR, r.Config.Artemis)

	if err := r.Get(ctx, client.ObjectKey{Name: pfCR.Name, Namespace: pfCR.Spec.Instance.Namespace}, activeMQ); err != nil {
		if apierrors.IsNotFound(err) {
			condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
				Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeActiveMQBrokersReady,
				Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonMissing,
				Message: fmt.Sprintf("Resource %q of kind %q does not exist.", pfCR.Name, activeMQ.Kind),
				Status:  metav1.ConditionFalse,
			})

			return nil
		}

		return err
	}

	// mismatch number of brokers
	if *activeMQ.Spec.DeploymentPlan.Size != pfCR.Spec.Instance.Replicas {
		condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
			Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeActiveMQBrokersReady,
			Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonMismatch,
			Message: fmt.Sprintf("Resource %q has mismatched number of brokers.", pfCR.Name),
			Status:  metav1.ConditionFalse,
		})

		return nil
	}

	// show zero size deployments as healthy
	if condition.HasConditionAndReason(activeMQ.Status.Conditions, metav1.Condition{
		Type:   activemqartemisv1beta1.DeployedConditionType,
		Reason: activemqartemisv1beta1.DeployedConditionZeroSizeReason,
		Status: metav1.ConditionFalse,
	}) {
		condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
			Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeActiveMQBrokersReady,
			Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonReady,
			Message: fmt.Sprintf("Resource %q has size 0. No pods deployt.", pfCR.Name),
			Status:  metav1.ConditionTrue,
		})

		return nil
	}

	if !condition.HasConditionAndReason(activeMQ.Status.Conditions, metav1.Condition{
		Type:   activemqartemisv1beta1.ReadyConditionType,
		Reason: activemqartemisv1beta1.ReadyConditionReason,
		Status: metav1.ConditionTrue,
	}) {
		condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
			Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeActiveMQBrokersReady,
			Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonReady,
			Message: fmt.Sprintf("Resource %q of kind %q is not ready.", pfCR.Name, activeMQ.Kind),
			Status:  metav1.ConditionFalse,
		})

		return nil
	}

	if !reflect.DeepEqual(desiredActiveMQ.Spec, activeMQ.Spec) {
		condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
			Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeActiveMQBrokersReady,
			Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonMismatch,
			Message: fmt.Sprintf("Desired spec not matching for resource %q of kind %q", pfCR.Name, activeMQ.Kind),
			Status:  metav1.ConditionFalse,
		})

		return nil
	}

	condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
		Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeActiveMQBrokersReady,
		Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonReady,
		Message: fmt.Sprintf("Resource %q of kind %q is ready.", pfCR.Name, activeMQ.Kind),
		Status:  metav1.ConditionTrue,
	})

	return nil
}

func (r *PFActiveMQReconciler) setCertSecretResourcesReady(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ, desiredMappingSecretName string) error {
	certMappingSecret := &corev1.Secret{Name: desiredMappingSecretName}

	alwaysEqual := func() bool {
		return true
	}

	return r.setResourceStatus(ctx, pfCR, certMappingSecret, pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeCertSecretResourcesReady, alwaysEqual)
}

func (r *PFActiveMQReconciler) setCertResourcesReady(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ, targetNamespace types.NamespacedName) (name *string, err error) {
	artemisCustomResource := resource.NewArtemis(ctx, pfCR, r.Config.Artemis)

	dnsNames := calcDNSNames(artemisCustomResource.Spec, targetNamespace, r.Config.Artemis.IngressDomain)
	desiredCert := resource.NewCert(pfCR, dnsNames, r.Config.Artemis.IngressDomain)
	desiredMappingSecret := resource.NewMappingSecret(pfCR)

	cert := &certmanagerv1.Certificate{}

	if err := r.Get(ctx, client.ObjectKey{Name: desiredCert.Name, Namespace: pfCR.Spec.Instance.Namespace}, cert); err != nil {
		if apierrors.IsNotFound(err) {
			condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
				Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeCertResourcesReady,
				Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonMissing,
				Message: fmt.Sprintf("Resource %q of kind %q does not exist.", desiredCert.Name, cert.Kind),
				Status:  metav1.ConditionFalse,
			})

			return &desiredMappingSecret.Name, nil
		}

		return nil, err
	}

	if !reflect.DeepEqual(cert.Spec.DNSNames, dnsNames) {
		condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
			Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeCertResourcesReady,
			Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonMismatch,
			Message: fmt.Sprintf("DnsNames not matching for resource %q of kind %q", pfCR.Name, cert.Kind),
			Status:  metav1.ConditionFalse,
		})

		return &desiredMappingSecret.Name, nil
	}

	condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
		Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeCertResourcesReady,
		Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonReady,
		Message: fmt.Sprintf("Resource %q of kind %q exists.", desiredCert.Name, cert.Kind),
		Status:  metav1.ConditionTrue,
	})

	return &desiredMappingSecret.Name, nil
}

func (r *PFActiveMQReconciler) setLoggingConfigResourcesReady(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) error {
	desiredLoggingConfig := resource.NewLoggingConfig(pfCR)
	configMap := &corev1.ConfigMap{Name: desiredLoggingConfig.Name}

	isEqual := func() bool {
		return reflect.DeepEqual(desiredLoggingConfig.Data, configMap.Data)
	}

	return r.setResourceStatus(ctx, pfCR, configMap, pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeLoggingConfigReady, isEqual)
}

func (r *PFActiveMQReconciler) setResourceStatus(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ, object client.Object, conditionType string, isEqualFunc func() bool) error {
	if err := r.Get(ctx, client.ObjectKey{Name: object.GetName(), Namespace: pfCR.Spec.Instance.Namespace}, object); err != nil {
		if apierrors.IsNotFound(err) {
			condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
				Type:    conditionType,
				Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonMissing,
				Message: fmt.Sprintf("Resource %q of kind %q does not exist.", object.GetName(), object.GetObjectKind().GroupVersionKind().Kind),
				Status:  metav1.ConditionFalse,
			})

			return nil
		}

		return err
	}

	isEqual := isEqualFunc()

	if !isEqual {
		condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
			Type:    conditionType,
			Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonMismatch,
			Message: fmt.Sprintf("Data for %s not matching for resource %q of kind %q", object.GetName(), pfCR.Name, object.GetObjectKind().GroupVersionKind().Kind),
			Status:  metav1.ConditionFalse,
		})

		return nil
	}

	condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
		Type:    conditionType,
		Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonReady,
		Message: fmt.Sprintf("Resource %q of kind %q exists.", object.GetName(), object.GetObjectKind().GroupVersionKind().Kind),
		Status:  metav1.ConditionTrue,
	})

	return nil
}

// createOrPatch is like controllerutil.CreateOrPatch but ignores status
// sub-resource.
func createOrPatch(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	key := client.ObjectKeyFromObject(obj)
	//nolint:nestif // not that complex
	if err := c.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return controllerutil.OperationResultNone, err
		}

		if f != nil {
			if err := mutate(f, key, obj); err != nil {
				return controllerutil.OperationResultNone, err
			}
		}

		if err := c.Create(ctx, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}

		return controllerutil.OperationResultCreated, nil
	}

	// Create patches for the object and its possible status.
	object, _ := obj.DeepCopyObject().(client.Object)

	objPatch := client.MergeFrom(object)

	// Create a copy of the original object as well as converting that copy to
	// unstructured data.
	before, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.DeepCopyObject())
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	// Mutate the original object.
	if f != nil {
		if err := mutate(f, key, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
	}

	// Convert the resource to unstructured to compare against our before copy.
	after, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	result := controllerutil.OperationResultNone

	if !reflect.DeepEqual(before, after) {
		// Only issue a Patch if the before and after resources (minus status) differ
		if err := c.Patch(ctx, obj, objPatch); err != nil {
			return result, err
		}

		result = controllerutil.OperationResultUpdated
	}

	return result, nil
}

// mutate wraps a MutateFn and applies validation to its result.
func mutate(f controllerutil.MutateFn, key client.ObjectKey, obj client.Object) error {
	if err := f(); err != nil {
		return err
	}

	if newKey := client.ObjectKeyFromObject(obj); key != newKey {
		return errors.New("MutateFn cannot mutate object name and/or object namespace")
	}

	//

	return nil
}

func (r *PFActiveMQReconciler) findPFActiveMQForObject(ctx context.Context, obj client.Object) []reconcile.Request {
	l := log.FromContext(ctx)

	pfCr := &pfactivemqoperatorpnetchv1alpha1.PFActiveMQ{}

	primaryObjNamespace := obj.GetLabels()[pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNamespaceLabelKey]
	primaryObjName := obj.GetLabels()[pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNameLabelKey]

	if err := r.Get(ctx, client.ObjectKey{Namespace: primaryObjNamespace, Name: primaryObjName}, pfCr); err != nil {
		l.Error(err, "failed to find PFActiveMQ resource")

		return []reconcile.Request{}
	}

	return []reconcile.Request{
		{
			Namespace: pfCr.GetNamespace(),
			Name:      pfCr.GetName(),
		},
	}
}

func (r *PFActiveMQReconciler) setAuthSecretResourcesReady(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) error {
	desiredAuthSecret := resource.NewAuthSecret(pfCR, r.Config.OperationCommonName)
	authSecret := &corev1.Secret{Name: desiredAuthSecret.Name}

	isEqual := func() bool {
		currentValues := map[string]string{}
		for key, value := range authSecret.Data {
			currentValues[key] = string(value)
		}

		return reflect.DeepEqual(desiredAuthSecret.StringData, currentValues)
	}

	return r.setResourceStatus(ctx, pfCR, authSecret, pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeAuthSecretReady, isEqual)
}

func (r *PFActiveMQReconciler) syncPVCs(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) error {
	l := log.FromContext(ctx)

	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := r.List(ctx, pvcList, client.InNamespace(pfCR.Spec.Instance.Namespace)); err != nil {
		return err
	}

	prefix := fmt.Sprintf("%s-%s-ss-", pfCR.Name, pfCR.Name)
	for _, pvc := range pvcList.Items {
		if strings.HasPrefix(pvc.Name, prefix) {
			err := r.resizePVC(ctx, pfCR, pvc)
			if err != nil {
				return err
			}
		}
	}

	pfCR.Status.LastAppliedStorage = pfCR.Spec.Instance.Storage
	condition.SetStatusCondition(ctx, &pfCR.Status.Conditions, metav1.Condition{
		Type:    pfactivemqoperatorpnetchv1alpha1.PFActiveMQConditionTypeStorageReady,
		Status:  metav1.ConditionTrue,
		Reason:  pfactivemqoperatorpnetchv1alpha1.ReasonReady,
		Message: "Storage successfully resized to " + pfCR.Spec.Instance.Storage,
	})

	if err := r.Status().Update(ctx, pfCR); err != nil {
		l.Error(err, "failed to update instance status")
		return err
	}

	return nil
}

func (r *PFActiveMQReconciler) resizePVC(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ, pvc corev1.PersistentVolumeClaim) error {
	l := log.FromContext(ctx)

	desiredSize, err := k8sresource.ParseQuantity(pfCR.Spec.Instance.Storage)
	if err != nil {
		return fmt.Errorf("failed to parse storage size %q: %w", pfCR.Spec.Instance.Storage, err)
	}

	storage := pvc.Spec.Resources.Requests["storage"]

	if storage != desiredSize {
		pvc.Spec.Resources.Requests["storage"] = desiredSize
		l.Info("updating PVC with new storage size", "pvc", fmt.Sprintf("%s/%s", pvc.Namespace, pvc.Name), "newSize", desiredSize.String())

		if err := r.Update(ctx, &pvc); err != nil {
			return fmt.Errorf("failed to update PVC %q: %w", pvc.Name, err)
		}

		if err := r.restartStatefulSet(ctx, pfCR); err != nil {
			return err
		}
	}

	return nil
}

func (r *PFActiveMQReconciler) restartStatefulSet(ctx context.Context, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) error {
	l := log.FromContext(ctx)

	statefulset := &appsv1.StatefulSet{}
	if err := r.Get(ctx, client.ObjectKey{Name: pfCR.Name + "-ss", Namespace: pfCR.Spec.Instance.Namespace}, statefulset); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	if statefulset.Spec.Template.Annotations == nil {
		statefulset.Spec.Template.Annotations = map[string]string{}
	}

	timestamp := time.Now().Format(time.RFC3339)
	statefulset.Spec.Template.Annotations["pf-activemq-operator.pnet.ch/restartedAt"] = timestamp

	if err := r.Update(ctx, statefulset); err != nil {
		return fmt.Errorf("failed to update statefulset %q: %w", statefulset.Name, err)
	}

	l.Info("restarting statefulset by updating annotation", "statefulset", fmt.Sprintf("%s/%s", statefulset.Namespace, statefulset.Name), "annotation", timestamp)

	return nil
}
