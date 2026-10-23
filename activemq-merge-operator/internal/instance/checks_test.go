package instance

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/merge"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/policy"
	clusterinfov1 "gitlab.pnet.ch/linux/grpc/connect/go/postfinance/clusterinfo/v1"
	"gitlab.pnet.ch/linux/grpc/connect/go/postfinance/clusterinfo/v1/clusterinfov1connect"
	cesMocks "gitlab.pnet.ch/middleware/activemq/activemq-merge-operator/gen/mock/ces"
	gitlabMocks "gitlab.pnet.ch/middleware/activemq/activemq-merge-operator/gen/mock/gitlab"
	k8sMocks "gitlab.pnet.ch/middleware/activemq/activemq-merge-operator/gen/mock/k8s"
	activemqv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
)

func TestCheckInstanceDelete(t *testing.T) {
	tests := []struct {
		operation merge.ChangeOperation
		want      policy.Results
	}{
		{
			operation: merge.ChangeOperationCreate,
			want:      policy.Results{},
		},
		{
			operation: merge.ChangeOperationDelete,
			want:      createResult(false, ReasonInstanceDelete, "Instance deletion requires manual intervention. The Tagesdienst must manually merge this MR and prune the instances ArgoCD Application afterwards."),
		},
	}

	for _, tt := range tests {
		c := merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
			Operation: tt.operation,
		}

		testName := "operation=" + tt.operation.String()

		t.Run(testName, func(t *testing.T) {
			r, err := CheckInstanceDelete(context.TODO(), nil, &c)
			require.NoError(t, err)
			require.Equal(t, tt.want, r)
		})
	}
}

func TestCheckFileRename(t *testing.T) {
	tests := []struct {
		operation merge.ChangeOperation
		want      policy.Results
	}{
		{
			operation: merge.ChangeOperationCreate,
			want:      policy.Results{},
		},
		{
			operation: merge.ChangeOperationDelete,
			want:      policy.Results{},
		},
		{
			operation: merge.ChangeOperationModify,
			want:      policy.Results{},
		},
		{
			operation: merge.ChangeOperationRename,
			want:      createResult(true, ReasonFileRename, "instance renames are not allowed - delete instance first"),
		},
	}

	for _, tt := range tests {
		c := merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
			Operation: tt.operation,
		}

		testName := "operation=" + tt.operation.String()

		t.Run(testName, func(t *testing.T) {
			r, err := CheckFileRename(context.TODO(), nil, &c)
			require.NoError(t, err)
			require.Equal(t, tt.want, r)
		})
	}
}

func createResult(isClose bool, reason string, messages ...string) policy.Results {
	results := policy.Results{}

	for _, msg := range messages {
		results = append(results, policy.Result{
			Reason:  reason,
			Close:   isClose,
			Message: msg,
		})
	}

	return results
}

