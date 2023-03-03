package progress

import (
	"context"
	"fmt"
	"strconv"
	"time"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/field"
	fn "github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/reconcile"
	"github.com/kanisterio/kanister/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	progressPercentCompleted  = 100.00
	progressPercentStarted    = 10.00
	progressPercentNotStarted = 0
	weightNormal              = 1.0
	weightHeavy               = 2.0
	pollDuration              = time.Second * 2
)

var longRunningFuncs = []string{
	fn.BackupDataFuncName,
	fn.BackupDataAllFuncName,
	fn.RestoreDataFuncName,
	fn.RestoreDataAllFuncName,
	fn.CopyVolumeDataFuncName,
	fn.CreateRDSSnapshotFuncName,
	fn.ExportRDSSnapshotToLocFuncName,
	fn.RestoreRDSSnapshotFuncName,
}

// TrackActionsProgress tries to assess the progress of an actionSet by
// watching the states of all the phases in its actions. It starts an infinite
// loop, using a ticker to determine when to assess the phases. The function
// returns when the provided context is either done or cancelled.
// Caller should invoke this function in a non-main goroutine, to avoid
// introducing any latencies on Kanister critical path.
// If an error happens while attempting to update the actionSet, the failed
// iteration will be skipped with no further retries, until the next tick.
func TrackActionsProgress(
	ctx context.Context,
	client versioned.Interface,
	actionSetName string,
	namespace string,
	p *kanister.Phase,
) error {
	ticker := time.NewTicker(pollDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			actionSet, err := client.CrV1alpha1().ActionSets(namespace).Get(ctx, actionSetName, metav1.GetOptions{})
			if err != nil {
				log.Error().WithError(err).Print("Failed to get actionset")
				return err
			}

			if actionSet.Status == nil {
				continue
			}

			if err := updateActionsProgress(ctx, client, actionSet, time.Now(), p); err != nil {
				fields := field.M{
					"actionSet":      actionSet.Name,
					"nextUpdateTime": time.Now().Add(pollDuration),
				}
				log.Error().WithError(err).Print("failed to update phase progress", fields)
				continue
			}

			if completedOrFailed(actionSet) {
				return nil
			}
		}
	}
}

func calculatePhaseWeights(
	ctx context.Context,
	actionSetName string,
	namespace string,
	client versioned.Interface,
) (map[string]float64, float64, error) {
	var (
		phaseWeights = map[string]float64{}
		totalWeight  = 0.0
	)

	actionSet, err := client.CrV1alpha1().ActionSets(namespace).Get(ctx, actionSetName, metav1.GetOptions{})
	if err != nil {
		return nil, 0.0, err
	}

	for _, action := range actionSet.Spec.Actions {
		blueprintName := action.Blueprint
		blueprint, err := client.CrV1alpha1().Blueprints(actionSet.GetNamespace()).Get(ctx, blueprintName, metav1.GetOptions{})
		if err != nil {
			return nil, 0.0, err
		}

		if err := validate.Blueprint(blueprint); err != nil {
			return nil, 0.0, err
		}

		blueprintAction, exists := blueprint.Actions[action.Name]
		if !exists {
			return nil, 0.0, fmt.Errorf("missing blueprint action: %s", action.Name)
		}

		for _, phase := range blueprintAction.Phases {
			phaseWeight := weight(&phase)
			phaseWeights[phase.Name] = phaseWeight
			totalWeight += phaseWeight
		}
	}

	return phaseWeights, totalWeight, nil
}

func updateActionsProgress(
	ctx context.Context,
	client versioned.Interface,
	actionSet *crv1alpha1.ActionSet,
	now time.Time,
	p *kanister.Phase,
) error {
	if err := validate.ActionSet(actionSet); err != nil {
		return err
	}

	phaseProgress, err := p.Progress()
	if err != nil {
		log.Error().WithError(err).Print("Failed to get progress")
		return err
	}
	return updateActionSet(ctx, client, actionSet, p.Name(), phaseProgress, now)
}

func weight(phase *crv1alpha1.BlueprintPhase) float64 {
	if longRunning(phase) {
		return weightHeavy
	}
	return weightNormal
}

func longRunning(phase *crv1alpha1.BlueprintPhase) bool {
	for _, f := range longRunningFuncs {
		if phase.Func == f {
			return true
		}
	}

	return false
}

func updateActionSet(
	ctx context.Context,
	client versioned.Interface,
	actionSet *crv1alpha1.ActionSet,
	phaseName string,
	phaseProgress crv1alpha1.PhaseProgress,
	lastTransitionTime time.Time,
) error {

	fields := field.M{
		"actionSet": actionSet.GetName(),
		"namespace": actionSet.GetNamespace(),
		"phase":     phaseName,
		"progress":  phaseProgress.ProgressPercent,
	}
	log.Info().Print("updating phase progress", fields)

	updateFunc := func(actionSet *crv1alpha1.ActionSet) error {
		// Update or create phase status in ActionSet status
		actionProgress := 0
		for i := 0; i < len(actionSet.Status.Actions[0].Phases); i++ {
			if actionSet.Status.Actions[0].Phases[i].Name == phaseName {
				actionSet.Status.Actions[0].Phases[i].State = crv1alpha1.StateRunning
				actionSet.Status.Actions[0].Phases[i].Progress.ProgressPercent = phaseProgress.ProgressPercent
			}
			actionProgress += (actionSet.Status.Actions[0].Phases[i].Progress.ProgressPercent / len(actionSet.Status.Actions[0].Phases))
		}
		metav1Time := metav1.NewTime(lastTransitionTime)

		actionSet.Status.Progress.PercentCompleted = strconv.Itoa(actionProgress)
		actionSet.Status.Progress.LastTransitionTime = &metav1Time
		return nil
	}

	if err := reconcile.ActionSet(ctx, client.CrV1alpha1(), actionSet.GetNamespace(), actionSet.GetName(), updateFunc); err != nil {
		return err
	}

	return nil
}

func completedOrFailed(actionSet *crv1alpha1.ActionSet) bool {
	return actionSet.Status.State == crv1alpha1.StateFailed ||
		actionSet.Status.State == crv1alpha1.StateComplete
}
