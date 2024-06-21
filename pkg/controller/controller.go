/*

Copyright 2016 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Some of the code below came from https://github.com/coreos/etcd-operator
which also has the apache 2.0 license.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/tomb.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/scheme"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/customresource"
	"github.com/kanisterio/kanister/pkg/eventer"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/reconcile"
	"github.com/kanisterio/kanister/pkg/validate"

	_ "github.com/kanisterio/kanister/pkg/metrics"
)

// Controller represents a controller object for kanister custom resources
type Controller struct {
	config           *rest.Config
	crClient         versioned.Interface
	clientset        kubernetes.Interface
	dynClient        dynamic.Interface
	osClient         osversioned.Interface
	recorder         record.EventRecorder
	actionSetTombMap sync.Map
	metrics          *metrics
}

// New create controller for watching kanister custom resources created
func New(c *rest.Config, reg prometheus.Registerer) *Controller {
	var m *metrics
	if reg != nil {
		m = newMetrics(reg)
	}
	return &Controller{
		config:  c,
		metrics: m,
	}
}

// StartWatch watches for instances of ActionSets and Blueprints acts on them.
func (c *Controller) StartWatch(ctx context.Context, namespace string) error {
	crClient, err := versioned.NewForConfig(c.config)
	if err != nil {
		return errors.Wrap(err, "failed to get a CustomResource client")
	}
	if err := checkCRAccess(ctx, crClient, namespace); err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return errors.Wrap(err, "failed to get a k8s client")
	}
	dynClient, err := dynamic.NewForConfig(c.config)
	if err != nil {
		return errors.Wrap(err, "failed to get a k8s dynamic client")
	}

	osClient, err := osversioned.NewForConfig(c.config)
	if err != nil {
		return errors.Wrap(err, "failed to get a openshift client")
	}

	c.crClient = crClient
	c.clientset = clientset
	c.dynClient = dynClient
	c.osClient = osClient
	c.recorder = eventer.NewEventRecorder(c.clientset, "Kanister Controller")

	for cr, o := range map[customresource.CustomResource]runtime.Object{
		crv1alpha1.ActionSetResource: &crv1alpha1.ActionSet{},
		crv1alpha1.BlueprintResource: &crv1alpha1.Blueprint{},
	} {
		resourceHandlers := cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		}
		watcher := customresource.NewWatcher(cr, namespace, resourceHandlers, crClient.CrV1alpha1().RESTClient())
		// TODO: remove this tmp channel once https://github.com/rook/operator-kit/pull/11 is merged.
		chTmp := make(chan struct{})
		go func() {
			<-ctx.Done()
			close(chTmp)
		}()
		go watcher.Watch(o, chTmp)
	}
	return nil
}

func checkCRAccess(ctx context.Context, cli versioned.Interface, ns string) error {
	if _, err := cli.CrV1alpha1().ActionSets(ns).List(ctx, metav1.ListOptions{}); err != nil {
		return errors.Wrap(err, "Could not list ActionSets")
	}
	if _, err := cli.CrV1alpha1().Blueprints(ns).List(ctx, metav1.ListOptions{}); err != nil {
		return errors.Wrap(err, "Could not list Blueprints")
	}
	if _, err := cli.CrV1alpha1().Profiles(ns).List(ctx, metav1.ListOptions{}); err != nil {
		return errors.Wrap(err, "Could not list Profiles")
	}
	return nil
}

func (c *Controller) incrementActionSetResolutionCounterVec(resolution string) {
	if c.metrics != nil {
		c.metrics.actionSetResolutionCounterVec.WithLabelValues(resolution).Inc()
	}
}

func (c *Controller) onAdd(obj interface{}) {
	o, ok := obj.(runtime.Object)
	if !ok {
		objType := fmt.Sprintf("%T", obj)
		log.Error().Print("Added object type does not implement runtime.Object", field.M{"ObjectType": objType})
		return
	}
	o = o.DeepCopyObject()
	switch v := o.(type) {
	case *crv1alpha1.ActionSet:
		t, ctx := c.LoadOrStoreTomb(context.Background(), v.Name)
		t.Go(func() error {
			if err := c.onAddActionSet(ctx, t, v); err != nil {
				log.Error().WithError(err).Print("Callback onAddActionSet() failed")
			}
			return nil
		})

	case *crv1alpha1.Blueprint:
		c.onAddBlueprint(v)
	default:
		objType := fmt.Sprintf("%T", o)
		log.Error().Print("Unknown object type", field.M{"ObjectType": objType})
	}
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	switch old := oldObj.(type) {
	case *crv1alpha1.ActionSet:
		new := newObj.(*crv1alpha1.ActionSet)
		if err := c.onUpdateActionSet(old, new); err != nil {
			bpName := new.Spec.Actions[0].Blueprint
			bp, _ := c.crClient.CrV1alpha1().Blueprints(new.GetNamespace()).Get(context.TODO(), bpName, metav1.GetOptions{})
			c.logAndErrorEvent(context.TODO(), "Callback onUpdateActionSet() failed:", "Error", err, new, bp)
		}
	case *crv1alpha1.Blueprint:
		new := newObj.(*crv1alpha1.Blueprint)
		c.onUpdateBlueprint(old, new)
	default:
		objType := fmt.Sprintf("%T", oldObj)
		log.Error().Print("Unknown object type", field.M{"ObjectType": objType})
	}
}

func (c *Controller) onDelete(obj interface{}) {
	switch v := obj.(type) {
	case *crv1alpha1.ActionSet:
		if err := c.onDeleteActionSet(v); err != nil {
			bpName := v.Spec.Actions[0].Blueprint
			bp, _ := c.crClient.CrV1alpha1().Blueprints(v.GetNamespace()).Get(context.TODO(), bpName, metav1.GetOptions{})
			c.logAndErrorEvent(context.TODO(), "Callback onDeleteActionSet() failed:", "Error", err, v, bp)
		}
	case *crv1alpha1.Blueprint:
		c.onDeleteBlueprint(v)
	default:
		objType := fmt.Sprintf("%T", obj)
		log.Error().Print("Unknown object type", field.M{"ObjectType": objType})
	}
}

func (c *Controller) onAddActionSet(ctx context.Context, t *tomb.Tomb, as *crv1alpha1.ActionSet) error {
	if err := validate.ActionSet(as); err != nil {
		return err
	}
	if as.Status == nil {
		c.initActionSetStatus(ctx, as)
	}
	as, err := c.crClient.CrV1alpha1().ActionSets(as.GetNamespace()).Get(ctx, as.GetName(), metav1.GetOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	if err = validate.ActionSet(as); err != nil {
		return err
	}
	return c.handleActionSet(ctx, t, as)
}

func (c *Controller) onAddBlueprint(bp *crv1alpha1.Blueprint) {
	c.logAndSuccessEvent(context.TODO(), fmt.Sprintf("Added blueprint %s", bp.GetName()), "Added", bp)
}

//nolint:unparam
func (c *Controller) onUpdateActionSet(oldAS, newAS *crv1alpha1.ActionSet) error {
	ctx := field.Context(context.Background(), consts.ActionsetNameKey, newAS.GetName())
	// adding labels with prefix "kanister.io/" in the context as field for better logging
	for key, value := range newAS.GetLabels() {
		if strings.HasPrefix(key, consts.LabelPrefix) {
			ctx = field.Context(ctx, key, value)
		}
	}

	if err := validate.ActionSet(newAS); err != nil {
		log.WithContext(ctx).Print("Updated ActionSet")
		return err
	}
	if newAS.Status == nil || newAS.Status.State != crv1alpha1.StateRunning {
		switch {
		case newAS.Status == nil:
			log.WithContext(ctx).Print("Updated ActionSet", field.M{"Status": "nil"})
		case newAS.Status.State == crv1alpha1.StateComplete:
			c.logAndSuccessEvent(ctx, fmt.Sprintf("Updated ActionSet '%s' Status->%s", newAS.Name, newAS.Status.State), "Update Complete", newAS)
		default:
			log.WithContext(ctx).Print("Updated ActionSet", field.M{"Status": newAS.Status.State})
		}
		return nil
	}
	for _, as := range newAS.Status.Actions {
		for _, p := range as.Phases {
			if p.State != crv1alpha1.StateComplete {
				log.WithContext(ctx).Print("Updated ActionSet", field.M{"Status": newAS.Status.State, "Phase": fmt.Sprintf("%s->%s", p.Name, p.State)})
				return nil
			}
		}
	}
	if len(newAS.Status.Actions) != 0 {
		return nil
	}
	return reconcile.ActionSet(context.TODO(), c.crClient.CrV1alpha1(), newAS.GetNamespace(), newAS.GetName(), func(ras *crv1alpha1.ActionSet) error {
		ras.Status.Progress.RunningPhase = ""
		ras.Status.State = crv1alpha1.StateComplete
		return nil
	})
}

//nolint:unparam
func (c *Controller) onUpdateBlueprint(oldBP, newBP *crv1alpha1.Blueprint) {
	log.Print("Updated Blueprint", field.M{"BlueprintName": newBP.Name})
}

//nolint:unparam
func (c *Controller) onDeleteActionSet(as *crv1alpha1.ActionSet) error {
	asName := as.GetName()
	log.Print("Deleted ActionSet", field.M{"ActionSetName": asName})
	v, ok := c.actionSetTombMap.Load(asName)
	if !ok {
		return nil
	}
	t, castOk := v.(*tomb.Tomb)
	if !castOk {
		return nil
	}
	t.Kill(nil) // TODO: @Deepika Give reason for ActionSet kill
	c.actionSetTombMap.Delete(asName)
	return nil
}

func (c *Controller) onDeleteBlueprint(bp *crv1alpha1.Blueprint) {
	log.Print("Deleted Blueprint ", field.M{"BlueprintName": bp.GetName()})
}

func (c *Controller) initActionSetStatus(ctx context.Context, as *crv1alpha1.ActionSet) {
	ctx = field.Context(ctx, consts.ActionsetNameKey, as.GetName())
	if as.Spec == nil {
		log.Error().WithContext(ctx).Print("Cannot initialize an ActionSet without a spec.")
		return
	}
	as.Status = &crv1alpha1.ActionSetStatus{State: crv1alpha1.StatePending}
	actions := make([]crv1alpha1.ActionStatus, 0, len(as.Spec.Actions))
	var err error
	for _, a := range as.Spec.Actions {
		if a.Blueprint == "" {
			// TODO: If no blueprint is specified, we should consider a default.
			err = errors.New("Blueprint is not specified for action")
			c.logAndErrorEvent(ctx, "Could not get blueprint:", "Blueprint not specified", err, as)
			break
		}
		var bp *crv1alpha1.Blueprint
		if bp, err = c.crClient.CrV1alpha1().Blueprints(as.GetNamespace()).Get(ctx, a.Blueprint, metav1.GetOptions{}); err != nil {
			err = errors.Wrap(err, "Failed to query blueprint")
			c.logAndErrorEvent(ctx, "Could not get blueprint:", "Error", err, as)
			break
		}
		var actionStatus *crv1alpha1.ActionStatus
		if actionStatus, err = c.initialActionStatus(a, bp); err != nil {
			reason := fmt.Sprintf("ActionSetFailed Action: %s", a.Name)
			c.logAndErrorEvent(ctx, "Could not get initial action:", reason, err, as, bp)
			break
		}
		actions = append(actions, *actionStatus)
	}
	if err != nil {
		as.Status.State = crv1alpha1.StateFailed
		as.Status.Progress.RunningPhase = ""
		as.Status.Error = crv1alpha1.Error{
			Message: err.Error(),
		}
	} else {
		as.Status.State = crv1alpha1.StatePending
		as.Status.Actions = actions
	}
	if _, err = c.crClient.CrV1alpha1().ActionSets(as.GetNamespace()).Update(ctx, as, metav1.UpdateOptions{}); err != nil {
		c.logAndErrorEvent(ctx, "Could not update ActionSet:", "Update Failed", err, as)
	}
}

func (c *Controller) initialActionStatus(a crv1alpha1.ActionSpec, bp *crv1alpha1.Blueprint) (*crv1alpha1.ActionStatus, error) {
	bpa, ok := bp.Actions[a.Name]
	if !ok {
		return nil, errors.Errorf("Action %s for object kind %s not found in blueprint %s", a.Name, a.Object.Kind, a.Blueprint)
	}
	phases := make([]crv1alpha1.Phase, 0, len(bpa.Phases))
	for _, p := range bpa.Phases {
		phases = append(phases, crv1alpha1.Phase{
			Name:  p.Name,
			State: crv1alpha1.StatePending,
		})
	}

	actionStatus := &crv1alpha1.ActionStatus{
		Name:      a.Name,
		Object:    a.Object,
		Blueprint: a.Blueprint,
		Phases:    phases,
		Artifacts: bpa.OutputArtifacts,
	}

	if bpa.DeferPhase != nil {
		actionStatus.DeferPhase = crv1alpha1.Phase{
			Name:  bpa.DeferPhase.Name,
			State: crv1alpha1.StatePending,
		}
	}

	return actionStatus, nil
}

func (c *Controller) handleActionSet(ctx context.Context, t *tomb.Tomb, as *crv1alpha1.ActionSet) (err error) {
	if as.Status == nil {
		return errors.New("ActionSet was not initialized")
	}
	if as.Status.State != crv1alpha1.StatePending {
		return nil
	}
	as.Status.State = crv1alpha1.StateRunning
	if as, err = c.crClient.CrV1alpha1().ActionSets(as.GetNamespace()).Update(ctx, as, metav1.UpdateOptions{}); err != nil {
		return errors.WithStack(err)
	}
	ctx = field.Context(ctx, consts.ActionsetNameKey, as.GetName())
	// adding labels with prefix "kanister.io/" in the context as field for better logging
	for key, value := range as.GetLabels() {
		if strings.HasPrefix(key, consts.LabelPrefix) {
			ctx = field.Context(ctx, key, value)
		}
	}

	for i, a := range as.Status.Actions {
		var bp *crv1alpha1.Blueprint
		if bp, err = c.crClient.CrV1alpha1().Blueprints(as.GetNamespace()).Get(ctx, a.Blueprint, metav1.GetOptions{}); err != nil {
			err = errors.Wrap(err, "Failed to query blueprint")
			c.logAndErrorEvent(ctx, "Could not get blueprint:", "Error", err, as)
			break
		}
		if err = c.runAction(ctx, t, as, i, bp); err != nil {
			// If runAction returns an error, it is a failure in the synchronous
			// part of running the action.
			reason := fmt.Sprintf("ActionSetFailed Action: %s", a.Name)
			c.logAndErrorEvent(ctx, fmt.Sprintf("Failed to launch Action %s:", as.GetName()), reason, err, as, bp)
			a.Phases[0].State = crv1alpha1.StateFailed
			break
		}
	}
	if err != nil {
		as.Status.State = crv1alpha1.StateFailed
		as.Status.Progress.RunningPhase = ""
		as.Status.Error = crv1alpha1.Error{
			Message: err.Error(),
		}
		_, err = c.crClient.CrV1alpha1().ActionSets(as.GetNamespace()).Update(ctx, as, metav1.UpdateOptions{})
		return errors.WithStack(err)
	}
	log.WithContext(ctx).Print("Created actionset and started executing actions", field.M{"NewActionSetName": as.GetName()})
	return nil
}

func (c *Controller) LoadOrStoreTomb(ctx context.Context, asName string) (*tomb.Tomb, context.Context) {
	var t *tomb.Tomb
	if v, ok := c.actionSetTombMap.Load(asName); ok {
		t = v.(*tomb.Tomb)
		return t, ctx
	}
	t, ctx = tomb.WithContext(ctx)
	c.actionSetTombMap.Store(asName, t)
	return t, ctx
}

//nolint:gocognit
func (c *Controller) runAction(ctx context.Context, t *tomb.Tomb, as *crv1alpha1.ActionSet, aIDX int, bp *crv1alpha1.Blueprint) error {
	action := as.Spec.Actions[aIDX]
	c.logAndSuccessEvent(ctx, fmt.Sprintf("Executing action %s", action.Name), "Started Action", as)
	tp, err := param.New(ctx, c.clientset, c.dynClient, c.crClient, c.osClient, action)
	if err != nil {
		c.incrementActionSetResolutionCounterVec(ActionSetCounterVecLabelResFailure)
		return err
	}
	phases, err := kanister.GetPhases(*bp, action.Name, action.PreferredVersion, *tp)
	if err != nil {
		c.incrementActionSetResolutionCounterVec(ActionSetCounterVecLabelResFailure)
		return err
	}

	// deferPhase is the phase that should be run after every successful or failed action run
	// can be specified in blueprint using actions[name].deferPhase
	deferPhase, err := kanister.GetDeferPhase(*bp, action.Name, action.PreferredVersion, *tp)
	if err != nil {
		c.incrementActionSetResolutionCounterVec(ActionSetCounterVecLabelResFailure)
		return err
	}

	ctx = field.Context(ctx, consts.ActionsetNameKey, as.GetName())
	t.Go(func() error {
		var coreErr error
		defer func() {
			var deferErr error
			if deferPhase != nil {
				c.updateActionSetRunningPhase(ctx, aIDX, as, deferPhase.Name())
				deferErr = c.executeDeferPhase(ctx, deferPhase, tp, bp, action.Name, aIDX, as)
			}
			// render artifacts only if all the phases are run successfully
			if deferErr == nil && coreErr == nil {
				c.renderActionsetArtifacts(ctx, as, aIDX, as.Namespace, as.Name, action.Name, bp, tp)
				c.maybeSetActionSetStateComplete(ctx, as, aIDX, bp, coreErr, deferErr)
				c.incrementActionSetResolutionCounterVec(ActionSetCounterVecLabelResSuccess)
			} else {
				c.incrementActionSetResolutionCounterVec(ActionSetCounterVecLabelResFailure)
			}
		}()

		for i, p := range phases {
			ctx = field.Context(ctx, consts.PhaseNameKey, p.Name())
			c.logAndSuccessEvent(ctx, fmt.Sprintf("Executing phase %s", p.Name()), "Started Phase", as)
			err = param.InitPhaseParams(ctx, c.clientset, tp, p.Name(), p.Objects())
			var output map[string]interface{}
			var msg string
			if err == nil {
				c.updateActionSetRunningPhase(ctx, aIDX, as, p.Name())
				progressTrackCtx, doneProgressTrack := context.WithCancel(ctx)
				defer doneProgressTrack()
				go func() {
					// progress update is computed on a best-effort basis.
					// if it exits with error, we will just log it.
					if err := progress.UpdateActionSetsProgress(progressTrackCtx, aIDX, c.crClient, as.GetName(), as.GetNamespace(), p); err != nil {
						log.Error().WithError(err)
					}
				}()
				output, err = p.Exec(ctx, *bp, action.Name, *tp)
				doneProgressTrack()
			} else {
				msg = fmt.Sprintf("Failed to init phase params: %#v:", as.Status.Actions[aIDX].Phases[i])
			}

			var rf func(*crv1alpha1.ActionSet) error
			if err != nil {
				coreErr = err
				rf = func(ras *crv1alpha1.ActionSet) error {
					ras.Status.Progress.RunningPhase = ""
					ras.Status.State = crv1alpha1.StateFailed
					ras.Status.Error = crv1alpha1.Error{
						Message: err.Error(),
					}
					ras.Status.Actions[aIDX].Phases[i].State = crv1alpha1.StateFailed
					return nil
				}
			} else {
				coreErr = nil
				rf = func(ras *crv1alpha1.ActionSet) error {
					ras.Status.Actions[aIDX].Phases[i].State = crv1alpha1.StateComplete
					pp, err := p.Progress()
					if err != nil {
						log.Error().WithError(err)
						return nil
					}
					ras.Status.Actions[aIDX].Phases[i].Progress = pp
					// this updates the phase output in the actionset status
					ras.Status.Actions[aIDX].Phases[i].Output = output
					if err := progress.SetActionSetPercentCompleted(ras); err != nil {
						log.Error().WithError(err)
					}
					return nil
				}
			}

			if rErr := reconcile.ActionSet(ctx, c.crClient.CrV1alpha1(), as.Namespace, as.Name, rf); rErr != nil {
				reason := fmt.Sprintf("ActionSetFailed Action: %s", as.Spec.Actions[aIDX].Name)
				msg := fmt.Sprintf("Failed to update phase: %#v:", as.Status.Actions[aIDX].Phases[i])
				c.logAndErrorEvent(ctx, msg, reason, rErr, as, bp)
				coreErr = rErr
				return nil
			}

			if err != nil {
				reason := fmt.Sprintf("ActionSetFailed Action: %s", as.Spec.Actions[aIDX].Name)
				if msg == "" {
					msg = fmt.Sprintf("Failed to execute phase: %#v:", as.Status.Actions[aIDX].Phases[i])
				}
				c.logAndErrorEvent(ctx, msg, reason, err, as, bp)
				coreErr = err
				return nil
			}
			param.UpdatePhaseParams(ctx, tp, p.Name(), output)
			c.logAndSuccessEvent(ctx, fmt.Sprintf("Completed phase %s", p.Name()), "Ended Phase", as)
		}
		return nil
	})
	return nil
}

// updateActionSetRunningPhase updates the actionset's `status.Progress.RunningPhase` with the phase name
// that is being run currently. It doesn't fail if there was a problem updating the actionset. It just logs
// the failure.
func (c *Controller) updateActionSetRunningPhase(ctx context.Context, aIDX int, as *crv1alpha1.ActionSet, phase string) {
	err := reconcile.ActionSet(ctx, c.crClient.CrV1alpha1(), as.Namespace, as.Name, func(as *crv1alpha1.ActionSet) error {
		as.Status.Progress.RunningPhase = phase
		// Iterate through all the phases and set current phase state to running
		for i := 0; i < len(as.Status.Actions[aIDX].Phases); i++ {
			if as.Status.Actions[aIDX].Phases[i].Name == phase {
				as.Status.Actions[aIDX].Phases[i].State = crv1alpha1.StateRunning
			}
		}
		return nil
	})
	if err != nil {
		log.Error().WithError(err).Print("Failed to update actionset with running phase")
	}
}

// executeDeferPhase executes the phase provided as a deferPhase in the blueprint action.
// deferPhase, if provided, must be run at the end of the blueprint action, irrespective of the
// statuses of the other phases. ActionSet `status.state` is going to be `complete` IFF all the
// phases and deferPhase are run successfully
// On failure, corresponding error messages are logged and recorded as events and the
// ActionSet's `status.state` is set to `failed`.
func (c *Controller) executeDeferPhase(ctx context.Context,
	deferPhase *kanister.Phase,
	tp *param.TemplateParams,
	bp *crv1alpha1.Blueprint,
	actionName string,
	aIDX int,
	as *crv1alpha1.ActionSet,
) error {
	actionsetName, actionsetNS := as.GetName(), as.GetNamespace()
	ctx = field.Context(ctx, consts.PhaseNameKey, as.Status.Actions[aIDX].DeferPhase.Name)
	c.logAndSuccessEvent(ctx, fmt.Sprintf("Executing deferPhase %s", as.Status.Actions[aIDX].DeferPhase.Name), "Started deferPhase", as)

	output, err := deferPhase.Exec(ctx, *bp, actionName, *tp)
	var rf func(*crv1alpha1.ActionSet) error
	if err != nil {
		rf = func(as *crv1alpha1.ActionSet) error {
			as.Status.Progress.RunningPhase = ""
			as.Status.State = crv1alpha1.StateFailed
			as.Status.Error = crv1alpha1.Error{
				Message: err.Error(),
			}
			as.Status.Actions[aIDX].DeferPhase.State = crv1alpha1.StateFailed
			return nil
		}
	} else {
		rf = func(as *crv1alpha1.ActionSet) error {
			as.Status.Actions[aIDX].DeferPhase.State = crv1alpha1.StateComplete
			as.Status.Actions[aIDX].DeferPhase.Output = output
			return nil
		}
	}
	var msg string
	if rErr := reconcile.ActionSet(ctx, c.crClient.CrV1alpha1(), actionsetNS, actionsetName, rf); rErr != nil {
		reason := fmt.Sprintf("ActionSetFailed Action: %s", as.Spec.Actions[aIDX].Name)
		msg := fmt.Sprintf("Failed to update defer phase: %#v:", as.Status.Actions[aIDX].DeferPhase)
		c.logAndErrorEvent(ctx, msg, reason, rErr, as, bp)
		return rErr
	}

	if err != nil {
		reason := fmt.Sprintf("ActionSetFailed Action: %s", as.Spec.Actions[aIDX].Name)
		if msg == "" {
			msg = fmt.Sprintf("Failed to execute defer phase: %#v:", as.Status.Actions[aIDX].DeferPhase)
		}
		c.logAndErrorEvent(ctx, msg, reason, err, as, bp)
		return err
	}

	c.logAndSuccessEvent(ctx, fmt.Sprintf("Completed deferPhase %s", as.Status.Actions[aIDX].DeferPhase.Name), "Ended deferPhase", as)
	param.UpdateDeferPhaseParams(ctx, tp, output)
	return nil
}

func (c *Controller) renderActionsetArtifacts(ctx context.Context,
	as *crv1alpha1.ActionSet,
	aIDX int,
	actionsetNS, actionsetName, actionName string,
	bp *crv1alpha1.Blueprint,
	tp *param.TemplateParams,
) {
	// Check if output artifacts are present
	artTpls := as.Status.Actions[aIDX].Artifacts
	if len(artTpls) == 0 {
		// No artifacts, will only update action status if failed
		af := updateActionSetArtifactsFun(aIDX, artTpls)
		if rErr := reconcile.ActionSet(ctx, c.crClient.CrV1alpha1(), actionsetNS, actionsetName, af); rErr != nil {
			reason := fmt.Sprintf("ActionSetFailed Action: %s", actionName)
			msg := fmt.Sprintf("Failed to update ActionSet: %s", actionsetName)
			c.logAndErrorEvent(ctx, msg, reason, rErr, as, bp)
		}
		return
	}
	// Render the artifacts
	arts, err := param.RenderArtifacts(artTpls, *tp)
	var af func(*crv1alpha1.ActionSet) error
	if err != nil {
		af = func(ras *crv1alpha1.ActionSet) error {
			ras.Status.State = crv1alpha1.StateFailed
			ras.Status.Progress.RunningPhase = ""
			ras.Status.Error = crv1alpha1.Error{
				Message: err.Error(),
			}
			return nil
		}
	} else {
		af = updateActionSetArtifactsFun(aIDX, arts)
	}
	// Update ActionSet
	if aErr := reconcile.ActionSet(ctx, c.crClient.CrV1alpha1(), actionsetNS, actionsetName, af); aErr != nil {
		reason := fmt.Sprintf("ActionSetFailed Action: %s", actionName)
		msg := fmt.Sprintf("Failed to update Output Artifacts: %#v:", artTpls)
		c.logAndErrorEvent(ctx, msg, reason, aErr, as, bp)
		return
	}

	if err != nil {
		reason := fmt.Sprintf("ActionSetFailed Action: %s", actionName)
		msg := "Failed to render output artifacts"
		c.logAndErrorEvent(ctx, msg, reason, err, as, bp)
		return
	}
}

func (c *Controller) maybeSetActionSetStateComplete(ctx context.Context,
	as *crv1alpha1.ActionSet,
	aIDX int,
	bp *crv1alpha1.Blueprint,
	coreErr, deferErr error,
) {
	af := func(ras *crv1alpha1.ActionSet) error {
		// Running phase is in the current action, TODO: #2270 track multiple phases
		if isPhaseInAction(ras.Status.Progress.RunningPhase, ras.Status.Actions[aIDX]) {
			// set the RunningPhase to empty string
			ras.Status.Progress.RunningPhase = ""
		}

		// make sure that the core phases that were run also didnt return any error
		// and then set actionset's state to be complete
		if coreErr != nil || deferErr != nil {
			ras.Status.State = crv1alpha1.StateFailed
			return nil
		}

		for _, as := range ras.Status.Actions {
			for _, p := range as.Phases {
				if p.State != crv1alpha1.StateComplete {
					log.WithContext(ctx).Print(
						"Finished action, but other action's phase is still running. Not setting state to complete.",
						field.M{
							"Status": ras.Status.State,
							"Action": ras.Status.Actions[aIDX].Name,
							"Phase":  fmt.Sprintf("%s->%s", p.Name, p.State)})
					return nil
				}
			}
		}

		// Set state to complete if it wasn't failed already
		if ras.Status.State != crv1alpha1.StateFailed {
			ras.Status.State = crv1alpha1.StateComplete
		}
		return nil
	}
	if rErr := reconcile.ActionSet(ctx, c.crClient.CrV1alpha1(), as.Namespace, as.Name, af); rErr != nil {
		reason := fmt.Sprintf("ActionSetFailed Action: %s", as.Status.Actions[aIDX].Name)
		msg := fmt.Sprintf("Failed to update ActionSet: %s", as.Name)
		c.logAndErrorEvent(ctx, msg, reason, rErr, as, bp)
	}
}

func (c *Controller) logAndErrorEvent(ctx context.Context, msg, reason string, err error, objects ...runtime.Object) {
	log.WithContext(ctx).WithError(err).Print(msg)
	if len(objects) == 0 {
		return
	}
	for _, object := range objects {
		o := object.DeepCopyObject()
		setObjectKind(o)
		// If no reference found then either the object points to an
		// empty struct or is an invalid object, so skip this object
		if _, refErr := reference.GetReference(scheme.Scheme, o); refErr != nil {
			continue
		}
		c.recorder.Event(o, corev1.EventTypeWarning, reason, fmt.Sprintf("%s %s", msg, err))
	}
}

func (c *Controller) logAndSuccessEvent(ctx context.Context, msg, reason string, objects ...runtime.Object) {
	log.WithContext(ctx).Print(msg)
	if len(objects) == 0 {
		return
	}
	for _, object := range objects {
		o := object.DeepCopyObject()
		setObjectKind(o)
		if _, refErr := reference.GetReference(scheme.Scheme, o); refErr != nil {
			continue
		}
		c.recorder.Event(o, corev1.EventTypeNormal, reason, msg)
	}
}

func setObjectKind(obj runtime.Object) {
	ok := obj.GetObjectKind()
	gvk := ok.GroupVersionKind()
	if gvk.Kind == "" {
		gvk.Kind = reflect.TypeOf(obj).Elem().Name()
	}
	ok.SetGroupVersionKind(gvk)
}

func updateActionSetArtifactsFun(aIDX int, arts map[string]crv1alpha1.Artifact) func(*crv1alpha1.ActionSet) error {
	return func(ras *crv1alpha1.ActionSet) error {
		ras.Status.Actions[aIDX].Artifacts = arts
		return nil
	}
}

func isPhaseInAction(phaseName string, actionStatus crv1alpha1.ActionStatus) bool {
	for _, p := range actionStatus.Phases {
		if p.Name == phaseName {
			return true
		}
	}
	return actionStatus.DeferPhase.Name == phaseName
}
