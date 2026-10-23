package instance

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"gopkg.in/yaml.v3"

	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/merge"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/policy"
	"gitlab.pnet.ch/middleware/activemq/activemq-merge-operator/internal/instance/ces"
	"gitlab.pnet.ch/middleware/activemq/activemq-merge-operator/internal/instance/gitlab"
	"gitlab.pnet.ch/middleware/activemq/activemq-merge-operator/internal/instance/k8s"
	activemqv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	blockStorageAnnotation = "postfinance.ch/opa-gatekeeper-exclusion.block-storage"
	requeueInterval        = 60 * time.Second
	// <resource-type>.<application>.[<identifier>_]<subject> for example: mqp.esbpf.In_Queue_1_Test, mqp.myapp.my-first-mqp-demo
	queueNamingRegex         = `^mqp\.[a-z0-9-]+\.[a-zA-Z0-9-]+(_[a-zA-Z0-9-]+)*$`
	esbpfIdentifierRegex     = `\.(In_Queue|Out_Queue)`
	esbpfQueueNamingRegex    = `^mqp\.[a-z0-9-]+\.(In_Queue|Out_Queue)(_[a-zA-Z0-9]+)*$`
	espfOperatorRole         = `esbpf.operator`
	configResourceLimitRatio = 0.6
)

// GitlabClient defines gitlabclient
type GitlabClient struct{}

// K8sClient defines a k8s client
type K8sClient struct{}

// CesClient defines ces endpoint
type CesClient struct {
	endpoint string
}