func TestCheckPropertiesChange(t *testing.T) {
	res := createResult(false, ReasonPropertiesChange, "properties changed. manual review and merge is required.")

	type args struct {
		c merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string
		args args
		want policy.Results
	}{
		{
			name: "new_file_with_properties",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationCreate,
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Properties: []string{"foo=bar"},
							},
						},
					},
				},
			},
			want: res,
		},
		{
			name: "delete_file_with_properties",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationDelete,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Properties: []string{"foo=bar"},
							},
						},
					},
				},
			},
			want: policy.Results{},
		},
		{
			name: "change_without_property_change",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationModify,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Properties: []string{"foo=bar"},
							},
						},
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Replicas:   5,
								Properties: []string{"foo=bar"},
							},
						},
					},
				},
			},
			want: policy.Results{},
		},
		{
			name: "change_with_property_change",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationModify,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Properties: []string{"foo=bar"},
							},
						},
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Properties: []string{"foo=bar", "new=property"},
							},
						},
					},
				},
			},
			want: res,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := CheckPropertiesChange(context.TODO(), nil, &tt.args.c)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckPropertiesChange() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckJavaArgsChange(t *testing.T) {
	res := createResult(false, ReasonJavaArgsChange, "javaArgs changed. manual review and merge is required.")

	type args struct {
		c merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string
		args args
		want policy.Results
	}{
		{
			name: "new_file_with_java_args",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationCreate,
					},
					OldObj: nil,
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								JavaArgs: []string{"-XX:MaxRAMPercentage=75"},
							},
						},
					},
				},
			},
			want: res,
		},
		{
			name: "delete_file_with_java_args",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationDelete,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								JavaArgs: []string{"-XX:MaxRAMPercentage=75"},
							},
						},
					},
					NewObj: nil,
				},
			},
			want: policy.Results{},
		},
		{
			name: "change_without_java_args_change",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationModify,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								JavaArgs: []string{"-XX:MaxRAMPercentage=75"},
							},
						},
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Replicas: 5,
								JavaArgs: []string{"-XX:MaxRAMPercentage=75"},
							},
						},
					},
				},
			},
			want: policy.Results{},
		},
		{
			name: "change_with_java_args_change",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationModify,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								JavaArgs: []string{"-XX:MaxRAMPercentage=75"},
							},
						},
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								JavaArgs: []string{"-XX:MaxRAMPercentage=75", "-XX:MaxRAMPercentage=85"},
							},
						},
					},
				},
			},
			want: res,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := CheckJavaArgsChange(context.TODO(), nil, &tt.args.c)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckJavaArgsChange() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckMismatchingCRName(t *testing.T) {
	type args struct {
		c merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string
		args args
		want policy.Results
	}{
		{
			name: "valid",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						FilePath:  "instances/e1-k8s-pfnet-lab-a/e1-mqp-middleware-a.yaml",
						Operation: merge.ChangeOperationCreate,
					},
					OldObj: nil,
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-middleware-a",
					},
				},
			},
			want: policy.Results{},
		},
		{
			name: "invalid",
			args: args{
				c: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						FilePath:  "instances/e1-k8s-pfnet-lab-a/e1-mqp-middleware-a.yaml",
						Operation: merge.ChangeOperationCreate,
					},
					OldObj: nil,
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "my-not-matching-name",
					},
				},
			},
			want: policy.Results{
				{
					Reason:  ReasonMismatchingCRName,
					Message: "the name must match the file name. Please use \"e1-mqp-middleware-a\" as name.",
					Path:    "instances/e1-k8s-pfnet-lab-a/e1-mqp-middleware-a.yaml",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := CheckMismatchingCRName(context.TODO(), nil, &tt.args.c)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckMismatchingCRName() got = %v, want %v", got, tt.want)
			}
		})
	}
}

type clusterInfoClient struct {
	clusterinfov1connect.UnimplementedInfoAPIHandler
	namespaces []*clusterinfov1.Namespace
	err        error
}

func (c clusterInfoClient) ListNamespaces(context.Context, *connect.Request[clusterinfov1.ListNamespacesRequest]) (*connect.Response[clusterinfov1.ListNamespacesResponse], error) {
	if c.err != nil {
		return nil, c.err
	}

	return &connect.Response[clusterinfov1.ListNamespacesResponse]{Msg: &clusterinfov1.ListNamespacesResponse{
		Namespaces: c.namespaces,
	}}, nil
}

