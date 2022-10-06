// Copyright 2022 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repositoryserver

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/customresource"
	"github.com/kanisterio/kanister/pkg/eventer"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

// Controller represents a controller object for RepositoryServer resource
type Controller struct {
	ctx       context.Context
	config    *rest.Config
	crClient  versioned.Interface
	clientset kubernetes.Interface
	dynClient dynamic.Interface
	recorder  record.EventRecorder
}

// NewController create controller for watching RepositoryServer resource created
func NewController(ctx context.Context, config *rest.Config) *Controller {
	return &Controller{
		ctx:    ctx,
		config: config,
	}
}

// StartWatch watches for instances of RepositoryServer and acts on them.
func (c *Controller) StartWatch(namespace string) error {
	crClient, err := versioned.NewForConfig(c.config)
	if err != nil {
		return errors.Wrap(err, "failed to get a CustomResource client")
	}
	if err := checkRepositoryServersAccess(c.ctx, crClient, namespace); err != nil {
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
	c.recorder = eventer.NewEventRecorder(c.clientset, "RepositoryServer Controller")

	resourceHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		DeleteFunc: c.onDelete,
	}
	watcher := customresource.NewWatcher(crv1alpha1.RepositoryServerResource, namespace, resourceHandlers, crClient.CrV1alpha1().RESTClient())
	chTmp := make(chan struct{})
	go func() {
		<-c.ctx.Done()
		close(chTmp)
	}()
	go watcher.Watch(&crv1alpha1.RepositoryServer{}, chTmp)

	return nil
}

func checkRepositoryServersAccess(ctx context.Context, cli versioned.Interface, ns string) error {
	if _, err := cli.CrV1alpha1().RepositoryServers(ns).List(ctx, v1.ListOptions{}); err != nil {
		return errors.Wrap(err, "Could not list RepositoryServers")
	}
	return nil
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
	case *crv1alpha1.RepositoryServer:
		if err := c.onAddRepositoryServer(v); err != nil {
			log.Error().WithError(err).Print("Callback onAddRepositoryServer() failed")
		}
	default:
		objType := fmt.Sprintf("%T", o)
		log.Error().Print("Unknown object type", field.M{"ObjectType": objType})
	}
}

func (c *Controller) onDelete(obj interface{}) {
	switch v := obj.(type) {
	case *crv1alpha1.RepositoryServer:
		if err := c.onDeleteRepositoryServer(v); err != nil {
			log.Error().WithError(err).Print("Callback onDeleteRepositoryServer() failed")
		}
	default:
		objType := fmt.Sprintf("%T", obj)
		log.Error().Print("Unknown object type", field.M{"ObjectType": objType})
	}
}

func (c *Controller) newRepositoryServerHandler(repositoryServer *crv1alpha1.RepositoryServer) Handler {
	repoServerCROwnerRef := metav1.OwnerReference{
		APIVersion: fmt.Sprintf("%s/%s", crv1alpha1.ResourceGroup, crv1alpha1.SchemeVersion),
		Kind:       "RepositoryServer",
		Name:       repositoryServer.Name,
		UID:        repositoryServer.UID,
	}
	return Handler{
		Ctx:              c.ctx,
		KubeCli:          c.clientset,
		CrCli:            c.crClient,
		RepositoryServer: repositoryServer,
		OwnerReference:   repoServerCROwnerRef,
	}
}

func (c *Controller) onAddRepositoryServer(rs *crv1alpha1.RepositoryServer) error {
	log.Info().Print("Successfully created RepositoryServer CR named " + rs.Name)
	handler := c.newRepositoryServerHandler(rs)
	if err := handler.RunRepositoryProxyServer(); err != nil {
		return errors.Wrap(err, "Failed to run RepositoryServer pod")
	}
	return nil
}

func (c *Controller) onDeleteRepositoryServer(rs *crv1alpha1.RepositoryServer) error {
	log.Info().Print("Successfully deleted RepositoryServer CR named " + rs.Name)
	return nil
}