// CheckInstanceDelete is a policy that blocks auto-merge when an instance is deleted.
func CheckInstanceDelete(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{
			{
				Reason:  ReasonInstanceDelete,
				Message: "Instance deletion requires manual intervention. The Tagesdienst must manually merge this MR and prune the instances ArgoCD Application afterwards.",
				Close:   false,
				Path:    c.FilePath,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// CheckFileRename is a policy to ensure that no namespace file is renamed.
//
// Policy failure closes merge request.
func CheckFileRename(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationRename {
		return policy.Results{
			{
				Reason:  ReasonFileRename,
				Message: "instance renames are not allowed - delete instance first",
				Close:   true,
				Path:    c.FilePath,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// CheckPropertiesChange is a policy to ensure that changes on the properties field are not auto-merged.
func CheckPropertiesChange(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	var beforeProps []string

	if c.OldObj != nil {
		beforeProps = c.OldObj.Spec.Instance.Properties
	}

	afterProps := c.NewObj.Spec.Instance.Properties

	if !reflect.DeepEqual(beforeProps, afterProps) {
		return policy.Results{
			{
				Reason:  ReasonPropertiesChange,
				Message: "properties changed. manual review and merge is required.",
				Close:   false,
				Path:    c.FilePath,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// CheckJavaArgsChange is a policy to ensure that changes on the javaArgs field are not auto-merged.
func CheckJavaArgsChange(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	var beforeArgs []string

	if c.OldObj != nil {
		beforeArgs = c.OldObj.Spec.Instance.JavaArgs
	}

	afterArgs := c.NewObj.Spec.Instance.JavaArgs

	if !reflect.DeepEqual(beforeArgs, afterArgs) {
		return policy.Results{
			{
				Reason:  ReasonJavaArgsChange,
				Message: "javaArgs changed. manual review and merge is required.",
				Close:   false,
				Path:    c.FilePath,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// CheckMismatchingCRName is a policy to ensure that the filename and cr name matches.
func CheckMismatchingCRName(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	filename := strings.TrimSuffix(filepath.Base(c.FilePath), filepath.Ext(c.FilePath))

	if filename != c.NewObj.Name {
		return policy.Results{
			{
				Reason:  ReasonMismatchingCRName,
				Message: fmt.Sprintf("the name must match the file name. Please use %q as name.", filename),
				Close:   false,
				Path:    c.FilePath,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// CheckPlatformInNamespace is a policy to ensure that the platform in the instance name and in the namespace matches.
func CheckPlatformInNamespace(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	platform := c.NewObj.Name[0:2]
	namespace := c.NewObj.Spec.Instance.Namespace

	if !strings.HasSuffix(namespace, platform) {
		return policy.Results{
			{
				Reason:  ReasonPlatformMismatchInNamespace,
				Message: fmt.Sprintf("the platform in namespace must match the platform in instance name. Please use %q as platform.", platform),
				Close:   false,
				Path:    c.FilePath,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// CheckSplunk is a policy to ensure that the platform in the instance name and in the splunk index matches and is valid/exists.
func CheckSplunk(k k8s.NamespaceClient) policy.ChangeCheckFunc[activemqv1alpha1.PFActiveMQ] {
	return func(ctx context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
		//nolint:goconst // strings are only used for this map.
		splunkSuffix := map[string]string{"e1": "itd", "t1": "itd", "t2": "itd", "p1": "prod"}

		if c.Operation == merge.ChangeOperationDelete {
			return policy.Results{}, nil
		}

		platform := c.NewObj.Name[0:2]
		suffix := splunkSuffix[platform]
		splunkIndex := c.NewObj.Spec.Instance.SplunkIndex

		if !strings.HasSuffix(splunkIndex, suffix) {
			return policy.Results{
				{
					Reason:  ReasonPlatformMismatchInSplunk,
					Message: fmt.Sprintf("the platform in instance name must match the platform in splunk index. Please use %q as suffix for splunk.", suffix),
					Close:   false,
					Path:    c.FilePath,
				},
			}, nil
		}

		// create a minimal namespace resource with collectord annotation (Splunk index)
		nsName := c.NewObj.Spec.Instance.Namespace
		nsAnnotations := map[string]string{"collectord.io/index": splunkIndex}

		err := k.DryRunNamespace(ctx, nsName, nsAnnotations)
		if err != nil {
			// catch http 403 response from admission controller
			if errors.IsForbidden(err) {
				return policy.Results{
					{
						Reason:  ReasonSplunkIndexMissing,
						Message: fmt.Sprintf("the provided splunk index %s isn't valid or doesn't exist yet on this k8s cluster. Please check your [SpaaS configuration](https://gitlab.pnet.ch/observability/logging/spaas).", splunkIndex),
						Close:   false,
						Path:    c.FilePath,
						Requeue: requeueInterval,
					},
				}, nil
			}
			// catch any other error
			return policy.Results{}, fmt.Errorf("validation for splunk index failed: %w", err)
		}

		return policy.Results{}, nil
	}
}

// CheckPlatformInPermissions is a policy to ensure that the platform in the instance name and in the permissions matches.
func CheckPlatformInPermissions(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	platform := c.NewObj.Name[0:2]

	identities := map[string]bool{}

	for i := range c.NewObj.Spec.Instance.Queues {
		for _, permissions := range c.NewObj.Spec.Instance.Queues[i].GetPermissions() {
			for _, permission := range permissions {
				identities[permission] = true
			}
		}
	}

	for identity := range identities {
		if strings.HasPrefix(identity, "a-pf-identity") {
			if !strings.HasSuffix(identity, platform) {
				return policy.Results{
					{
						Reason:  ReasonPlatformMismatchInPermission,
						Message: fmt.Sprintf("the platform in permissions must match the platform in instance name. Please use %q as platform.", platform),
						Close:   false,
						Path:    c.FilePath,
						Requeue: requeueInterval,
					},
				}, nil
			}
		}
	}

	return policy.Results{}, nil
}

// CheckTargetNamespace is a policy to ensure that the target namespace with annotation exists.
//
// Policy failure closes merge request.
func CheckTargetNamespace(g gitlab.NamespaceClient) policy.ChangeCheckFunc[activemqv1alpha1.PFActiveMQ] {
	return func(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
		if c.Operation == merge.ChangeOperationDelete {
			return policy.Results{}, nil
		}

		dir := filepath.Dir(c.FilePath)
		clusterID := filepath.Base(dir)
		targetNamespace := c.NewObj.Spec.Instance.Namespace

		project, retry, err := g.GetNamespaceConfig(clusterID, targetNamespace)
		if err != nil && retry {
			return policy.Results{
				{
					Reason:  ReasonInvalidNamespace,
					Message: fmt.Sprintf("the namespace `%s` does not exist.", targetNamespace),
					Close:   false,
					Path:    c.FilePath,
					Requeue: requeueInterval,
				},
			}, nil
		}

		if err != nil {
			return policy.Results{}, fmt.Errorf("failed to get project of target namespace from namespaces repo: %w", err)
		}

		//nolint:errcheck // is a check here really needed
		annotations := project["spec"].(map[string]any)["namespace"].(map[string]any)["annotations"]
		//nolint:errcheck // is a check here really needed
		if annotations == nil || annotations.(map[string]any)[blockStorageAnnotation] == nil {
			return policy.Results{
				{
					Reason:  ReasonBlockStorageAnnotationMissing,
					Message: fmt.Sprintf("the block storage annotation in namespace `%s` is missing.", targetNamespace),
					Close:   false,
					Path:    c.FilePath,
					Requeue: requeueInterval,
				},
			}, nil
		}

		//nolint:errcheck // is a check here really needed
		if c.NewObj.Spec.Instance.Storage != annotations.(map[string]any)[blockStorageAnnotation] {
			return policy.Results{
				{
					Reason:  ReasonBlockStorageAnnotationInvalid,
					Message: fmt.Sprintf("the block storage annotation value in namespace `%s` does not match the instance storage.", targetNamespace),
					Close:   false,
					Path:    c.FilePath,
					Requeue: requeueInterval,
				},
			}, nil
		}

		return policy.Results{}, nil
	}
}

// CheckIdentity is a policy to ensure that the certificate of the defined identities exists.
func CheckIdentity(cc ces.IdentityClient) policy.ChangeCheckFunc[activemqv1alpha1.PFActiveMQ] {
	return func(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
		if c.Operation == merge.ChangeOperationDelete {
			return policy.Results{}, nil
		}

		identities := map[string]bool{}

		for i := range c.NewObj.Spec.Instance.Queues {
			for _, permissions := range c.NewObj.Spec.Instance.Queues[i].GetPermissions() {
				for _, permission := range permissions {
					identities[permission] = true
				}
			}
		}

		for identity := range identities {
			if strings.HasPrefix(identity, "a-pf-identity") {
				exists, err := cc.IdentityExist(identity)
				if err != nil {
					return policy.Results{}, err
				}

				if !exists {
					return policy.Results{
						{
							Reason:  ReasonCertificateMissing,
							Message: fmt.Sprintf("the certificate `%s` does not yet exist in CES. Please provide target VM or Host for the certificate in a comment.", identity),
							Close:   false,
							Path:    c.FilePath,
							Requeue: requeueInterval,
						},
					}, nil
				}
			}
		}

		return policy.Results{}, nil
	}
}

// CheckQueueRename is a policy to ensure queue names are not renamed on modifications.
func CheckQueueRename(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation != merge.ChangeOperationModify || c.OldObj == nil || c.NewObj == nil {
		return policy.Results{}, nil
	}

	oldQueueNames := map[string]int{}
	newQueueNames := map[string]int{}

	for _, queue := range c.OldObj.Spec.Instance.Queues {
		oldQueueNames[queue.Name]++
	}

	for _, queue := range c.NewObj.Spec.Instance.Queues {
		newQueueNames[queue.Name]++
	}

	removedNames := []string{}
	addedNames := []string{}

	for name, oldCount := range oldQueueNames {
		diff := oldCount - newQueueNames[name]
		for range diff {
			removedNames = append(removedNames, name)
		}
	}

	for name, newCount := range newQueueNames {
		diff := newCount - oldQueueNames[name]
		for range diff {
			addedNames = append(addedNames, name)
		}
	}

	if len(addedNames) == 0 && len(removedNames) == 0 {
		return policy.Results{}, nil
	}

	if len(addedNames) == len(removedNames) {
		slices.Sort(removedNames)
		slices.Sort(addedNames)

		return policy.Results{
			{
				Reason:  ReasonQueueRenamed,
				Message: fmt.Sprintf("queue rename detected. removed queue name(s): `%s`, added queue name(s): `%s`. queue renames are not allowed. Ensure the queue is emptied first and that no producers or consumers are active, then delete it. Afterwards, open a new merge request to create the new queue.", strings.Join(removedNames, ", "), strings.Join(addedNames, ", ")),
				Close:   false,
				Path:    c.FilePath,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// CheckQueueNaming is a policy to ensure that the queue name matches the naming convention.
func CheckQueueNaming(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	queuePattern := regexp.MustCompile(queueNamingRegex)
	ebspfIdentifier := regexp.MustCompile(esbpfIdentifierRegex)
	esbpfPattern := regexp.MustCompile(esbpfQueueNamingRegex)

	for _, queue := range c.NewObj.Spec.Instance.Queues {
		// check the queue name against the general naming convention
		if !queuePattern.MatchString(queue.Name) {
			return policy.Results{
				{
					Reason:  ReasonInvalidQueueName,
					Message: fmt.Sprintf("the queue name `%s` does not follow the general naming convention `%s`, examples: mqp.esbpf.In_Queue_1_Test, mqp.myapp.my-first-mqp-demo, mqp.myapp.My_Test_Queue", queue.Name, queueNamingRegex),
					Close:   false,
					Path:    c.FilePath,
				},
			}, nil
		}

		// check if the queue name contains the reserved ESBPF identifiers In_Queue or Out_Queue
		if ebspfIdentifier.MatchString(queue.Name) {
			// check if the queue name matches the ESBPF naming convention
			if !esbpfPattern.MatchString(queue.Name) {
				return policy.Results{
					{
						Reason:  ReasonInvalidQueueName,
						Message: fmt.Sprintf("the queue name `%s` does not follow the ESBPF naming convention for `%s`, examples: mqp.esbpf.In_Queue_ECM_Trigger, mqp.myapp.Out_Queue", queue.Name, esbpfQueueNamingRegex),
						Close:   false,
						Path:    c.FilePath,
					},
				}, nil
			}
		}

		// check if the queue name is not a DLQ, as these are implicit created per queue by the operator
		if strings.HasSuffix(queue.Name, "DLQ") {
			return policy.Results{
				{
					Reason:  ReasonInvalidQueueName,
					Message: fmt.Sprintf("the queue `%s` ends with DLQ. DLQs are created implicit for every queue and must not be defined", queue.Name),
					Close:   false,
					Path:    c.FilePath,
				},
			}, nil
		}
	}

	return policy.Results{}, nil
}

// CheckQueueDuplicate is a policy to ensure that no queue name is defined more than once.
func CheckQueueDuplicate(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	names := make([]string, 0, len(c.NewObj.Spec.Instance.Queues))
	for _, queue := range c.NewObj.Spec.Instance.Queues {
		names = append(names, queue.Name)
	}

	slices.Sort(names)

	results := policy.Results{}

	// Check for duplicates
	for i := 0; i < len(names); {
		j := i + 1
		for j < len(names) && names[j] == names[i] {
			j++
		}

		if count := j - i; count > 1 {
			results = append(results, policy.Result{
				Reason:  ReasonDuplicateQueue,
				Message: fmt.Sprintf("the queue `%s` is defined %d times, each queue name must be unique", names[i], count),
				Close:   false,
				Path:    c.FilePath,
			})
		}

		i = j
	}

	return results, nil
}

// CheckESBPFOperator is a policy to ensure that ESBPF operator role is defined when Queue with ESBPF identifier exists.
func CheckESBPFOperator(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	if slices.Contains(c.NewObj.Spec.Roles.Operators, espfOperatorRole) {
		return policy.Results{}, nil
	}

	ebspfIdentifier := regexp.MustCompile(esbpfIdentifierRegex)

	for _, queue := range c.NewObj.Spec.Instance.Queues {
		// check if the queue name contains the reserved ESBPF identifiers In_Queue or Out_Queue
		if ebspfIdentifier.MatchString(queue.Name) {
			return policy.Results{
				{
					Reason:  ReasonMissingESBPFOperatorRole,
					Message: fmt.Sprintf("the queue %s contains the ESBPF identifier, therefore the ESBPF operator role %s must be added to Spec.Roles.Operators.", queue.Name, espfOperatorRole),
					Close:   false,
					Path:    c.FilePath,
				},
			}, nil
		}
	}

	return policy.Results{}, nil
}

// CheckQueuePageFullMessagePolicyConfigChange is a policy to ensure that the queue configuration is valid.
func CheckQueuePageFullMessagePolicyConfigChange(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	for _, queue := range c.NewObj.Spec.Instance.Queues {
		if queue.Config == nil || queue.Config.PageFullMessagePolicy == "" {
			return policy.Results{}, nil
		}

		switch queue.Config.PageFullMessagePolicy {
		case "FAIL", "DROP":
		default:
			return policy.Results{
				{
					Reason:  ReasonInvalidQueueConfig,
					Message: fmt.Sprintf("the queue `%s` has an invalid pageFullMessagePolicy `%s`. Valid values are: FAIL, DROP.", queue.Name, queue.Config.PageFullMessagePolicy),
					Close:   false,
					Path:    c.FilePath,
				},
			}, nil
		}

		for _, otherQueue := range c.NewObj.Spec.Instance.Queues {
			if otherQueue.Name == queue.Name {
				continue
			}

			if (otherQueue.Config == nil || otherQueue.Config.PageFullMessagePolicy == "") && queue.Config.PageFullMessagePolicy != "" {
				return policy.Results{
					{
						Reason:  ReasonInvalidQueueConfig,
						Message: fmt.Sprintf("the queue `%s` has pageFullMessagePolicy set to `%s`, but the queue `%s` does not have pageFullMessagePolicy set. Please set pageFullMessagePolicy for all queues or remove it from all queues.", queue.Name, queue.Config.PageFullMessagePolicy, otherQueue.Name),
						Close:   false,
						Path:    c.FilePath,
					},
				}, nil
			}
		}
	}

	return policy.Results{}, nil
}

// CheckQueueMaxSizeBytesConfigChange is a policy to ensure that the queue maxSizeBytes configuration is valid and the total maxSizeBytes of all queues does not exceed the allowed memory for config (60% of instance memory limit).
func CheckQueueMaxSizeBytesConfigChange(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	memoryLimits := c.NewObj.Spec.Instance.Resources.Limits.Memory()
	memoryLimitsStr := memoryLimits.String()

	maxBytes, err := ParseSizeToBytes(memoryLimitsStr)
	if err != nil {
		return policy.Results{}, err
	}

	allowedBytes := int(float64(maxBytes) * configResourceLimitRatio)

	totalMaxSizeBytes := 0

	for _, queue := range c.NewObj.Spec.Instance.Queues {
		if queue.Config == nil || queue.Config.MaxSizeBytes == 0 {
			continue
		}

		if queue.Config.MaxSizeBytes < 0 {
			return policy.Results{
				{
					Reason:  ReasonInvalidQueueConfig,
					Message: fmt.Sprintf("the queue `%s` has an invalid maxSizeBytes `%d`. The value must be a positive integer.", queue.Name, queue.Config.MaxSizeBytes),
					Close:   false,
				},
			}, nil
		}

		if queue.Config.MaxSizeBytes > 0 {
			totalMaxSizeBytes += queue.Config.MaxSizeBytes
		}

		for _, otherQueue := range c.NewObj.Spec.Instance.Queues {
			if otherQueue.Name == queue.Name {
				continue
			}

			if (otherQueue.Config == nil || (otherQueue.Config.MaxSizeBytes <= 0)) && queue.Config.MaxSizeBytes > 0 {
				return policy.Results{
					{
						Reason:  ReasonInvalidQueueConfig,
						Message: fmt.Sprintf("the queue `%s` has maxSizeBytes set to `%d`, but the queue `%s` does not have maxSizeBytes set. Please set maxSizeBytes for all queues or remove it from all queues.", queue.Name, queue.Config.MaxSizeBytes, otherQueue.Name),
						Close:   false,
						Path:    c.FilePath,
					},
				}, nil
			}
		}
	}

	if allowedBytes > 0 && totalMaxSizeBytes > allowedBytes {
		return policy.Results{
			{
				Reason:  ReasonInvalidQueueConfig,
				Message: fmt.Sprintf("the total maxSizeBytes of all queues `%d` exceeds the allowed memory for config `%d` (60%% of instance limit). Please reduce the maxSizeBytes or increase the memory limit.", totalMaxSizeBytes, allowedBytes),
				Close:   false,
				Path:    c.FilePath,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// CheckQueueExpiryDelayConfigChange is a policy to ensure that the queue expiryDelay configuration is valid.
func CheckQueueExpiryDelayConfigChange(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	for _, queue := range c.NewObj.Spec.Instance.Queues {
		if queue.Config != nil && queue.Config.ExpiryDelay < 0 {
			return policy.Results{
				{
					Reason:  ReasonInvalidQueueConfig,
					Message: fmt.Sprintf("the queue `%s` has an invalid expiryDelay `%d`. The value must be a positive integer.", queue.Name, queue.Config.ExpiryDelay),
					Close:   false,
				},
			}, nil
		}

		for _, otherQueue := range c.NewObj.Spec.Instance.Queues {
			if otherQueue.Name == queue.Name {
				continue
			}

			if (otherQueue.Config == nil || otherQueue.Config.ExpiryDelay <= 0) && queue.Config != nil && queue.Config.ExpiryDelay > 0 {
				return policy.Results{
					{
						Reason:  ReasonInvalidQueueConfig,
						Message: fmt.Sprintf("the queue `%s` has expiryDelay set to `%d`, but the queue `%s` does not have expiryDelay set. Please set expiryDelay for all queues or remove it from all queues.", queue.Name, queue.Config.ExpiryDelay, otherQueue.Name),
						Close:   false,
						Path:    c.FilePath,
					},
				}, nil
			}
		}
	}

	return policy.Results{}, nil
}

// CheckQueuePageLimitBytesConfigChange is a policy to ensure that the queue pageLimitBytes configuration is valid and the total pageLimitBytes of all queues does not exceed the allowed storage for config (60% of instance storage).
func CheckQueuePageLimitBytesConfigChange(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	storageLimits := c.NewObj.Spec.Instance.Storage

	maxBytes, err := ParseSizeToBytes(storageLimits)
	if err != nil {
		return policy.Results{}, err
	}

	allowedBytes := int(float64(maxBytes) * configResourceLimitRatio)

	totalPageLimitBytes := 0

	for _, queue := range c.NewObj.Spec.Instance.Queues {
		if queue.Config == nil || queue.Config.PageLimitBytes == 0 {
			continue
		}

		if queue.Config.PageLimitBytes > 0 {
			totalPageLimitBytes += queue.Config.PageLimitBytes
		}

		if queue.Config.PageLimitBytes < 0 {
			return policy.Results{
				{
					Reason:  ReasonInvalidQueueConfig,
					Message: fmt.Sprintf("the queue `%s` has an invalid pageLimitBytes `%d`. The value must be a positive integer.", queue.Name, queue.Config.PageLimitBytes),
					Close:   false,
				},
			}, nil
		}

		for _, otherQueue := range c.NewObj.Spec.Instance.Queues {
			if otherQueue.Name == queue.Name {
				continue
			}

			if (otherQueue.Config == nil || (otherQueue.Config.PageLimitBytes <= 0)) && queue.Config.PageLimitBytes > 0 {
				return policy.Results{
					{
						Reason:  ReasonInvalidQueueConfig,
						Message: fmt.Sprintf("the queue `%s` has pageLimitBytes set to `%d`, but the queue `%s` does not have pageLimitBytes set. Please set pageLimitBytes for all queues or remove it from all queues.", queue.Name, queue.Config.PageLimitBytes, otherQueue.Name),
						Close:   false,
						Path:    c.FilePath,
					},
				}, nil
			}
		}
	}

	if allowedBytes > 0 && totalPageLimitBytes > allowedBytes {
		return policy.Results{
			{
				Reason:  ReasonInvalidQueueConfig,
				Message: fmt.Sprintf("the total pageLimitBytes of all queues `%d` exceeds the allowed storage for config `%d` (60%% of instance limit). Please reduce the pageLimitBytes or increase storage.", totalPageLimitBytes, allowedBytes),
				Close:   false,
				Path:    c.FilePath,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// CheckAllowedPermissions is a policy to ensure that only allowed permissions are used in the queue configuration.
func CheckAllowedPermissions(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	for _, queue := range c.NewObj.Spec.Instance.Queues {
		for permission := range queue.GetPermissions() {
			switch permission {
			case "send", "consume", "browse":
			default:
				return policy.Results{
					{
						Reason:  ReasonInvalidPermission,
						Message: fmt.Sprintf("the queue `%s` has an invalid permission `%s`. Valid permissions are: Send, Consume, Browse.", queue.Name, permission),
						Close:   false,
						Path:    c.FilePath,
					},
				}, nil
			}
		}
	}

	return policy.Results{}, nil
}

// CheckOperatorRole is a policy to ensure that the operator role is defined when queues are defined.
func CheckOperatorRole(_ context.Context, _ *merge.Request, c *merge.ParsedChange[activemqv1alpha1.PFActiveMQ]) (policy.Results, error) {
	if c.Operation == merge.ChangeOperationDelete {
		return policy.Results{}, nil
	}

	if len(c.NewObj.Spec.Roles.Operators) == 0 {
		return policy.Results{
			{
				Reason:  ReasonMissingOperatorRole,
				Message: "an operator role must be defined.",
				Close:   false,
			},
		}, nil
	}

	return policy.Results{}, nil
}

// IdentityExist checks if certificate for given identity exists
func (c CesClient) IdentityExist(identity string) (bool, error) {
	resp, err := http.Get(c.endpoint + identity)
	if err != nil {
		return false, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	return resp.StatusCode == http.StatusOK, nil
}

// GetNamespaceConfig returns the namespace config
//
//nolint:gocritic // need to name error
func (g GitlabClient) GetNamespaceConfig(cluster, namespace string) (project map[string]any, retry bool, err error) {
	url := fmt.Sprintf("https://gitlab.pnet.ch/kubernetes/namespaces/-/raw/main/projects/%s/%s.yaml?ref_type=heads", cluster, namespace)

	// retries if connection error from client is returned or 500-range response code is received
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 5 * time.Second

	client := retryClient.StandardClient()

	resp, err := client.Get(url)
	if err != nil {
		return nil, false, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read response body: %w", err)
	}

	switch status := resp.StatusCode; status {
	case http.StatusNotFound:
		return nil, true, fmt.Errorf("project yaml with cluster: %s and namespace: %s could not be fetched", cluster, namespace)
	}

	var result map[string]any

	err = yaml.Unmarshal(body, &result)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return result, false, nil
}

// DryRunNamespace performs a dry-run validation of a namespace creation.
// Returns an error if the namespace would be rejected by admission controllers
func (k K8sClient) DryRunNamespace(ctx context.Context, namespace string, annotations map[string]string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get k8s config object containing the service account from the pod: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create a client set from the cluster config object: %w", err)
	}

	ns := &corev1.Namespace{
		Name:        namespace,
		Annotations: annotations,
	}

	_, err = clientset.CoreV1().Namespaces().Create(
		ctx,
		ns,
		metav1.CreateOptions{
			DryRun: []string{metav1.DryRunAll},
		},
	)

	return err
}
