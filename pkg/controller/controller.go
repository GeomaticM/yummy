package controller

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/silenceshell/yummy/pkg/lvm"
	"github.com/silenceshell/yummy/pkg/utils"
	"os"

	//"k8s.io/apimachinery/pkg/util/rand"

	//"github.com/silenceshell/yummy/agent"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type Message struct {
	messageType string
	key         string
}

const (
	MessageAdd    = "Add"
	MessageUpdate = "Update"
	MessageDelete = "Delete"
)

type Controller struct {
	indexer     cache.Indexer
	queue       workqueue.RateLimitingInterface
	informer    cache.Controller
	nodeName    string
	volumeGroup string
	mountDir    string
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer,
	informer cache.Controller, nodeName, volumeGroup, mountDir string) *Controller {
	return &Controller{
		informer:    informer,
		indexer:     indexer,
		queue:       queue,
		nodeName:    nodeName,
		volumeGroup: volumeGroup,
		mountDir:    mountDir,
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	//key, quit := c.queue.Get()
	message, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pvcs with the same key are never processed in
	// parallel.
	defer c.queue.Done(message)

	// Invoke the method containing the business logic
	err := c.handle(message.(Message))
	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, message)
	return true
}

const (
	AnnotationKey = "yummyNodeName"
)

func (c *Controller) handleAddAndUpdate(pvc *v1.PersistentVolumeClaim) error {
	//annotations := pvc.GetAnnotations()
	//nodeName, ok := annotations[AnnotationKey]
	//if !ok {
	//	glog.Infof("pvc %s has no yummy annotation, wait for master to set", pvc.Name)
	//	return nil
	//}
	//
	//// if pvc is not scheduled to this node, agent will just ignore it.
	//if nodeName != c.nodeName {
	//	return nil
	//}
	if !c.isMyPvc(pvc) {
		return nil
	}

	// otherwise agent will create lv on this node
	lvName := pvc.Namespace + "-" + pvc.Name //+ "-" + rand.String(8)

	isExist, err := lvm.IsLvExist(c.volumeGroup, lvName)
	if err != nil {
		return err
	}
	// lv not exist, create lv now
	if !isExist {
		request, found := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		if !found {
			glog.Error("storage resource not specified")
			return fmt.Errorf("storage resource not specified")
		}
		size, ret := request.AsInt64()
		if !ret {
			glog.Error("storage resource request to int64 failed")
			return fmt.Errorf("storage resource request to int64 failed")
		}
		glog.Infof("pvc size %v %v %v", pvc.Spec.Size(), pvc.Size(), size)
		if err := lvm.CreateLv(c.volumeGroup, lvName, size); err != nil {
			return err
		}
	}

	//mkfs.ext4 /dev/volume-group1/lv1
	lvFullName := fmt.Sprintf("/dev/%s/%s", c.volumeGroup, lvName)
	err = utils.Run("mkfs.ext4", lvFullName)
	if err != nil {
		return err
	}

	//mkdir /lvm-mount
	//todo: check the mount point dir is exist or not. if true, emm..
	mountPoint := fmt.Sprintf("%s/%s", c.mountDir, lvName)
	err = os.Mkdir(mountPoint, os.ModeDir)
	if err != nil {
		return err
	}

	//mount /dev/volume-group1/lv1 /lvm-mount/
	err = utils.Run("mount", lvFullName, mountPoint)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) isMyPvc(pvc *v1.PersistentVolumeClaim) bool {
	annotations := pvc.GetAnnotations()
	nodeName, ok := annotations[AnnotationKey]
	if !ok {
		glog.Infof("pvc %s has no yummy annotation, wait for master to set", pvc.Name)
		return false
	}

	// if pvc is not scheduled to this node, agent will just ignore it.
	if nodeName != c.nodeName {
		return false
	}

	return true
}

func (c *Controller) handleDelete(pvc *v1.PersistentVolumeClaim) error {
	if !c.isMyPvc(pvc) {
		return nil
	}

	// otherwise agent will create lv on this node
	lvName := pvc.Namespace + "-" + pvc.Name //+ "-" + rand.String(8)
	isExist, err := lvm.IsLvExist(c.volumeGroup, lvName)
	if err != nil {
		return err
	}
	if !isExist {
		return fmt.Errorf("lv %s not found in gv %s", lvName, c.volumeGroup)
	}



	return nil
}

// handle is the business logic of the controller. In this controller it simply prints
// information about the pvc to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) handle(message Message) error {
	key := message.key
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Pvc, so that we will see a delete for one pvc
		glog.Infof("Pvc %s does not exist anymore\n", key)
	} else {
		// Note that you also have to check the uid if you have a local controlled resource, which
		// is dependent on the actual instance, to detect that a Pvc was recreated with the same name
		pvc := obj.(*v1.PersistentVolumeClaim)
		glog.Infof("%s Pvc %s\n", message.messageType, pvc.GetName())

		switch message.messageType {
		case MessageAdd:
			glog.Infof("PVC Create, %v", pvc)
			err = c.handleAddAndUpdate(pvc)
		case MessageUpdate:
			glog.Infof("PVC Update, %v", pvc)
			err = c.handleAddAndUpdate(pvc)
		case MessageDelete:
			glog.Infof("PVC Delete, %v", pvc)
		default:
			glog.Error("Invalid Message Type")
			err = c.handleDelete(pvc)
		}
	}
	return err
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, message interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the message on every successful synchronization.
		// This ensures that future processing of updates for this message is not delayed because of
		// an outdated error history.
		c.queue.Forget(message)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(message) < 5 {
		glog.Infof("Error syncing pvc %v: %v", message, err)

		// Re-enqueue the message rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the message will be processed later again.
		c.queue.AddRateLimited(message)
		return
	}

	c.queue.Forget(message)
	// Report to an external entity that, even after several retries, we could not successfully process this message
	runtime.HandleError(err)
	glog.Infof("Dropping pvc %q out of the queue: %v", message, err)
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	glog.Info("Starting Pvc controller")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Stopping Pvc controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func StartController(clientset *kubernetes.Clientset, nodeName, volumeGroup, mountDir string) {
	// create the pvc watcher
	pvcListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(),
		"persistentvolumeclaims", v1.NamespaceAll, fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pvc key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Pvc than the version which was responsible for triggering the update.
	indexer, informer := cache.NewIndexerInformer(pvcListWatcher, &v1.PersistentVolumeClaim{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(Message{messageType: MessageAdd, key: key})
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(Message{messageType: MessageUpdate, key: key})
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(Message{messageType: MessageDelete, key: key})
			}
		},
	}, cache.Indexers{})

	controller := NewController(queue, indexer, informer, nodeName, volumeGroup, mountDir)

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	// Wait forever
	select {}
}
