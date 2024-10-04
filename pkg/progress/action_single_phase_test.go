package progress

import (
	"context"
	"fmt"
	"testing"

	"gopkg.in/check.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/fake"
)

const (
	testBlueprint = "test-blueprint-progress"
	testActionset = "test-actionset-progress"
	testNamespace = "default"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type TestSuiteSinglePhase struct {
	blueprint *crv1alpha1.Blueprint
	actionSet *crv1alpha1.ActionSet
	clientset *fake.Clientset
}

var _ = check.Suite(&TestSuiteSinglePhase{})

func (s *TestSuiteSinglePhase) SetUpTest(c *check.C) {
	mockBlueprint := &crv1alpha1.Blueprint{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testBlueprint,
			Namespace: testNamespace,
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"action-01": {
				Name: "action-01",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "echo-hello-0-0",
						Func: "echo-hello-func",
					},
				},
			},
		},
	}

	mockActionSet := &crv1alpha1.ActionSet{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testActionset,
			Namespace: testNamespace,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Name: "action-01",
					Object: crv1alpha1.ObjectReference{
						APIVersion: "v1",
						Group:      "",
						Resource:   "Namespace",
					},
					Blueprint: testBlueprint,
				},
			},
		},
		Status: &crv1alpha1.ActionSetStatus{
			State: crv1alpha1.StateRunning,
			Actions: []crv1alpha1.ActionStatus{
				{
					Name: "action-01",
					Phases: []crv1alpha1.Phase{
						{
							Name:  "echo-hello-0-0",
							State: crv1alpha1.StatePending,
						},
					},
				},
			},
		},
	}

	s.clientset = fake.NewSimpleClientset()
	err := s.createFixtures(mockBlueprint, mockActionSet)
	c.Assert(err, check.IsNil)
}

func (s *TestSuiteSinglePhase) TearDownTest(c *check.C) {
	blueprintErr := s.clientset.CrV1alpha1().Blueprints(s.blueprint.GetNamespace()).Delete(
		context.Background(),
		s.blueprint.GetName(),
		metav1.DeleteOptions{})
	c.Assert(blueprintErr, check.IsNil)

	actionSetErr := s.clientset.CrV1alpha1().ActionSets(s.actionSet.GetNamespace()).Delete(
		context.Background(),
		s.actionSet.GetName(),
		metav1.DeleteOptions{})
	c.Assert(actionSetErr, check.IsNil)
}

func (s *TestSuiteSinglePhase) createFixtures(blueprint *crv1alpha1.Blueprint, actionSet *crv1alpha1.ActionSet) error {
	createdBlueprint, err := s.clientset.CrV1alpha1().Blueprints(blueprint.GetNamespace()).Create(
		context.Background(),
		blueprint,
		metav1.CreateOptions{})
	if err != nil {
		return err
	}
	s.blueprint = createdBlueprint

	createdActionSet, err := s.clientset.CrV1alpha1().ActionSets(actionSet.GetNamespace()).Create(
		context.Background(),
		actionSet,
		metav1.CreateOptions{})
	if err != nil {
		return err
	}
	s.actionSet = createdActionSet

	return nil
}

