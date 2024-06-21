package progress

import (
	"context"
	"strconv"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/reconcile"
	"github.com/kanisterio/kanister/pkg/validate"
)

const (
	pollDuration = time.Second * 5

	StartedPercent   = "0"
	CompletedPercent = "100"
)

// UpdateActionSetsProgress tries to assess the progress of an actionSet by
// watching the states of all the phases in its actions. It starts an infinite
// loop, using a ticker to determine when to assess the phases. The function
// returns when the provided context is either done or cancelled.
// Caller should invoke this function in a non-main goroutine, to avoid
// introducing any latencies on Kanister critical path.
// If an error happens while attempting to update the actionSet, the failed
// iteration will be skipped with no further retries, until the next tick.
func UpdateActionSetsProgress(
	ctx context.Context,
	aIDX int,
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
			retry, err := updateActionProgress(ctx, aIDX, client, actionSetName, namespace, p)
			if err != nil {
				fields := field.M{
					"actionSet":      actionSetName,
					"nextUpdateTime": time.Now().Add(pollDuration),
				}
				log.Error().WithError(err).Print("failed to update action progress", fields)
				return err
			}
			if !retry {
				return nil
			}
		}
	}
}

func updateActionProgress(
	ctx context.Context,
	aIDX int,
	client versioned.Interface,
	actionSetName string,
	namespace string,
	p *kanister.Phase,
) (bool, error) {
	actionSet, err := client.CrV1alpha1().ActionSets(namespace).Get(ctx, actionSetName, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrap(err, "Failed to get actionset")
	}

	if actionSet.Status == nil {
		return true, nil
	}

	if completedOrFailed(aIDX, actionSet, p.Name()) {
		return false, nil
	}

	if err := updateActionPhaseProgress(ctx, client, actionSet, p); err != nil {
		return true, err
	}
	return true, nil
}

func updateActionPhaseProgress(
	ctx context.Context,
	client versioned.Interface,
	actionSet *crv1alpha1.ActionSet,
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
	return updateActionSetStatus(ctx, client, actionSet, p.Name(), phaseProgress)
}

func updateActionSetStatus(
	ctx context.Context,
	client versioned.Interface,
	actionSet *crv1alpha1.ActionSet,
	phaseName string,
	phaseProgress crv1alpha1.PhaseProgress,
) error {
	fields := field.M{
		"actionSet": actionSet.GetName(),
		"namespace": actionSet.GetNamespace(),
		"phase":     phaseName,
		"progress":  phaseProgress.ProgressPercent,
	}
	log.Debug().Print("Updating phase progress", fields)
	updateFunc := func(actionSet *crv1alpha1.ActionSet) error {
		return setActionSetPhaseProgress(actionSet, phaseName, phaseProgress)
	}
	if err := reconcile.ActionSet(ctx, client.CrV1alpha1(), actionSet.GetNamespace(), actionSet.GetName(), updateFunc); err != nil {
		return err
	}
	return nil
}

func completedOrFailed(aIDX int, actionSet *crv1alpha1.ActionSet, phaseName string) bool {
	for _, phase := range actionSet.Status.Actions[aIDX].Phases {
		if phase.Name != phaseName {
			continue
		}
		return phase.State == crv1alpha1.StateFailed ||
			phase.State == crv1alpha1.StateComplete
	}
	return false
}

func arePhaseProgressesDifferent(a, b crv1alpha1.PhaseProgress) bool {
	if a.ProgressPercent != b.ProgressPercent {
		return true
	}

	if a.SizeDownloadedB != b.SizeDownloadedB || a.SizeUploadedB != b.SizeUploadedB {
		return true
	}

	if a.EstimatedDownloadSizeB != b.EstimatedDownloadSizeB || a.EstimatedUploadSizeB != b.EstimatedUploadSizeB {
		return true
	}

	if a.EstimatedTimeSeconds != b.EstimatedTimeSeconds {
		return true
	}

	return false
}

func setActionSetPhaseProgress(actionSet *crv1alpha1.ActionSet, phaseName string, phaseProgress crv1alpha1.PhaseProgress) error {
	// Update or create phase status in ActionSet status
	// Update phase progress if there is a change
	for i := 0; i < len(actionSet.Status.Actions); i++ {
		for j := 0; j < len(actionSet.Status.Actions[i].Phases); j++ {
			if actionSet.Status.Actions[i].Phases[j].Name != phaseName {
				continue
			}
			if actionSet.Status.Actions[i].Phases[j].State == crv1alpha1.StatePending ||
				actionSet.Status.Actions[i].Phases[j].State == crv1alpha1.StateFailed {
				continue
			}
			if arePhaseProgressesDifferent(actionSet.Status.Actions[i].Phases[j].Progress, phaseProgress) {
				actionSet.Status.Actions[i].Phases[j].Progress = phaseProgress
				if err := SetActionSetPercentCompleted(actionSet); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// SetActionSetPercentCompleted calculate and set percent completion of the action. The action completion percentage
// is calculated by taking an average of all the involved phases.
func SetActionSetPercentCompleted(actionSet *crv1alpha1.ActionSet) error {
	var actionProgress, totalPhases int
	var sizeUploadedB, sizeDownloadedB, estimatedUploadSizeB, estimatedDownloadSizeB int64
	for _, actions := range actionSet.Status.Actions {
		for _, phase := range actions.Phases {
			sizeUploadedB += phase.Progress.SizeUploadedB
			sizeDownloadedB += phase.Progress.SizeDownloadedB
			estimatedUploadSizeB += phase.Progress.EstimatedUploadSizeB
			estimatedDownloadSizeB += phase.Progress.EstimatedDownloadSizeB
			totalPhases++

			if phase.Progress.ProgressPercent == "" {
				continue
			}

			pp, err := strconv.Atoi(phase.Progress.ProgressPercent)
			if err != nil {
				return errors.Wrap(err, "Invalid phase progress percent")
			}
			actionProgress += pp
		}
	}
	actionProgress /= totalPhases

	// Update LastTransitionTime only if there is a change in action progress
	if strconv.Itoa(actionProgress) == actionSet.Status.Progress.PercentCompleted &&
		actionSet.Status.Progress.SizeDownloadedB == sizeDownloadedB &&
		actionSet.Status.Progress.SizeUploadedB == sizeUploadedB &&
		actionSet.Status.Progress.EstimatedDownloadSizeB == estimatedDownloadSizeB &&
		actionSet.Status.Progress.EstimatedUploadSizeB == estimatedUploadSizeB {
		return nil
	}
	metav1Time := metav1.NewTime(time.Now())
	actionSet.Status.Progress.LastTransitionTime = &metav1Time
	actionSet.Status.Progress.PercentCompleted = strconv.Itoa(actionProgress)
	actionSet.Status.Progress.SizeDownloadedB = sizeDownloadedB
	actionSet.Status.Progress.SizeUploadedB = sizeUploadedB
	actionSet.Status.Progress.EstimatedDownloadSizeB = estimatedDownloadSizeB
	actionSet.Status.Progress.EstimatedUploadSizeB = estimatedUploadSizeB
	return nil
}
