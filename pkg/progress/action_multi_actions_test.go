package progress

import (
	"context"

	. "gopkg.in/check.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/fake"
)

type TestSuiteMultiActions struct {
	blueprint *crv1alpha1.Blueprint
	actionSet *crv1alpha1.ActionSet
	clientset *fake.Clientset
}

var _ = Suite(&TestSuiteMultiActions{})

func (s *TestSuiteMultiActions) SetUpTest(c *C) {
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
					{
						Name: "echo-hello-0-1",
						Func: "echo-hello-func",
					},
				},
			},
			"action-02": {
				Name: "action-02",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "echo-hello-1-0",
						Func: "echo-hello-func",
					},
					{
						Name: "echo-hello-1-1",
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
				{
					Name: "action-02",
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
						{
							Name:  "echo-hello-0-1",
							State: crv1alpha1.StatePending,
						},
					},
				},
				{
					Name: "action-02",
					Phases: []crv1alpha1.Phase{
						{
							Name:  "echo-hello-1-0",
							State: crv1alpha1.StatePending,
						},
						{
							Name:  "echo-hello-1-1",
							State: crv1alpha1.StatePending,
						},
					},
				},
			},
		},
	}

	s.clientset = fake.NewSimpleClientset()
	err := s.createFixtures(mockBlueprint, mockActionSet)
	c.Assert(err, IsNil)
}

func (s *TestSuiteMultiActions) TearDownTest(c *C) {
	blueprintErr := s.clientset.CrV1alpha1().Blueprints(s.blueprint.GetNamespace()).Delete(
		context.Background(),
		s.blueprint.GetName(),
		metav1.DeleteOptions{})
	c.Assert(blueprintErr, IsNil)

	actionSetErr := s.clientset.CrV1alpha1().ActionSets(s.actionSet.GetNamespace()).Delete(
		context.Background(),
		s.actionSet.GetName(),
		metav1.DeleteOptions{})
	c.Assert(actionSetErr, IsNil)
}

