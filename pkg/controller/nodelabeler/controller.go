/*
Copyright 2019 Sorbonne Université

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodelabeler

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"headnode/pkg/authorization"
	"headnode/pkg/node"

	log "github.com/Sirupsen/logrus"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// The main structure of controller
type controller struct {
	logger    *log.Entry
	clientset kubernetes.Interface
	queue     workqueue.RateLimitingInterface
	informer  cache.SharedIndexInformer
	handler   HandlerInterface
}

// Start function is entry point of the controller
func Start() {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	// Create the shared informer to list and watch node resources
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			// The main purpose of listing is to attach geo labels to whole nodes at the beginning
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return clientset.CoreV1().Nodes().List(options)
			},
			// This function watches all changes/updates of nodes
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return clientset.CoreV1().Nodes().Watch(options)
			},
		},
		&core_v1.Node{},
		0,
		cache.Indexers{},
	)
	// Create a work queue which contains a key of the resource to be handled by the handler
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	// Event handlers deal with events of resources. In here, we take into consideration of adding and updating nodes.
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Put the resource object into a key
			key, err := cache.MetaNamespaceKeyFunc(obj)
			log.Infof("Add node detected: %s", key)
			if err == nil {
				// Add the key to the queue
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			updated := node.CompareIPAddresses(oldObj.(*core_v1.Node), newObj.(*core_v1.Node))
			if updated {
				key, err := cache.MetaNamespaceKeyFunc(newObj)
				log.Infof("Update node detected: %s", key)
				if err == nil {
					queue.Add(key)
				}
			}
		},
	})
	controller := controller{
		logger:    log.NewEntry(log.New()),
		clientset: clientset,
		informer:  informer,
		queue:     queue,
		handler:   &Handler{},
	}

	// A channel to terminate elegantly
	stopCh := make(chan struct{})
	defer close(stopCh)
	// Run the controller loop as a background task to start processing resources
	go controller.run(stopCh)
	// A channel to observe OS signals for smooth shut down
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	signal.Notify(sigTerm, syscall.SIGINT)
	<-sigTerm
}

// Run starts the controller loop
func (c *controller) run(stopCh <-chan struct{}) {
	// A Go panic which includes logging and terminating
	defer utilruntime.HandleCrash()
	// Shutdown after all goroutines have done
	defer c.queue.ShutDown()
	c.logger.Info("run: initiating")

	// Run the informer to list and watch resources
	go c.informer.Run(stopCh)

	// Synchronization to settle resources one
	if !cache.WaitForCacheSync(stopCh, c.hasSynced) {
		utilruntime.HandleError(fmt.Errorf("Error syncing cache"))
		return
	}
	c.logger.Info("run: cache sync complete")
	// Operate the runWorker
	wait.Until(c.runWorker, time.Second, stopCh)
}

// To link the informer's HasSynced method to the Controller interface
func (c *controller) hasSynced() bool {
	return c.informer.HasSynced()
}

// To process new objects added to the queue
func (c *controller) runWorker() {
	log.Info("runWorker: starting")
	// Run processNextItem for all the changes
	for c.processNextItem() {
		log.Info("runWorker: processing next item")
	}

	log.Info("runWorker: completed")
}

// This function deals with the queue and sends each item in it to the specified handler to be processed.
func (c *controller) processNextItem() bool {
	log.Info("processNextItem: start")
	// Fetch the next item of the queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	// Get the key string
	keyRaw := key.(string)
	// Use the string key to get the object from the indexer
	item, exists, err := c.informer.GetIndexer().GetByKey(keyRaw)
	if err != nil {
		if c.queue.NumRequeues(key) < 3 {
			c.logger.Errorf("processNextItem: Failed fetching item with key %s, error is %v, retrying...", key, err)
			c.queue.AddRateLimited(key)
		} else {
			c.logger.Errorf("processNextItem: Failed fetching item with key %s, error is %v, no more retries", key, err)
			c.queue.Forget(key)
			utilruntime.HandleError(err)
		}
	}

	if exists {
		c.logger.Infof("processNextItem: object created/updated detected: %s", keyRaw)
		c.handler.SetNodeGeolocation(item)
		c.queue.Forget(key)
	}
	return true
}
