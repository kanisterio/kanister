package validate

import (
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

func ActionSet(as *crv1alpha1.ActionSet) error {
	if err := actionSetSpec(as.Spec); err != nil {
		return err
	}
	if ns := as.GetNamespace(); ns != "" {
		for _, a := range as.Spec.Actions {
			for _, v := range a.ConfigMaps {
				if v.Namespace != ns {
					return errorf("Referenced ConfigMaps must be in the same namespace as the controller")
				}
			}
			for _, v := range a.Secrets {
				if v.Namespace != ns {
					return errorf("Referenced Secrets must be in the same namespace as the controller")
				}
			}
		}
	}
	if as.Status != nil {
		if len(as.Spec.Actions) != len(as.Status.Actions) {
			return errorf("Number of actions in status actions and spec must match")
		}
		if err := actionSetStatus(as.Status); err != nil {
			return err
		}
	}
	return nil
}

func actionSetSpec(as *crv1alpha1.ActionSetSpec) error {
	if as == nil {
		return errorf("Spec must be non-nil")
	}
	return nil
}

func actionSetStatus(as *crv1alpha1.ActionSetStatus) error {
	if as == nil {
		return nil
	}
	if err := actionSetStatusActions(as.Actions); err != nil {
		return err
	}
	saw := map[crv1alpha1.State]bool{
		crv1alpha1.StatePending:  false,
		crv1alpha1.StateRunning:  false,
		crv1alpha1.StateFailed:   false,
		crv1alpha1.StateComplete: false,
	}
	for _, a := range as.Actions {
		for _, p := range a.Phases {
			if _, ok := saw[p.State]; !ok {
				return errorf("Action has unknown state '%s'", p.State)
			}
			for s := range saw {
				saw[s] = saw[s] || (p.State == s)
			}
		}
	}
	if _, ok := saw[as.State]; !ok {
		return errorf("ActionSet has unknown state '%s'", as.State)
	}
	if saw[crv1alpha1.StateRunning] || saw[crv1alpha1.StatePending] {
		if as.State == crv1alpha1.StateComplete {
			return errorf("ActionSet cannot be complete if any actions are not complete")
		}
	}
	if saw[crv1alpha1.StateFailed] != (as.State == crv1alpha1.StateFailed) {
		return errorf("Iff any action is failed, the whole ActionSet must be failed")
	}
	return nil
}

func actionSetStatusActions(as []crv1alpha1.ActionStatus) error {
	for _, a := range as {
		var sawNotComplete bool
		var lastNonComplete crv1alpha1.State
		for _, p := range a.Phases {
			if sawNotComplete && p.State != crv1alpha1.StatePending {
				return errorf("Phases after a %s one must be pending", lastNonComplete)
			}
			if !sawNotComplete {
				lastNonComplete = p.State
			}
			sawNotComplete = p.State != crv1alpha1.StateComplete
		}
	}
	return nil
}

func Blueprint(bp *crv1alpha1.Blueprint) error {
	// TODO: Add blueprint validation.
	return nil
}