func TestCheckTargetNamespace(t *testing.T) {
	validProject := map[string]any{
		"spec": map[string]any{
			"namespace": map[string]any{
				"annotations": map[string]any{
					"postfinance.ch/opa-gatekeeper-exclusion.block-storage": "2Gi",
				},
			},
		},
	}

	invalidProject := map[string]any{
		"spec": map[string]any{
			"namespace": map[string]any{
				"annotations": map[string]any{},
			},
		},
	}

	namespaceClient := &gitlabMocks.NamespaceClient{}
	namespaceClient.EXPECT().GetNamespaceConfig("valid", "appl-mqp-e2e-e1").Return(validProject, false, nil)
	namespaceClient.EXPECT().GetNamespaceConfig("invalid", "not-existing-namespace").Return(nil, true, fmt.Errorf("test"))
	namespaceClient.EXPECT().GetNamespaceConfig("invalid", "fetching-error").Return(nil, false, fmt.Errorf("test"))
	namespaceClient.EXPECT().GetNamespaceConfig("invalid", "noAnnotation").Return(invalidProject, false, nil)

	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name    string
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						FilePath: "instances/valid/appl-mqp-e2e-e1.yaml",
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{Namespace: "appl-mqp-e2e-e1", Storage: "2Gi"}},
					}},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "namespace_not_existing",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						FilePath: "instances/invalid/not-existing-namespace.yaml",
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{Namespace: "not-existing-namespace"}},
					}},
			},
			want: policy.Results{
				{
					Reason:  ReasonInvalidNamespace,
					Message: "the namespace `not-existing-namespace` does not exist.",
					Path:    "instances/invalid/not-existing-namespace.yaml",
					Requeue: requeueInterval,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "error_fetching_project",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						FilePath: "instances/invalid/fetching-error.yaml",
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{Namespace: "fetching-error"}},
					}},
			},
			want:    policy.Results{},
			wantErr: assert.Error,
		},
		{
			name: "missing",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						FilePath: "instances/invalid/noAnnotation.yaml",
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{Namespace: "noAnnotation"}},
					}},
			},
			want: policy.Results{
				{
					Reason:  ReasonBlockStorageAnnotationMissing,
					Message: "the block storage annotation in namespace `noAnnotation` is missing.",
					Path:    "instances/invalid/noAnnotation.yaml",
					Requeue: requeueInterval,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						FilePath: "instances/valid/appl-mqp-e2e-e1.yaml",
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{Namespace: "appl-mqp-e2e-e1", Storage: "4Gi"}},
					}},
			},
			want: policy.Results{
				{
					Reason:  ReasonBlockStorageAnnotationInvalid,
					Message: "the block storage annotation value in namespace `appl-mqp-e2e-e1` does not match the instance storage.",
					Path:    "instances/valid/appl-mqp-e2e-e1.yaml",
					Requeue: requeueInterval,
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckTargetNamespace(namespaceClient)(context.TODO(), nil, &tt.args.change)
			tt.wantErr(t, err)

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("CheckTargetNamespace() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestCheckIdentity(t *testing.T) {
	identityClient := &cesMocks.IdentityClient{}
	identityClient.EXPECT().IdentityExist("a-pf-identity-mqp-e2e-e1").Return(true, nil)
	identityClient.EXPECT().IdentityExist("a-pf-identity-mqp-e2e-p1").Return(false, nil)

	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name    string
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid_with_identities",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name: "mqp.egw.In_Queue",
									Permission: activemqv1alpha1.Permission{
										Send:    []string{"a-pf-identity-mqp-e2e-e1"},
										Consume: []string{"a-pf-identity-mqp-e2e-e1"},
									}},
									{
										Name: "mqp.egw.Out_Queue",
										Permission: activemqv1alpha1.Permission{
											Send:    []string{"a-pf-identity-mqp-e2e-e1"},
											Consume: []string{"bancs/appl_xyz"},
										}},
								},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "valid_no_queues",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "valid_no_permissions",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name:       "mqp.egw.In_Queue",
									Permission: activemqv1alpha1.Permission{},
								},
									{
										Name:       "mqp.egw.Out_Queue",
										Permission: activemqv1alpha1.Permission{},
									},
								},
							},
						},
					},
				},
			},

			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name: "mqp.egw.In_Queue",
									Permission: activemqv1alpha1.Permission{
										Send:    []string{"a-pf-identity-mqp-e2e-p1"},
										Consume: []string{"a-pf-identity-mqp-e2e-e1"},
									}},
									{
										Name: "mqp.egw.Out_Queue",
										Permission: activemqv1alpha1.Permission{
											Send:    []string{"a-pf-identity-mqp-e2e-e1"},
											Consume: []string{"bancs/appl_xyz"},
										}},
								},
							},
						},
					},
				},
			},
			want: policy.Results{
				{
					Reason:  ReasonCertificateMissing,
					Message: "the certificate `a-pf-identity-mqp-e2e-p1` does not yet exist in CES. Please provide target VM or Host for the certificate in a comment.",
					Path:    "",
					Requeue: requeueInterval,
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckIdentity(identityClient)(context.TODO(), nil, &tt.args.change)
			tt.wantErr(t, err)

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("CheckTargetNamespace() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestCheckPlatformInPermissions(t *testing.T) {
	identityClient := &cesMocks.IdentityClient{}
	identityClient.EXPECT().IdentityExist("a-pf-identity-mqp-e2e-e1").Return(true, nil)
	identityClient.EXPECT().IdentityExist("a-pf-identity-mqp-e2e-p1").Return(false, nil)

	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name    string
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid_with_identities",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name: "mqp.egw.In_Queue",
									Permission: activemqv1alpha1.Permission{
										Send:    []string{"a-pf-identity-mqp-e2e-e1"},
										Consume: []string{"a-pf-identity-mqp-e2e-e1"},
									}},
									{
										Name: "mqp.egw.Out_Queue",
										Permission: activemqv1alpha1.Permission{
											Send:    []string{"a-pf-identity-mqp-e2e-e1"},
											Consume: []string{"a-pf-identity-mqp-e2e-e1"},
										}},
								},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "valid_no_queues",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "valid_no_permissions",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name:       "mqp.egw.In_Queue",
									Permission: activemqv1alpha1.Permission{},
								},
									{
										Name:       "mqp.egw.Out_Queue",
										Permission: activemqv1alpha1.Permission{},
									},
								},
							},
						},
					},
				},
			},

			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name: "mqp.egw.In_Queue",
									Permission: activemqv1alpha1.Permission{
										Send:    []string{"a-pf-identity-mqp-e2e-p1"},
										Consume: []string{"a-pf-identity-mqp-e2e-t1"},
									}},
								},
							},
						},
					},
				},
			},
			want: policy.Results{
				{
					Reason:  ReasonPlatformMismatchInPermission,
					Message: "the platform in permissions must match the platform in instance name. Please use \"t1\" as platform.",
					Path:    "",
					Requeue: requeueInterval,
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckPlatformInPermissions(context.TODO(), nil, &tt.args.change)
			tt.wantErr(t, err)

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("CheckPlatform() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestCheckPlatformInNamespace(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}

	tests := []struct {
		name    string
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{Namespace: "appl-esbpf-t1"}},
						Name: "t1-esbpf-a",
					}},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{Namespace: "appl-esbpf-p1"}},
						Name: "t1-esbpf-a",
					}},
			},
			want:    createResult(false, ReasonPlatformMismatchInNamespace, "the platform in namespace must match the platform in instance name. Please use \"t1\" as platform."),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckPlatformInNamespace(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckPlatformInNamespace(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckPlatformInNamespace() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPlatformInSplunkIndex(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(*k8sMocks.NamespaceClient)
		want      policy.Results
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "midwitd", Namespace: "appl-mqp-e2e-e1"}},
						Name: "t1-midw-a",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				m.EXPECT().DryRunNamespace(
					mock.Anything,
					"appl-mqp-e2e-e1",
					map[string]string{"collectord.io/index": "midwitd"},
				).Return(nil).Once()
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "esbprod", Namespace: "appl-mqp-e2e-e1"}},
						Name: "t1-esbpf-a",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				// No mock setup needed - validation fails before DryRunNamespace
			},
			want:    createResult(false, ReasonPlatformMismatchInSplunk, "the platform in instance name must match the platform in splunk index. Please use \"itd\" as suffix for splunk."),
			wantErr: assert.NoError,
		},
		{
			name: "no_splunk",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-esbpf-a",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				// No mock setup needed
			},
			want:    createResult(false, ReasonPlatformMismatchInSplunk, "the platform in instance name must match the platform in splunk index. Please use \"itd\" as suffix for splunk."),
			wantErr: assert.NoError,
		},
		{
			name: "e1_platform_with_itd_suffix_valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "test-itd", Namespace: "appl-mqp-e2e-e1"}},
						Name: "e1-test-instance",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				m.EXPECT().DryRunNamespace(
					mock.Anything,
					"appl-mqp-e2e-e1",
					map[string]string{"collectord.io/index": "test-itd"},
				).Return(nil).Once()
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "t2_platform_with_itd_suffix_valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "test-itd", Namespace: "appl-mqp-e2e-e1"}},
						Name: "t2-test-instance",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				m.EXPECT().DryRunNamespace(
					mock.Anything,
					"appl-mqp-e2e-e1",
					map[string]string{"collectord.io/index": "test-itd"},
				).Return(nil).Once()
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "p1_platform_with_prod_suffix_valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "test-prod", Namespace: "appl-mqp-e2e-e1"}},
						Name: "p1-test-instance",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				m.EXPECT().DryRunNamespace(
					mock.Anything,
					"appl-mqp-e2e-e1",
					map[string]string{"collectord.io/index": "test-prod"},
				).Return(nil).Once()
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "p1_platform_with_wrong_suffix_invalid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "test-itd", Namespace: "appl-mqp-e2e-e1"}},
						Name: "p1-test-instance",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				// No mock setup needed
			},
			want:    createResult(false, ReasonPlatformMismatchInSplunk, "the platform in instance name must match the platform in splunk index. Please use \"prod\" as suffix for splunk."),
			wantErr: assert.NoError,
		},
		{
			name: "e1_platform_with_wrong_suffix_invalid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "test-prod"}},
						Name: "e1-test-instance",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				// No mock setup needed
			},
			want:    createResult(false, ReasonPlatformMismatchInSplunk, "the platform in instance name must match the platform in splunk index. Please use \"itd\" as suffix for splunk."),
			wantErr: assert.NoError,
		},
		{
			name: "splunk_index_does_not_exist_forbidden_error",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "nonexistent-itd", Namespace: "appl-mqp-e2e-e1"}},
						Name: "e1-test-instance",
					},
				},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				forbiddenErr := apierrors.NewForbidden(
					schema.GroupResource{Group: "", Resource: "namespaces"},
					"appl-mqp-e2e-e1",
					fmt.Errorf("admission webhook denied"),
				)
				m.EXPECT().DryRunNamespace(
					mock.Anything,
					"appl-mqp-e2e-e1",
					map[string]string{"collectord.io/index": "nonexistent-itd"},
				).Return(forbiddenErr).Once()
			},
			want: policy.Results{
				{
					Reason:  ReasonSplunkIndexMissing,
					Message: "the provided splunk index nonexistent-itd isn't valid or doesn't exist yet on this k8s cluster. Please check your [SpaaS configuration](https://gitlab.pnet.ch/observability/logging/spaas).",
					Close:   false,
					Requeue: requeueInterval,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "network_error_during_dry_run",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "test-itd", Namespace: "appl-mqp-e2e-e1"}},
						Name: "e1-test-instance",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				m.EXPECT().DryRunNamespace(
					mock.Anything,
					"appl-mqp-e2e-e1",
					map[string]string{"collectord.io/index": "test-itd"},
				).Return(fmt.Errorf("network timeout")).Once()
			},
			want: policy.Results{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.Error(t, err) && assert.Contains(t, err.Error(), "validation for splunk index failed")
			},
		},
		{
			name: "update_operation_with_valid_splunk",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{Instance: activemqv1alpha1.Instance{SplunkIndex: "test-itd", Namespace: "appl-mqp-e2e-e1"}},
						Name: "t1-test-instance",
					}},
			},
			mockSetup: func(m *k8sMocks.NamespaceClient) {
				m.EXPECT().DryRunNamespace(
					mock.Anything,
					"appl-mqp-e2e-e1",
					map[string]string{"collectord.io/index": "test-itd"},
				).Return(nil).Once()
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespaceClient := &k8sMocks.NamespaceClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(namespaceClient)
			}

			got, err := CheckSplunk(namespaceClient)(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckPlatformInSplunkIndex(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckPlatformInSplunkIndex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckQueueNaming(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger"}, {Name: "mqp.egw.In_Queue"}, {Name: "mqp.esign.Out_Queue_PDF"}, {Name: "mqp.adw.my-first-queue"}},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "valid_no_queues",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid_general_pattern",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name: "mqp.myapp.In.Queue.1"},
								},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueName, "the queue name `mqp.myapp.In.Queue.1` does not follow the general naming convention `^mqp\\.[a-z0-9-]+\\.[a-zA-Z0-9-]+(_[a-zA-Z0-9-]+)*$`, examples: mqp.esbpf.In_Queue_1_Test, mqp.myapp.my-first-mqp-demo, mqp.myapp.My_Test_Queue"),
			wantErr: assert.NoError,
		},
		{
			name: "invalid_esbpf_pattern",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name: "mqp.egw.In_Queue-PDF"},
								},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueName, "the queue name `mqp.egw.In_Queue-PDF` does not follow the ESBPF naming convention for `^mqp\\.[a-z0-9-]+\\.(In_Queue|Out_Queue)(_[a-zA-Z0-9]+)*$`, examples: mqp.esbpf.In_Queue_ECM_Trigger, mqp.myapp.Out_Queue"),
			wantErr: assert.NoError,
		},
		{
			name: "invalid_dlq_definition",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name: "mqp.egw.In_Queue_Queue_DLQ"},
								},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueName, "the queue `mqp.egw.In_Queue_Queue_DLQ` ends with DLQ. DLQs are created implicit for every queue and must not be defined"),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckQueueNaming(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckQueueNaming(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckQueueNaming() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckQueueDuplicate(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "no_duplicates",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger"}, {Name: "mqp.egw.In_Queue"}},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "no_queues",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "one_duplicate",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.myapp.my-queue"}, {Name: "mqp.myapp.my-queue"}},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonDuplicateQueue, "the queue `mqp.myapp.my-queue` is defined 2 times, each queue name must be unique"),
			wantErr: assert.NoError,
		},
		{
			name: "multiple_duplicates",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{
									{Name: "mqp.myapp.queue-a"},
									{Name: "mqp.myapp.queue-b"},
									{Name: "mqp.myapp.queue-a"},
									{Name: "mqp.myapp.queue-b"},
									{Name: "mqp.myapp.queue-b"},
								},
							},
						},
					},
				},
			},
			want: createResult(false, ReasonDuplicateQueue,
				"the queue `mqp.myapp.queue-a` is defined 2 times, each queue name must be unique",
				"the queue `mqp.myapp.queue-b` is defined 3 times, each queue name must be unique",
			),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckQueueDuplicate(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckQueueDuplicate(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckQueueDuplicate() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckQueueRename(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}

	tests := []struct {
		name    string
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "rename_detected_on_modify",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationModify,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{
									{Name: "mqp.app.orders"},
									{Name: "mqp.app.invoices"},
								},
							},
						},
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{
									{Name: "mqp.app.orders"},
									{Name: "mqp.app.invoices-v2"},
								},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonQueueRenamed, "queue rename detected. removed queue name(s): `mqp.app.invoices`, added queue name(s): `mqp.app.invoices-v2`. queue renames are not allowed. Ensure the queue is emptied first and that no producers or consumers are active, then delete it. Afterwards, open a new merge request to create the new queue."),
			wantErr: assert.NoError,
		},
		{
			name: "no_rename_when_only_reordered",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationModify,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{
									{Name: "mqp.app.orders"},
									{Name: "mqp.app.invoices"},
								},
							},
						},
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{
									{Name: "mqp.app.invoices"},
									{Name: "mqp.app.orders"},
								},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "no_rename_when_queue_added_only",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationModify,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{
									{Name: "mqp.app.orders"},
								},
							},
						},
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{
									{Name: "mqp.app.orders"},
									{Name: "mqp.app.invoices"},
								},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "no_check_on_create",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationCreate,
					},
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{
									{Name: "mqp.app.orders"},
								},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "no_check_on_delete",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					Change: merge.Change{
						Operation: merge.ChangeOperationDelete,
					},
					OldObj: &activemqv1alpha1.PFActiveMQ{
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{
									{Name: "mqp.app.orders"},
								},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckQueueRename(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckQueueRename(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckQueueRename() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckESBPFOperator(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid_operator_role_required_and_exists",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Roles: activemqv1alpha1.Roles{
								Operators: []string{"esbpf.operator"},
							},
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger"}, {Name: "mqp.egw.In_Queue"}, {Name: "mqp.esign.Out_Queue_PDF"}, {Name: "mqp.adw.my-first-queue"}},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid_operator_role_required_and_does_not_exist",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Roles: activemqv1alpha1.Roles{
								Operators: []string{"test.operator"},
							},
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger"}, {Name: "mqp.egw.In_Queue"}, {Name: "mqp.esign.Out_Queue_PDF"}, {Name: "mqp.adw.my-first-queue"}},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonMissingESBPFOperatorRole, "the queue mqp.esbpf.In_Queue_ECM_Trigger contains the ESBPF identifier, therefore the ESBPF operator role esbpf.operator must be added to Spec.Roles.Operators."),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckESBPFOperator(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckESBPFOperator(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckESBPFOperator() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckQueuePageFullMessagePolicyConfigChange(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid_no_config_defined",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger"}}},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{PageFullMessagePolicy: "FAIL"}}, {Name: "mqp.egw.In_Queue_Queue_DLQ", Config: &activemqv1alpha1.Config{PageFullMessagePolicy: "DROP"}}}},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid_page_full_message_policy",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{PageFullMessagePolicy: "NO_VALID"}}},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the queue `mqp.esbpf.In_Queue_ECM_Trigger` has an invalid pageFullMessagePolicy `NO_VALID`. Valid values are: FAIL, DROP."),
			wantErr: assert.NoError,
		},
		{
			name: "invalid_other_queues_not_defined",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{PageFullMessagePolicy: "DROP"}}, {Name: "mqp.egw.In_Queue_Queue_DLQ"}},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the queue `mqp.esbpf.In_Queue_ECM_Trigger` has pageFullMessagePolicy set to `DROP`, but the queue `mqp.egw.In_Queue_Queue_DLQ` does not have pageFullMessagePolicy set. Please set pageFullMessagePolicy for all queues or remove it from all queues."),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckQueuePageFullMessagePolicyConfigChange(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckQueuePageFullMessagePolicyConfigChange(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckQueuePageFullMessagePolicyConfigChange() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckQueueMaxSizeBytesConfigChange(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"memory": resource.MustParse("2Gi")}},
								Queues:    []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{MaxSizeBytes: 500000}}, {Name: "mqp.egw.In_Queue_Queue_DLQ", Config: &activemqv1alpha1.Config{MaxSizeBytes: 500000}}}},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid_exceed_instance_memory_limit",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"memory": resource.MustParse("2Gi")}},
								Queues:    []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{MaxSizeBytes: 2147483648}}, {Name: "mqp.egw.In_Queue_Queue_DLQ", Config: &activemqv1alpha1.Config{MaxSizeBytes: 2147483640}}},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the total maxSizeBytes of all queues `4294967288` exceeds the allowed memory for config `1288490188` (60% of instance limit). Please reduce the maxSizeBytes or increase the memory limit."),
			wantErr: assert.NoError,
		},
		{
			name: "invalid_other_queues_not_defined",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"memory": resource.MustParse("4Gi")}},
								Queues:    []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{MaxSizeBytes: 2147483648}}, {Name: "mqp.egw.In_Queue_Queue_DLQ"}},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the queue `mqp.esbpf.In_Queue_ECM_Trigger` has maxSizeBytes set to `2147483648`, but the queue `mqp.egw.In_Queue_Queue_DLQ` does not have maxSizeBytes set. Please set maxSizeBytes for all queues or remove it from all queues."),
			wantErr: assert.NoError,
		},
		{
			name: "invalid_no_positive_value",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"memory": resource.MustParse("4Gi")}},
								Queues:    []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{MaxSizeBytes: -100}}},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the queue `mqp.esbpf.In_Queue_ECM_Trigger` has an invalid maxSizeBytes `-100`. The value must be a positive integer."),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckQueueMaxSizeBytesConfigChange(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckQueueMaxSizeBytesConfigChange(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckQueueMaxSizeBytesConfigChange() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckQueuePageLimitBytesConfigChange(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Storage: "2Gi",
								Queues:  []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{PageLimitBytes: 500000}}, {Name: "mqp.egw.In_Queue_Queue_DLQ", Config: &activemqv1alpha1.Config{PageLimitBytes: 500000}}}},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid_exceed_instance_storage_limit",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Storage: "2Gi",
								Queues:  []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{PageLimitBytes: 2147483648}}, {Name: "mqp.egw.In_Queue_Queue_DLQ", Config: &activemqv1alpha1.Config{PageLimitBytes: 2147483648}}}},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the total pageLimitBytes of all queues `4294967296` exceeds the allowed storage for config `1288490188` (60% of instance limit). Please reduce the pageLimitBytes or increase storage."),
			wantErr: assert.NoError,
		},
		{
			name: "invalid_other_queues_not_defined",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Storage: "2Gi",
								Queues:  []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{PageLimitBytes: 2147483648}}, {Name: "mqp.egw.In_Queue_Queue_DLQ"}}},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the queue `mqp.esbpf.In_Queue_ECM_Trigger` has pageLimitBytes set to `2147483648`, but the queue `mqp.egw.In_Queue_Queue_DLQ` does not have pageLimitBytes set. Please set pageLimitBytes for all queues or remove it from all queues."),
			wantErr: assert.NoError,
		},
		{
			name: "invalid_no_positive_value",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "t1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Storage: "2Gi",
								Queues:  []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{PageLimitBytes: -100}}},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the queue `mqp.esbpf.In_Queue_ECM_Trigger` has an invalid pageLimitBytes `-100`. The value must be a positive integer."),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckQueuePageLimitBytesConfigChange(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckQueuePageLimitBytesConfigChange(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckQueuePageLimitBytesConfigChange() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckQueueExpiryDelayConfigChange(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{ExpiryDelay: 500000}}}},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{ExpiryDelay: -333}}}},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the queue `mqp.esbpf.In_Queue_ECM_Trigger` has an invalid expiryDelay `-333`. The value must be a positive integer."),
			wantErr: assert.NoError,
		},
		{
			name: "invalid_other_queues_not_defined",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Storage: "2Gi",
								Queues:  []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger"}, {Name: "mqp.egw.In_Queue_Queue_DLQ"}}},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "queue_without_expiry_config_exists",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Storage: "2Gi",
								Queues:  []activemqv1alpha1.Queue{{Name: "mqp.esbpf.In_Queue_ECM_Trigger", Config: &activemqv1alpha1.Config{ExpiryDelay: 500000}}, {Name: "mqp.egw.In_Queue_Queue_DLQ"}}},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidQueueConfig, "the queue `mqp.esbpf.In_Queue_ECM_Trigger` has expiryDelay set to `500000`, but the queue `mqp.egw.In_Queue_Queue_DLQ` does not have expiryDelay set. Please set expiryDelay for all queues or remove it from all queues."),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckQueueExpiryDelayConfigChange(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckQueueExpiryDelayConfigChange(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckQueueExpiryDelayConfigChange() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckAllowedPermissions(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name: "mqp.esbpf.In_Queue_ECM_Trigger",
									Permission: activemqv1alpha1.Permission{
										Send:    []string{"a-pf-identity-mqp-e2e-e1"},
										Consume: []string{"a-pf-identity-mqp-e2e-e1"},
										Browse:  []string{"a-pf-identity-mqp-e2e-e1"},
									},
								}},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Instance: activemqv1alpha1.Instance{
								Queues: []activemqv1alpha1.Queue{{
									Name: "mqp.esbpf.In_Queue_ECM_Trigger",
									Permission: activemqv1alpha1.Permission{
										CreateAddress: []string{"a-pf-identity-mqp-e2e-e1"},
										Consume:       []string{"a-pf-identity-mqp-e2e-e1"},
										Browse:        []string{"a-pf-identity-mqp-e2e-e1"},
									},
								}},
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonInvalidPermission, "the queue `mqp.esbpf.In_Queue_ECM_Trigger` has an invalid permission `createAddress`. Valid permissions are: Send, Consume, Browse."),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckAllowedPermissions(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckAllowedPermissions(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckAllowedPermissions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckOperatorRole(t *testing.T) {
	type args struct {
		change merge.ParsedChange[activemqv1alpha1.PFActiveMQ]
	}
	tests := []struct {
		name    string
		args    args
		want    policy.Results
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Roles: activemqv1alpha1.Roles{
								Operators: []string{"asm.mw_mqp_owner"},
							},
						},
					},
				},
			},
			want:    policy.Results{},
			wantErr: assert.NoError,
		},
		{
			name: "invalid_no_operator_role_defined",
			args: args{
				change: merge.ParsedChange[activemqv1alpha1.PFActiveMQ]{
					NewObj: &activemqv1alpha1.PFActiveMQ{
						Name: "e1-mqp-e2e-a",
						Spec: activemqv1alpha1.PFActiveMQSpec{
							Roles: activemqv1alpha1.Roles{
								Owner: "asm.mw_mqp_owner",
							},
						},
					},
				},
			},
			want:    createResult(false, ReasonMissingOperatorRole, "an operator role must be defined."),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckOperatorRole(context.TODO(), nil, &tt.args.change)
			if !tt.wantErr(t, err, fmt.Sprintf("CheckOperatorRole(%v, %v)", context.TODO(), tt.args.change)) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckOperatorRole() got = %v, want %v", got, tt.want)
			}
		})
	}
}
