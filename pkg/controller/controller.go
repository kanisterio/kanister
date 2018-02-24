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

	"github.com/pkg/errors"
	opkit "github.com/rook/operator-kit"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/validate"
)

// Controller represents a controller object for kanister custom resources
type Controller struct {
	config    *rest.Config
	crClient  crclientv1alpha1.CrV1alpha1Interface
	clientset kubernetes.Interface
}

// New create controller for watching kanister custom resources created
func New(c *rest.Config) *Controller {
	return &Controller{
		config: c,
	}
}

// StartWatch watches for instances of ActionSets and Blueprints acts on them.
func (c *Controller) StartWatch(ctx context.Context, namespace string) error {
	crClient, err := crclientv1alpha1.NewForConfig(c.config)
	//crClient, err := crv1alpha1.NewForConfig(c.config)
	if err != nil {
		return errors.Wrap(err, "failed to get a CustomResource client")
	}
	clientset, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return errors.Wrap(err, "failed to get a k8s client")
	}
	c.crClient = crClient
	c.clientset = clientset

	for cr, o := range map[opkit.CustomResource]runtime.Object{
		crv1alpha1.ActionSetResource: &crv1alpha1.ActionSet{},
		crv1alpha1.BlueprintResource: &crv1alpha1.Blueprint{},
	} {
		resourceHandlers := cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		}
		watcher := opkit.NewWatcher(cr, namespace, resourceHandlers, crClient.RESTClient())
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

func (c *Controller) onAdd(obj interface{}) {
	o, ok := obj.(runtime.Object)
	if !ok {
		log.Errorf("Added object type <%T> does not implement runtime.Object", obj)
		return
	}
	o = o.DeepCopyObject()
	switch v := o.(type) {
	case *crv1alpha1.ActionSet:
		if err := c.onAddActionSet(v); err != nil {
			log.Errorf("Callback onAddActionSet() failed: %+v", err)
		}
	case *crv1alpha1.Blueprint:
		if err := c.onAddBlueprint(v); err != nil {
			log.Errorf("Callback onAddBlueprint() failed: %+v", err)
		}
	default:
		log.Errorf("Unkown object type <%T>", o)
	}
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	switch old := oldObj.(type) {
	case *crv1alpha1.ActionSet:
		new := newObj.(*crv1alpha1.ActionSet)
		if err := c.onUpdateActionSet(old, new); err != nil {
			log.Errorf("Callback onUpdateActionSet() failed: %+v", err)
		}
	case *crv1alpha1.Blueprint:
		new := newObj.(*crv1alpha1.Blueprint)
		if err := c.onUpdateBlueprint(old, new); err != nil {
			log.Errorf("Callback onUpdateBlueprint() failed: %+v", err)
		}
	default:
		log.Errorf("Unkown object type <%T>", oldObj)
	}
}

func (c *Controller) onDelete(obj interface{}) {
	switch v := obj.(type) {
	case *crv1alpha1.ActionSet:
		if err := c.onDeleteActionSet(v); err != nil {
			log.Errorf("Callback onDeleteActionSet() failed: %+v", err)
		}
	case *crv1alpha1.Blueprint:
		if err := c.onDeleteBlueprint(v); err != nil {
			log.Errorf("Callback onDeleteBlueprint() failed: %+v", err)
		}
	default:
		log.Errorf("Unkown object type <%T>", obj)
	}
}

