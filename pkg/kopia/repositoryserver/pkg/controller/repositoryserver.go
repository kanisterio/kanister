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

package controller

import (
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	rsclientset "github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/client/clientset/versioned"
	rsscheme "github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/client/clientset/versioned/scheme"
	rsinformers "github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/client/informers/externalversions/cr.kanister.io/v1alpha1"
	rslister "github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/client/listers/cr.kanister.io/v1alpha1"
)

type Controller struct {
	kubeClientSet kubernetes.Interface
	rsClientSet   rsclientset.Interface
	rsLister      rslister.RepositoryServerLister
	workQueue     workqueue.RateLimitingInterface
	rsCacheSynced cache.InformerSynced
	recorder      record.EventRecorder
}

func NewController(kubeClientSet kubernetes.Interface, clientset rsclientset.Interface, rsInformer rsinformers.RepositoryServerInformer) *Controller {
	// Create event broadcaster
	// Add controller types to the default Kubernetes Scheme so Events can be
	// logged for controller types.
	utilruntime.Must(rsscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "repository-server"})

	controller := &Controller{
		rsClientSet:   clientset,
		rsLister:      rsInformer.Lister(),
		workQueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "repository-server"),
		rsCacheSynced: rsInformer.Informer().HasSynced,
		recorder:      recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when RepositoryServer resources change
	rsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.handleAdd,
			DeleteFunc: controller.handleDel,
		},
	)

	return controller
}

func (c *Controller) handleAdd(obj interface{}) {
	log.Println("handleAdd was called")
	c.workQueue.Add(obj)
}

func (c *Controller) handleDel(obj interface{}) {
	log.Println("handleDel was called")
	c.workQueue.Done(obj)
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	log.Println("Starting RepositoryServer Controller....")
	if !cache.WaitForCacheSync(stopCh, c.rsCacheSynced) {
		log.Println("Waiting for the cache to be synced....")
	}

	go wait.Until(c.worker, 1*time.Second, stopCh)

	<-stopCh
	return nil
}

func (c *Controller) worker() {
	for c.processItem() {
	}
}

func (c *Controller) processItem() bool {
	item, shutdown := c.workQueue.Get()
	if shutdown {
		return false
	}

	defer c.workQueue.Forget(item)

	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		log.Printf("Error %s calling Namespace key func on cache for item", err.Error())
		return false
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Printf("Splitting key into namespace and name, error %s\n", err.Error())
		return false
	}

	err = c.logOutputAfterCreateCR(ns, name)
	if err != nil {
		log.Println("Error while listing CR: ", err.Error())
		return false
	}
	return true
}

func (c *Controller) logOutputAfterCreateCR(ns, name string) error {
	reposerver, err := c.rsLister.RepositoryServers(ns).Get(name)
	if err != nil {
		log.Printf("Error in listing RepositoryServer resource: %s", err.Error())
		return err
	}

	log.Printf("Successfully created RepositoryServer CR named %s", reposerver.Name)
	return nil
}

func (c *Controller) startRepositoryServerPod() {

}

func (c *Controller) createService() {

}

func (c *Controller) createNetworkPolicy() {

}

func (c *Controller) addTLSCertConfigurationInPodOverride() {

}

func (c *Controller) createRepoServerPod() {

}

func (c *Controller) waitForPodReady() {

}

func (c *Controller) connectToRepository() {

}

func (c *Controller) startRepoProxyServer() {

}

func (c *Controller) waitForServerToStart() {

}

func (c *Controller) addClientUsersToServer() {

}

func (c *Controller) refreshServer() {

}