func (s *TestSuiteMultiActions) createFixtures(blueprint *crv1alpha1.Blueprint, actionSet *crv1alpha1.ActionSet) error {
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

func (s *TestSuiteMultiActions) TestUpdateActionsProgress(c *C) {
	// This test simulates ActionSet consisting of two actions with two phases in each
	var testCases = []struct {
		indexAction                   int
		indexPhase                    int
		phaseState                    crv1alpha1.State
		phaseProgress                 crv1alpha1.PhaseProgress
		expectedPhasePercent          string
		expectedActionPercent         string
		expectedSizeUploadedB         int64
		expectedSizeDownloadedB       int64
		expectedEstimatedUploadSizeB  int64
		expectedEstimateDownloadSizeB int64
	}{
		// The first phase of the first action is in a pending state.
		{
			indexAction:                  0,
			indexPhase:                   0,
			phaseState:                   crv1alpha1.StatePending,
			expectedPhasePercent:         "",
			expectedActionPercent:        "",
			expectedSizeUploadedB:        0,
			expectedEstimatedUploadSizeB: 0,
		},
		// The first phase of the first action is in a running state.
		{
			indexAction: 0,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateRunning,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      "40",
				SizeUploadedB:        2345,
				EstimatedUploadSizeB: 12345,
			},
			expectedPhasePercent:         "40",
			expectedActionPercent:        "10", // Total 4 phases, calculating average to set action progress
			expectedSizeUploadedB:        2345,
			expectedEstimatedUploadSizeB: 12345,
		},
		// The first phase of the first action is in a complete state.
		{
			indexAction: 0,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateComplete,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      CompletedPercent,
				SizeUploadedB:        12345,
				EstimatedUploadSizeB: 12345,
			},
			expectedPhasePercent:         CompletedPercent,
			expectedActionPercent:        "25",
			expectedSizeUploadedB:        12345,
			expectedEstimatedUploadSizeB: 12345,
		},
		// The second phase of the first action is in a pending state.
		{
			indexAction: 0,
			indexPhase:  1,
			phaseState:  crv1alpha1.StatePending,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent: "0",
			},
			expectedPhasePercent:         "",
			expectedActionPercent:        "25",
			expectedSizeUploadedB:        12345,
			expectedEstimatedUploadSizeB: 12345,
		},
		// The second phase of the first action is in a running state.
		{
			indexAction: 0,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateRunning,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:        "60",
				SizeDownloadedB:        1000,
				EstimatedDownloadSizeB: 10000,
			},
			expectedPhasePercent:          "60",
			expectedActionPercent:         "40",
			expectedSizeDownloadedB:       1000,
			expectedSizeUploadedB:         12345,
			expectedEstimateDownloadSizeB: 10000,
			expectedEstimatedUploadSizeB:  12345,
		},
		// The second phase of the first action is in a completed state.
		{
			indexAction: 0,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateComplete,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:        CompletedPercent,
				SizeDownloadedB:        10000,
				EstimatedDownloadSizeB: 10000,
			},
			expectedPhasePercent:          CompletedPercent,
			expectedActionPercent:         "50",
			expectedSizeDownloadedB:       10000,
			expectedSizeUploadedB:         12345,
			expectedEstimateDownloadSizeB: 10000,
			expectedEstimatedUploadSizeB:  12345,
		},
		// The first phase of the second action is in a completed state.
		{
			indexAction:                   1,
			indexPhase:                    0,
			phaseState:                    crv1alpha1.StatePending,
			expectedPhasePercent:          "",
			expectedActionPercent:         "50",
			expectedSizeDownloadedB:       10000,
			expectedSizeUploadedB:         12345,
			expectedEstimateDownloadSizeB: 10000,
			expectedEstimatedUploadSizeB:  12345,
		},
		// The first phase of the second action is in a running state.
		{
			indexAction: 1,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateRunning,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      "37",
				SizeUploadedB:        1000,
				EstimatedUploadSizeB: 10000,
			},
			expectedPhasePercent:          "37",
			expectedActionPercent:         "59",
			expectedSizeDownloadedB:       10000,
			expectedSizeUploadedB:         13345,
			expectedEstimateDownloadSizeB: 10000,
			expectedEstimatedUploadSizeB:  22345,
		},
		// The first phase of the second action is in a complete state.
		{
			indexAction: 1,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateComplete,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      CompletedPercent,
				SizeUploadedB:        10000,
				EstimatedUploadSizeB: 10000,
			},
			expectedPhasePercent:          CompletedPercent,
			expectedActionPercent:         "75",
			expectedSizeDownloadedB:       10000,
			expectedSizeUploadedB:         22345,
			expectedEstimateDownloadSizeB: 10000,
			expectedEstimatedUploadSizeB:  22345,
		},
		// The second phase of the second action is in a pending state.
		{
			indexAction: 1,
			indexPhase:  1,
			phaseState:  crv1alpha1.StatePending,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent: "0",
			},
			expectedPhasePercent:          "",
			expectedActionPercent:         "75",
			expectedSizeDownloadedB:       10000,
			expectedSizeUploadedB:         22345,
			expectedEstimateDownloadSizeB: 10000,
			expectedEstimatedUploadSizeB:  22345,
		},
		// The second phase of the second action is in a running state.
		{
			indexAction: 1,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateRunning,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:        "23",
				SizeDownloadedB:        1000,
				EstimatedDownloadSizeB: 10000,
			},
			expectedPhasePercent:          "23",
			expectedActionPercent:         "80",
			expectedSizeDownloadedB:       11000,
			expectedSizeUploadedB:         22345,
			expectedEstimateDownloadSizeB: 20000,
			expectedEstimatedUploadSizeB:  22345,
		},
		// The second phase of the second action is in a complete state.
		{
			indexAction: 1,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateComplete,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:        CompletedPercent,
				SizeDownloadedB:        10000,
				EstimatedDownloadSizeB: 10000,
			},
			expectedPhasePercent:          CompletedPercent,
			expectedActionPercent:         CompletedPercent,
			expectedSizeDownloadedB:       20000,
			expectedSizeUploadedB:         22345,
			expectedEstimateDownloadSizeB: 20000,
			expectedEstimatedUploadSizeB:  22345,
		},
	}

	for id, tc := range testCases {
		// Get latest rev of actionset resource
		as, err := s.clientset.CrV1alpha1().ActionSets(s.actionSet.GetNamespace()).Get(context.Background(), s.actionSet.GetName(), metav1.GetOptions{})
		c.Assert(err, IsNil)
		assertActionProgress(
			c,
			s.clientset,
			as,
			tc.indexAction,
			tc.indexPhase,
			tc.phaseState,
			tc.phaseProgress,
			tc.expectedPhasePercent,
			tc.expectedActionPercent,
			tc.expectedSizeDownloadedB,
			tc.expectedSizeUploadedB,
			tc.expectedEstimateDownloadSizeB,
			tc.expectedEstimatedUploadSizeB,
			id)
	}
}

