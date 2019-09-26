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
	"sync"

	customresource "github.com/kanisterio/kanister/pkg/customresource"
	"github.com/pkg/errors"
	"gopkg.in/tomb.v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	"github.com/kanisterio/kanister/pkg/eventer"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/reconcile"
	"github.com/kanisterio/kanister/pkg/validate"
)

// Controller represents a controller object for kanister custom resources
type Controller struct {
	config           *rest.Config
	crClient         versioned.Interface
	clientset        kubernetes.Interface
	dynClient        dynamic.Interface
	recorder         record.EventRecorder
	actionSetTombMap sync.Map
}

// New create controller for watching kanister custom resources created
func New(c *rest.Config) *Controller {
	return &Controller{
		config: c,
	}
}

// StartWatch watches for instances of ActionSets and Blueprints acts on them.
func (c *Controller) StartWatch(ctx context.Context, namespace string) error {
	crClient, err := versioned.NewForConfig(c.config)
	if err != nil {
		return errors.Wrap(err, "failed to get a CustomResource client")
	}
	if err := checkCRAccess(crClient, namespace); err != nil {
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

	c.crClient = crClient
	c.clientset = clientset
	c.dynClient = dynClient
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
			select {
			case <-ctx.Done():
				close(chTmp)
			}
		}()
		go watcher.Watch(o, chTmp)
	}
	return nil
}

func checkCRAccess(cli versioned.Interface, ns string) error {
	if _, err := cli.CrV1alpha1().ActionSets(ns).List(v1.ListOptions{}); err != nil {
		return errors.Wrap(err, "Could not list ActionSets")
	}
	if _, err := cli.CrV1alpha1().Blueprints(ns).List(v1.ListOptions{}); err != nil {
		return errors.Wrap(err, "Could not list Blueprints")
	}
	if _, err := cli.CrV1alpha1().Profiles(ns).List(v1.ListOptions{}); err != nil {
		return errors.Wrap(err, "Could not list Profiles")
	}
	return nil
}

func (c *Controller) onAdd(obj interface{}) {
	o, ok := obj.(runtime.Object)
	if !ok {
		log.Error().Print(fmt.Sprintf("Added object type <%T> does not implement runtime.Object", obj))
		return
	}
	o = o.DeepCopyObject()
	switch v := o.(type) {
	case *crv1alpha1.ActionSet:
		if err := c.onAddActionSet(v); err != nil {
			log.Error().WithError(err).Print("Callback onAddActionSet() failed")
		}
	case *crv1alpha1.Blueprint:
		if err := c.onAddBlueprint(v); err != nil {
			log.Error().WithError(err).Print("Callback onAddBlueprint() failed")
		}
	default:
		log.Error().Print(fmt.Sprintf("Unknown object type <%T>", o))
	}
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	switch old := oldObj.(type) {
	case *crv1alpha1.ActionSet:
		new := newObj.(*crv1alpha1.ActionSet)
		if err := c.onUpdateActionSet(old, new); err != nil {
			bpName := new.Spec.Actions[0].Blueprint
			bp, _ := c.crClient.CrV1alpha1().Blueprints(new.GetNamespace()).Get(bpName, v1.GetOptions{})
			c.logAndErrorEvent(nil, "Callback onUpdateActionSet() failed:", "Error", err, new, bp)

		}
	case *crv1alpha1.Blueprint:
		new := newObj.(*crv1alpha1.Blueprint)
		if err := c.onUpdateBlueprint(old, new); err != nil {
			c.logAndErrorEvent(nil, "Callback onUpdateBlueprint() failed:", "Error", err, new)
		}
	default:
		log.Error().Print(fmt.Sprintf("Unknown object type <%T>", oldObj))
	}
}

func (c *Controller) onDelete(obj interface{}) {
	switch v := obj.(type) {
	case *crv1alpha1.ActionSet:
		if err := c.onDeleteActionSet(v); err != nil {
			bpName := v.Spec.Actions[0].Blueprint
			bp, _ := c.crClient.CrV1alpha1().Blueprints(v.GetNamespace()).Get(bpName, v1.GetOptions{})
			c.logAndErrorEvent(nil, "Callback onDeleteActionSet() failed:", "Error", err, v, bp)
		}
	case *crv1alpha1.Blueprint:
		if err := c.onDeleteBlueprint(v); err != nil {
			c.logAndErrorEvent(nil, "Callback onDeleteBlueprint() failed:", "Error", err, v)
		}
	default:
		log.Error().Print(fmt.Sprintf("Unknown object type <%T>", obj))
	}
}