func (s *TestSuiteSinglePhase) TestUpdateActionPhaseProgress(c *check.C) {
	var testCases = []struct {
		indexAction                  int
		indexPhase                   int
		phaseState                   crv1alpha1.State
		phaseProgress                crv1alpha1.PhaseProgress
		expectedPhasePercent         string
		expectedActionPercent        string
		expectedSizeUploadB          int64
		expectedEstimatedUploadSizeB int64
	}{
		{
			phaseState:            crv1alpha1.StatePending,
			expectedPhasePercent:  "",
			expectedActionPercent: "",
		},
		{
			phaseState: crv1alpha1.StateRunning,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      "20",
				SizeUploadedB:        2000,
				EstimatedUploadSizeB: 10000,
			},
			expectedPhasePercent:         "20",
			expectedActionPercent:        "20",
			expectedSizeUploadB:          2000,
			expectedEstimatedUploadSizeB: 10000,
		},
		{
			phaseState: crv1alpha1.StateRunning,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      "50",
				SizeUploadedB:        5000,
				EstimatedUploadSizeB: 10000,
			},
			expectedPhasePercent:         "50",
			expectedActionPercent:        "50",
			expectedSizeUploadB:          5000,
			expectedEstimatedUploadSizeB: 10000,
		},
		{
			phaseState: crv1alpha1.StateFailed,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      "50",
				SizeUploadedB:        5000,
				EstimatedUploadSizeB: 10000,
			},
			expectedPhasePercent:         "",
			expectedActionPercent:        "",
			expectedSizeUploadB:          0,
			expectedEstimatedUploadSizeB: 0,
		},
		{
			phaseState: crv1alpha1.StateComplete,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      CompletedPercent,
				SizeUploadedB:        10000,
				EstimatedUploadSizeB: 10000,
			},
			expectedPhasePercent:         CompletedPercent,
			expectedActionPercent:        CompletedPercent,
			expectedSizeUploadB:          10000,
			expectedEstimatedUploadSizeB: 10000,
		},
	}
	for id, tc := range testCases {
		assertActionProgress(
			c,
			s.clientset,
			s.actionSet,
			tc.indexAction,
			tc.indexPhase,
			tc.phaseState,
			tc.phaseProgress,
			tc.expectedPhasePercent,
			tc.expectedActionPercent,
			0, // Since the phase is upload only, download size expected to remain 0.
			tc.expectedSizeUploadB,
			0, // Since the phase is upload only, download size expected to remain 0.
			tc.expectedEstimatedUploadSizeB,
			id)
	}
}

func assertActionProgress(
	c *check.C,
	clientset versioned.Interface,
	actionSet *crv1alpha1.ActionSet,
	indexAction int,
	indexPhase int,
	phaseState crv1alpha1.State,
	phaseProgress crv1alpha1.PhaseProgress,
	expectedPhasePercent string,
	expectedActionPercent string,
	expectedSizeDownloadedB int64,
	expectedSizeUploadedB int64,
	expectedEstimatedDownloadSizeB int64,
	expectedEstimatedUploadSizeB int64,
	testCaseID int,
) {
	now := metav1.Now()
	actionSet.Status.Actions[indexAction].Phases[indexPhase].State = phaseState
	updated, err := clientset.CrV1alpha1().ActionSets(actionSet.GetNamespace()).Update(context.Background(), actionSet, metav1.UpdateOptions{})
	c.Assert(err, check.IsNil)
	phaseName := fmt.Sprintf("echo-hello-%d-%d", indexAction, indexPhase)
	phaseProgress.LastTransitionTime = &now
	err1 := updateActionSetStatus(context.Background(), clientset, actionSet, phaseName, phaseProgress)
	c.Assert(err1, check.IsNil, check.Commentf("test case #: %d", testCaseID))
	actual, err := clientset.CrV1alpha1().ActionSets(actionSet.GetNamespace()).Get(context.Background(), updated.GetName(), metav1.GetOptions{})
	c.Assert(err, check.IsNil)
	// Check phase progress percent
	c.Assert(actual.Status.Actions[indexAction].Phases[indexPhase].Progress.ProgressPercent, check.Equals, expectedPhasePercent, check.Commentf("test case #: %d", testCaseID))
	// Check action progress percent
	c.Assert(actual.Status.Progress.PercentCompleted, check.Equals, expectedActionPercent, check.Commentf("test case #: %d", testCaseID))
	c.Assert(actual.Status.Progress.SizeDownloadedB, check.Equals, expectedSizeDownloadedB, check.Commentf("test case #: %d", testCaseID))
	c.Assert(actual.Status.Progress.SizeUploadedB, check.Equals, expectedSizeUploadedB, check.Commentf("test case #: %d", testCaseID))
	c.Assert(actual.Status.Progress.EstimatedDownloadSizeB, check.Equals, expectedEstimatedDownloadSizeB, check.Commentf("test case #: %d", testCaseID))
	c.Assert(actual.Status.Progress.EstimatedUploadSizeB, check.Equals, expectedEstimatedUploadSizeB, check.Commentf("test case #: %d", testCaseID))
	if phaseState != crv1alpha1.StateFailed &&
		phaseState != crv1alpha1.StatePending {
		c.Assert(actual.Status.Actions[indexAction].Phases[indexPhase].Progress.LastTransitionTime, check.NotNil)
		c.Assert(*actual.Status.Actions[indexAction].Phases[indexPhase].Progress.LastTransitionTime, check.Equals, now, check.Commentf("test case #: %d", testCaseID))
	}
}