func (s *TestSuiteMultiActions) TestUpdateActionsProgressWithFailures(c *C) {
	var testCases = []struct {
		indexAction                    int
		indexPhase                     int
		phaseState                     crv1alpha1.State
		phaseProgress                  crv1alpha1.PhaseProgress
		expectedPhasePercent           string
		expectedActionPercent          string
		expectedDownloadedB            int64
		expectedSizeUploadB            int64
		expectedEstimatedUploadSizeB   int64
		expectedEstimatedDownloadSizeB int64
	}{
		{
			indexAction: 0,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateComplete,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      CompletedPercent,
				SizeUploadedB:        10000,
				EstimatedUploadSizeB: 10000,
			},
			expectedPhasePercent:         CompletedPercent,
			expectedActionPercent:        "25",
			expectedSizeUploadB:          10000,
			expectedEstimatedUploadSizeB: 10000,
		},
		{
			indexAction: 0,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateComplete,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:      CompletedPercent,
				SizeUploadedB:        1000,
				EstimatedUploadSizeB: 1000,
			},
			expectedPhasePercent:         CompletedPercent,
			expectedActionPercent:        "50",
			expectedSizeUploadB:          11000,
			expectedEstimatedUploadSizeB: 11000,
		},
		{
			indexAction: 1,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateComplete,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:        CompletedPercent,
				SizeDownloadedB:        2000,
				EstimatedDownloadSizeB: 2000,
			},
			expectedPhasePercent:           CompletedPercent,
			expectedActionPercent:          "75",
			expectedDownloadedB:            2000,
			expectedSizeUploadB:            11000,
			expectedEstimatedDownloadSizeB: 2000,
			expectedEstimatedUploadSizeB:   11000,
		},
		{
			indexAction: 1,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateFailed,
			phaseProgress: crv1alpha1.PhaseProgress{
				ProgressPercent:        "30",
				SizeDownloadedB:        3000,
				EstimatedDownloadSizeB: 9000,
			},
			expectedPhasePercent:           "",
			expectedActionPercent:          "75",
			expectedDownloadedB:            2000,
			expectedSizeUploadB:            11000,
			expectedEstimatedDownloadSizeB: 2000,
			expectedEstimatedUploadSizeB:   11000,
		},
	}

	for id, tc := range testCases {
		// Get latest rev of actionset resource
		as, err := s.clientset.CrV1alpha1().ActionSets(s.actionSet.GetNamespace()).Get(context.Background(), s.actionSet.GetName(), metav1.GetOptions{})
		c.Assert(err, IsNil)
		assertActionProgress(
			c,
			s.clientset,
			as,
			tc.indexAction,
			tc.indexPhase,
			tc.phaseState,
			tc.phaseProgress,
			tc.expectedPhasePercent,
			tc.expectedActionPercent,
			tc.expectedDownloadedB,
			tc.expectedSizeUploadB,
			tc.expectedEstimatedDownloadSizeB,
			tc.expectedEstimatedUploadSizeB,
			id)
	}
}