func (c *Controller) onAddActionSet(as *crv1alpha1.ActionSet) error {
	as, err := c.crClient.ActionSets(as.GetNamespace()).Get(as.GetName(), v1.GetOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	if err := validate.ActionSet(as); err != nil {
		return err
	}
	c.initActionSetStatus(as)
	as, err = c.crClient.ActionSets(as.GetNamespace()).Get(as.GetName(), v1.GetOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	if err := validate.ActionSet(as); err != nil {
		return err
	}
	return c.handleActionSet(as)
}

func (c *Controller) onAddBlueprint(bp *crv1alpha1.Blueprint) error {
	log.Infof("Added blueprint %s", bp.GetName())
	return nil
}

func (c *Controller) onUpdateActionSet(oldAS, newAS *crv1alpha1.ActionSet) error {
	log.Infof("Updated ActionSet '%s'", newAS.Name)
	if err := validate.ActionSet(newAS); err != nil {
		return err
	}
	if newAS.Status == nil || newAS.Status.State != crv1alpha1.StateRunning {
		return nil
	}
	for _, as := range newAS.Status.Actions {
		for _, p := range as.Phases {
			if p.State != crv1alpha1.StateComplete {
				return nil
			}
		}
	}
	newAS.Status.State = crv1alpha1.StateComplete
	as, err := c.crClient.ActionSets(newAS.GetNamespace()).Update(newAS)
	if err != nil {
		return err
	}
	err = validate.ActionSet(as)
	return err
}

func (c *Controller) onUpdateBlueprint(oldBP, newBP *crv1alpha1.Blueprint) error {
	log.Infof("Updated Blueprint '%s' from %#v to %#v", newBP.Name, oldBP, newBP)
	return nil
}

func (c *Controller) onDeleteActionSet(as *crv1alpha1.ActionSet) error {
	log.Infof("Deleted ActionSet %s", as.GetName())
	return nil
}

func (c *Controller) onDeleteBlueprint(bp *crv1alpha1.Blueprint) error {
	log.Infof("Deleted Blueprint %s", bp.GetName())
	return nil
}

func (c *Controller) initActionSetStatus(as *crv1alpha1.ActionSet) {
	if as.Spec == nil {
		log.Error("Cannot initialize an ActionSet without a spec.")
		return
	}
	if as.Status != nil {
		log.Error("Cannot initialize non-nil ActionSet Status")
		return
	}
	as.Status = &crv1alpha1.ActionSetStatus{State: crv1alpha1.StatePending}
	actions := make([]crv1alpha1.ActionStatus, 0, len(as.Spec.Actions))
	var err error
	for _, a := range as.Spec.Actions {
		var actionStatus *crv1alpha1.ActionStatus
		actionStatus, err = c.initialActionStatus(as.GetNamespace(), a)
		if err != nil {
			log.Errorf("Could not get initial action: %#v", err)
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
	if _, err = c.crClient.ActionSets(as.GetNamespace()).Update(as); err != nil {
		log.Errorf("Could not update ActionSet: %#v", err)
	}
}

func (c *Controller) initialActionStatus(namespace string, a crv1alpha1.ActionSpec) (*crv1alpha1.ActionStatus, error) {
	if a.Blueprint == "" {
		// TODO: If no blueprint is specified, we should consider a default.
		return nil, errors.New("Blueprint not specified")
	}
	bp, err := c.crClient.Blueprints(namespace).Get(a.Blueprint, v1.GetOptions{})
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
	if as, err = c.crClient.ActionSets(as.GetNamespace()).Update(as); err != nil {
		return errors.WithStack(err)
	}
	for i := range as.Status.Actions {
		if err = c.runAction(context.TODO(), as, i); err != nil {
			// If runAction returns an error, it is a failure in the synchronous
			// part of running the action.
			log.Errorf("Failed launch Action %s: %#v", as.GetName(), err)
			as.Status.State = crv1alpha1.StateFailed
			_, err = c.crClient.ActionSets(as.GetNamespace()).Update(as)
			return errors.WithStack(err)
		}
	}
	log.Infof("Started all actions %s", as.GetName())
	return nil
}

func (c *Controller) runAction(ctx context.Context, as *crv1alpha1.ActionSet, aIDX int) error {
	log.Infof("Executing action %#v", as.Spec.Actions[aIDX])
	bpName := as.Spec.Actions[aIDX].Blueprint
	action := as.Spec.Actions[aIDX]
	bp, err := c.crClient.Blueprints(as.GetNamespace()).Get(bpName, v1.GetOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	tp, err := param.New(ctx, c.clientset, action)
	if err != nil {
		return err
	}
	artTpls := as.Status.Actions[aIDX].Artifacts
	arts, err := param.RenderArtifacts(artTpls, *tp)
	if err != nil {
		return err
	}
	tp.ArtifactsOut = arts
	phases, err := kanister.GetPhases(*bp, action.Name, *tp)
	if err != nil {
		return err
	}
	go func() {
		for i, p := range phases {
			log.Debugf("Executing phase: %s", p.Name())
			err = p.Exec(ctx)
			if err != nil {
				log.Errorf("Failed to run phase %d of action %s: %+v", i, as.Spec.Actions[aIDX].Name, err)
			}
			// We need to refresh the actionset in case there were other updates.
			// TODO: We should retry, backoff here to make this more robust.
			as, getErr := c.crClient.ActionSets(as.Namespace).Get(as.GetName(), v1.GetOptions{})
			if getErr != nil {
				log.Errorf("Failed to get recent ActionSet: %s, Cause: %#v", as.GetName(), err)
				return
			}
			if valErr := validate.ActionSet(as); valErr != nil {
				log.Errorf("Validation Failed: %+v", valErr)
				return
			}
			// This means another action in this action set failed. We bail as this is a terminal state.
			if as.Status.State == crv1alpha1.StateFailed {
				return
			}
			if err != nil {
				as.Status.State = crv1alpha1.StateFailed
				as.Status.Actions[aIDX].Phases[i].State = crv1alpha1.StateFailed
				if _, err := c.crClient.ActionSets(as.GetNamespace()).Update(as); err != nil {
					log.Errorf("Failed to update %#v: %#v", as.Status.Actions[aIDX].Phases[i], err)
				}
				return
			}
			as.Status.Actions[aIDX].Artifacts = arts
			as.Status.Actions[aIDX].Phases[i].State = crv1alpha1.StateComplete
			if _, err := c.crClient.ActionSets(as.GetNamespace()).Update(as); err != nil {
				log.Errorf("Failed to update phase: %#v: %#v", as.Status.Actions[aIDX].Phases[i], err)
				return
			}
		}
	}()
	return nil
}