func (c *Controller) onAddActionSet(as *crv1alpha1.ActionSet) error {
	as, err := c.crClient.CrV1alpha1().ActionSets(as.GetNamespace()).Get(as.GetName(), v1.GetOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	if err := validate.ActionSet(as); err != nil {
		return err
	}
	c.initActionSetStatus(as)
	as, err = c.crClient.CrV1alpha1().ActionSets(as.GetNamespace()).Get(as.GetName(), v1.GetOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	if err := validate.ActionSet(as); err != nil {
		return err
	}
	return c.handleActionSet(as)
}

func (c *Controller) onAddBlueprint(bp *crv1alpha1.Blueprint) error {
	c.logAndSuccessEvent(nil, fmt.Sprintf("Added blueprint %s", bp.GetName()), "Added", bp)
	return nil
}

func (c *Controller) onUpdateActionSet(oldAS, newAS *crv1alpha1.ActionSet) error {
	if err := validate.ActionSet(newAS); err != nil {
		log.Print(fmt.Sprintf("Updated ActionSet '%s'", newAS.Name))
		return err
	}
	if newAS.Status == nil || newAS.Status.State != crv1alpha1.StateRunning {
		if newAS.Status == nil {
			log.Print(fmt.Sprintf("Updated ActionSet '%s' Status->nil", newAS.Name))
		} else if newAS.Status.State == crv1alpha1.StateComplete {
			c.logAndSuccessEvent(nil, fmt.Sprintf("Updated ActionSet '%s' Status->%s", newAS.Name, newAS.Status.State), "Update Complete", newAS)
		} else {
			log.Print(fmt.Sprintf("Updated ActionSet '%s' Status->%s", newAS.Name, newAS.Status.State))
		}
		return nil
	}
	for _, as := range newAS.Status.Actions {
		for _, p := range as.Phases {
			if p.State != crv1alpha1.StateComplete {
				log.Print(fmt.Sprintf("Updated ActionSet '%s' Status->%s, Phase: %s->%s", newAS.Name, newAS.Status.State, p.Name, p.State))
				return nil
			}
		}
	}
	if len(newAS.Status.Actions) != 0 {
		return nil
	}
	return reconcile.ActionSet(context.TODO(), c.crClient.CrV1alpha1(), newAS.GetNamespace(), newAS.GetName(), func(ras *crv1alpha1.ActionSet) error {
		ras.Status.State = crv1alpha1.StateComplete
		return nil
	})
}

func (c *Controller) onUpdateBlueprint(oldBP, newBP *crv1alpha1.Blueprint) error {
	log.Print(fmt.Sprintf("Updated Blueprint '%s'", newBP.Name))
	return nil
}

func (c *Controller) onDeleteActionSet(as *crv1alpha1.ActionSet) error {
	asName := as.GetName()
	log.Print(fmt.Sprintf("Deleted ActionSet %s", asName))
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

func (c *Controller) onDeleteBlueprint(bp *crv1alpha1.Blueprint) error {
	log.Print(fmt.Sprintf("Deleted Blueprint %s", bp.GetName()))
	return nil
}

func (c *Controller) initActionSetStatus(as *crv1alpha1.ActionSet) {
	ctx := context.Background()
	ctx = field.Context(ctx, consts.ActionsetNameKey, as.GetName())
	if as.Spec == nil {
		log.Error().WithContext(ctx).Print("Cannot initialize an ActionSet without a spec.")
		return
	}
	if as.Status != nil {
		log.Error().WithContext(ctx).Print("Cannot initialize non-nil ActionSet Status")
		return
	}
	as.Status = &crv1alpha1.ActionSetStatus{State: crv1alpha1.StatePending}
	actions := make([]crv1alpha1.ActionStatus, 0, len(as.Spec.Actions))
	var err error
	for _, a := range as.Spec.Actions {
		var actionStatus *crv1alpha1.ActionStatus
		actionStatus, err = c.initialActionStatus(as.GetNamespace(), a)
		if err != nil {
			bp, _ := c.crClient.CrV1alpha1().Blueprints(as.GetNamespace()).Get(a.Blueprint, v1.GetOptions{})
			reason := fmt.Sprintf("ActionSetFailed Action: %s", a.Name)
			c.logAndErrorEvent(ctx, "Could not get initial action:", reason, err, as, bp)
			break
		}
		actions = append(actions, *actionStatus)
	}
	if err != nil {
		as.Status.State = crv1alpha1.StateFailed
	} else {
		as.Status.State = crv1alpha1.StatePending
		as.Status.Actions = actions
	}
	if _, err = c.crClient.CrV1alpha1().ActionSets(as.GetNamespace()).Update(as); err != nil {
		c.logAndErrorEvent(ctx, "Could not update ActionSet:", "Update Failed", err, as)
	}
}

func (c *Controller) initialActionStatus(namespace string, a crv1alpha1.ActionSpec) (*crv1alpha1.ActionStatus, error) {
	if a.Blueprint == "" {
		// TODO: If no blueprint is specified, we should consider a default.
		return nil, errors.New("Blueprint not specified")
	}
	bp, err := c.crClient.CrV1alpha1().Blueprints(namespace).Get(a.Blueprint, v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query blueprint")
	}
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
	return &crv1alpha1.ActionStatus{
		Name:      a.Name,
		Object:    a.Object,
		Blueprint: a.Blueprint,
		Phases:    phases,
		Artifacts: bpa.OutputArtifacts,
	}, nil

}

func (c *Controller) handleActionSet(as *crv1alpha1.ActionSet) (err error) {
	if as.Status == nil {
		return errors.New("ActionSet was not initialized")
	}
	if as.Status.State != crv1alpha1.StatePending {
		return nil
	}
	as.Status.State = crv1alpha1.StateRunning
	if as, err = c.crClient.CrV1alpha1().ActionSets(as.GetNamespace()).Update(as); err != nil {
		return errors.WithStack(err)
	}
	ctx := context.Background()
	ctx = field.Context(ctx, consts.ActionsetNameKey, as.GetName())
	for i := range as.Status.Actions {
		if err = c.runAction(ctx, as, i); err != nil {
			// If runAction returns an error, it is a failure in the synchronous
			// part of running the action.
			bpName := as.Spec.Actions[i].Blueprint
			bp, _ := c.crClient.CrV1alpha1().Blueprints(as.GetNamespace()).Get(bpName, v1.GetOptions{})
			reason := fmt.Sprintf("ActionSetFailed Action: %s", as.Status.Actions[i].Name)
			c.logAndErrorEvent(ctx, fmt.Sprintf("Failed to launch Action %s:", as.GetName()), reason, err, as, bp)
			as.Status.State = crv1alpha1.StateFailed
			as.Status.Actions[i].Phases[0].State = crv1alpha1.StateFailed
			_, err = c.crClient.CrV1alpha1().ActionSets(as.GetNamespace()).Update(as)
			return errors.WithStack(err)
		}
	}
	log.WithContext(ctx).Print(fmt.Sprintf("Created actionset %s and started executing actions", as.GetName()))
	return nil
}

func (c *Controller) runAction(ctx context.Context, as *crv1alpha1.ActionSet, aIDX int) error {
	action := as.Spec.Actions[aIDX]
	c.logAndSuccessEvent(ctx, fmt.Sprintf("Executing action %s", action.Name), "Started Action", as)
	bpName := as.Spec.Actions[aIDX].Blueprint
	bp, err := c.crClient.CrV1alpha1().Blueprints(as.GetNamespace()).Get(bpName, v1.GetOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	tp, err := param.New(ctx, c.clientset, c.dynClient, c.crClient, action)
	if err != nil {
		return err
	}
	phases, err := kanister.GetPhases(*bp, action.Name, *tp)
	if err != nil {
		return err
	}
	ns, name := as.GetNamespace(), as.GetName()
	var t *tomb.Tomb
	t, ctx = tomb.WithContext(ctx)
	c.actionSetTombMap.Store(as.Name, t)
	ctx = field.Context(ctx, consts.ActionsetNameKey, as.GetName())
	t.Go(func() error {
		for i, p := range phases {
			ctx = field.Context(ctx, consts.PhaseNameKey, p.Name())
			c.logAndSuccessEvent(ctx, fmt.Sprintf("Executing phase %s", p.Name()), "Started Phase", as)
			err = param.InitPhaseParams(ctx, c.clientset, tp, p.Name(), p.Objects())
			var output map[string]interface{}
			var msg string
			if err == nil {
				output, err = p.Exec(ctx, *bp, action.Name, *tp)
			} else {
				msg = fmt.Sprintf("Failed to init phase params: %#v:", as.Status.Actions[aIDX].Phases[i])
			}
			var rf func(*crv1alpha1.ActionSet) error
			if err != nil {
				rf = func(ras *crv1alpha1.ActionSet) error {
					ras.Status.State = crv1alpha1.StateFailed
					ras.Status.Actions[aIDX].Phases[i].State = crv1alpha1.StateFailed
					return nil
				}
			} else {
				rf = func(ras *crv1alpha1.ActionSet) error {
					ras.Status.Actions[aIDX].Phases[i].State = crv1alpha1.StateComplete
					ras.Status.Actions[aIDX].Phases[i].Output = output
					return nil
				}
			}
			if rErr := reconcile.ActionSet(ctx, c.crClient.CrV1alpha1(), ns, name, rf); rErr != nil {
				reason := fmt.Sprintf("ActionSetFailed Action: %s", as.Spec.Actions[aIDX].Name)
				msg := fmt.Sprintf("Failed to update phase: %#v:", as.Status.Actions[aIDX].Phases[i])
				c.logAndErrorEvent(ctx, msg, reason, rErr, as, bp)
				return nil
			}
			if err != nil {
				reason := fmt.Sprintf("ActionSetFailed Action: %s", as.Spec.Actions[aIDX].Name)
				if msg == "" {
					msg = fmt.Sprintf("Failed to execute phase: %#v:", as.Status.Actions[aIDX].Phases[i])
				}
				c.logAndErrorEvent(ctx, msg, reason, err, as, bp)
				return nil
			}
			param.UpdatePhaseParams(ctx, tp, p.Name(), output)
			c.logAndSuccessEvent(ctx, fmt.Sprintf("Completed phase %s", p.Name()), "Ended Phase", as)
		}
		// Check if output artifacts are present
		artTpls := as.Status.Actions[aIDX].Artifacts
		if len(artTpls) == 0 {
			// No artifacts, set ActionSetStatus to complete
			if rErr := reconcile.ActionSet(ctx, c.crClient.CrV1alpha1(), ns, name, func(ras *crv1alpha1.ActionSet) error {
				ras.Status.State = crv1alpha1.StateComplete
				return nil
			}); rErr != nil {
				reason := fmt.Sprintf("ActionSetFailed Action: %s", action.Name)
				msg := fmt.Sprintf("Failed to update ActionSet: %s", name)
				c.logAndErrorEvent(ctx, msg, reason, rErr, as, bp)
			}
			return nil
		}
		// Render the artifacts
		arts, err := param.RenderArtifacts(artTpls, *tp)
		var af func(*crv1alpha1.ActionSet) error
		if err != nil {
			af = func(ras *crv1alpha1.ActionSet) error {
				ras.Status.State = crv1alpha1.StateFailed
				return nil
			}
		} else {
			af = func(ras *crv1alpha1.ActionSet) error {
				ras.Status.Actions[aIDX].Artifacts = arts
				ras.Status.State = crv1alpha1.StateComplete
				return nil
			}
		}
		// Update ActionSet
		if aErr := reconcile.ActionSet(ctx, c.crClient.CrV1alpha1(), ns, name, af); aErr != nil {
			reason := fmt.Sprintf("ActionSetFailed Action: %s", action.Name)
			msg := fmt.Sprintf("Failed to update Output Artifacts: %#v:", artTpls)
			c.logAndErrorEvent(ctx, msg, reason, aErr, as, bp)
			return nil
		}
		if err != nil {
			reason := fmt.Sprintf("ActionSetFailed Action: %s", action.Name)
			msg := "Failed to render output artifacts"
			c.logAndErrorEvent(ctx, msg, reason, err, as, bp)
			return nil
		}
		return nil
	})
	return nil
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
