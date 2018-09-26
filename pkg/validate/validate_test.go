package validate

import (
	"github.com/kanisterio/kanister/pkg/param"
	"testing"

	. "gopkg.in/check.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ValidateSuite struct{}

var _ = Suite(&ValidateSuite{})

func (s *ValidateSuite) TestActionSet(c *C) {
	for _, tc := range []struct {
		as      *crv1alpha1.ActionSet
		checker Checker
	}{
		{
			as:      &crv1alpha1.ActionSet{},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec:       &crv1alpha1.ActionSetSpec{},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
							ConfigMaps: map[string]crv1alpha1.ObjectReference{
								"testCM": crv1alpha1.ObjectReference{
									Namespace: "ns2",
								},
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
							ConfigMaps: map[string]crv1alpha1.ObjectReference{
								"testCM": crv1alpha1.ObjectReference{
									Namespace: "ns1",
								},
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
							Secrets: map[string]crv1alpha1.ObjectReference{
								"testSecrets": crv1alpha1.ObjectReference{
									Namespace: "ns2",
								},
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
							Secrets: map[string]crv1alpha1.ObjectReference{
								"testSecrets": crv1alpha1.ObjectReference{
									Namespace: "ns1",
								},
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec:   &crv1alpha1.ActionSetSpec{},
				Status: &crv1alpha1.ActionSetStatus{},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{},
				Status: &crv1alpha1.ActionSetStatus{
					State: crv1alpha1.StatePending,
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
						},
					},
				},
				Status: &crv1alpha1.ActionSetStatus{
					State: crv1alpha1.StatePending,
				},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
						},
					},
				},
				Status: &crv1alpha1.ActionSetStatus{
					State: crv1alpha1.StatePending,
					Actions: []crv1alpha1.ActionStatus{
						crv1alpha1.ActionStatus{},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
						},
					},
				},
				Status: &crv1alpha1.ActionSetStatus{
					State: crv1alpha1.StatePending,
					Actions: []crv1alpha1.ActionStatus{
						crv1alpha1.ActionStatus{},
					},
				},
			},
			checker: IsNil,
		},
		// NamespaceKind
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: param.NamespaceKind,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		// StatefulSetKind
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: param.StatefulSetKind,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		// DeploymentKind
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: param.DeploymentKind,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		// PVCKind
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: param.PVCKind,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		// Generic K8s resource (apiversion, resource missing)
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: "unknown",
							},
						},
					},
				},
			},
			checker: NotNil,
		},
		// Generic K8s resource
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{
							Object: crv1alpha1.ObjectReference{
								Name:       "foo",
								APIVersion: "v1",
								Resource:   "serviceaccount",
							},
						},
					},
				},
			},
			checker: IsNil,
		}, // No object specified
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						crv1alpha1.ActionSpec{},
					},
				},
			},
			checker: NotNil,
		},
	} {
		err := ActionSet(tc.as)
		c.Check(err, tc.checker)
	}
}

func (s *ValidateSuite) TestActionSetStatus(c *C) {
	for _, tc := range []struct {
		as      *crv1alpha1.ActionSetStatus
		checker Checker
	}{
		{
			as:      nil,
			checker: IsNil,
		},
		{
			as:      &crv1alpha1.ActionSetStatus{},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					crv1alpha1.ActionStatus{},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					crv1alpha1.ActionStatus{},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					crv1alpha1.ActionStatus{
						Phases: []crv1alpha1.Phase{},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					crv1alpha1.ActionStatus{
						Phases: []crv1alpha1.Phase{
							crv1alpha1.Phase{},
						},
					},
				},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					crv1alpha1.ActionStatus{
						Phases: []crv1alpha1.Phase{
							crv1alpha1.Phase{},
						},
					},
				},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					crv1alpha1.ActionStatus{
						Phases: []crv1alpha1.Phase{
							crv1alpha1.Phase{
								State: crv1alpha1.StatePending,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StateFailed,
				Actions: []crv1alpha1.ActionStatus{
					crv1alpha1.ActionStatus{
						Phases: []crv1alpha1.Phase{
							crv1alpha1.Phase{
								State: crv1alpha1.StatePending,
							},
						},
					},
				},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StateComplete,
				Actions: []crv1alpha1.ActionStatus{
					crv1alpha1.ActionStatus{
						Phases: []crv1alpha1.Phase{
							crv1alpha1.Phase{
								State: crv1alpha1.StatePending,
							},
						},
					},
				},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StateComplete,
				Actions: []crv1alpha1.ActionStatus{
					crv1alpha1.ActionStatus{
						Phases: []crv1alpha1.Phase{
							crv1alpha1.Phase{
								State: crv1alpha1.StatePending,
							},
							crv1alpha1.Phase{
								State: crv1alpha1.StateComplete,
							},
						},
					},
				},
			},
			checker: NotNil,
		},
	} {
		err := actionSetStatus(tc.as)
		c.Check(err, tc.checker)
	}
}

func (s *ValidateSuite) TestBlueprint(c *C) {
	err := Blueprint(nil)
	c.Assert(err, IsNil)
}

func (s *ValidateSuite) TestCloudObjectProvider(c *C) {
	cases := []struct {
		cop     crv1alpha1.CloudObjectProvider
		checker Checker
	}{
		{crv1alpha1.CloudObjectProviderGCS, IsNil},
		{crv1alpha1.CloudObjectProviderS3, IsNil},
		{crv1alpha1.CloudObjectProvider(""), NotNil},
		{crv1alpha1.CloudObjectProvider("unsupported provider"), NotNil},
	}
	for _, tc := range cases {
		err := CloudObjectProvider(tc.cop)
		c.Assert(err, tc.checker)
	}
}
