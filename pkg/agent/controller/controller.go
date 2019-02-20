package controller

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/silenceshell/yummy/pkg/agent/lvm"
	"github.com/silenceshell/yummy/pkg/constants"
	"github.com/silenceshell/yummy/pkg/utils"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"os"
	"strconv"
	"strings"
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
	clientset   *kubernetes.Clientset
	nodeName    string
	volumeGroup string
	mountDir    string
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer,
	informer cache.Controller, nodeName, volumeGroup, mountDir string, clientset *kubernetes.Clientset) *Controller {
	return &Controller{
		informer:    informer,
		indexer:     indexer,
		queue:       queue,
		nodeName:    nodeName,
		volumeGroup: volumeGroup,
		mountDir:    mountDir,
		clientset:   clientset,
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

func (c *Controller) handleAddAndUpdate(pvc *v1.PersistentVolumeClaim) error {
	var mountPoint string
	var node *v1.Node
	var vgSize, vgFree uint64

	if !c.isMyPvc(pvc) {
		return nil
	}

	// otherwise agent will create lv on this node
	lvName := pvc.Namespace + "-" + pvc.Name

	isExist := lvm.IsLvExist(c.volumeGroup, lvName)
	// if lvm exist, it must has been mounted.
	if isExist {
		return nil
	}

	// lv not exist, create lv now
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

	/*
		get info from /etc/mke2fs.conf
		[defaults]
		blocksize = 4096
		inode_size = 256
		inode_ratio = 16384
	*/
	//inodeCount := size / 16384
	//inodeSize := inodeCount * 256
	//size += inodeSize

	// reserved block count / block count = 0.95, and plus 4096 in case.
	// block count * 4096 is a little smaller than df() info, so it will met the pvc requirement.
	// totalSize := size * 100 / 95 + 4096
	size = size * 100 / 95

	glog.Infof("pvc size %v %v size %v", pvc.Spec.Size(), pvc.Size(), size)
	lv, err := lvm.CreateLv(c.volumeGroup, lvName, size)
	if err != nil {
		return err
	}

	//mkfs.ext4 /dev/volume-group1/lv1
	lvFullName := fmt.Sprintf("/dev/%s/%s", c.volumeGroup, lvName)
	err = utils.Run("mkfs.ext4", lvFullName)
	if err != nil {
		goto failed
	}

	//mkdir /lvm-mount
	mountPoint = fmt.Sprintf("%s/%s", c.mountDir, lvName)
	_, err = os.Stat(mountPoint)
	if os.IsNotExist(err) {
		glog.Infof("mount point not exist, create now")
		err = os.Mkdir(mountPoint, os.ModeDir)
		if err != nil {
			goto failed
		}
	}

	//mount /dev/volume-group1/lv1 /lvm-mount/
	err = utils.Run("mount", lvFullName, mountPoint)
	if err != nil {
		goto failed
	}

	// update node annotation
	vgSize, vgFree, err = lvm.GetVgInfo(c.volumeGroup)
	if err != nil {
		goto failed
	}
	node = utils.GetNode(c.clientset, c.nodeName)
	node.Annotations[constants.AnnotationVgSize] = strconv.FormatUint(vgSize, 10)
	node.Annotations[constants.AnnotationVgFreeSize] = strconv.FormatUint(vgFree, 10)

	_, err = c.clientset.CoreV1().Nodes().Update(node)
	if err != nil {
		goto failed
	}

	return nil
failed:
	e := lv.Remove()
	if e != nil {
		glog.Error("lv remove failed", e)
	}
	return err
}

func (c *Controller) isMyPvc(pvc *v1.PersistentVolumeClaim) bool {
	if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != constants.StorageClassLocalVolume {
		return false
	}

	annotations := pvc.GetAnnotations()
	nodeName, ok := annotations[constants.AnnotationNodeName]
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

//func (c *Controller) handleDelete(pvc *v1.PersistentVolumeClaim) error {
func (c *Controller) handleDelete(pvc string) error {
	//pvc in fmt {namespace}/{pvc name}

	//umount and rmdir
	sp := strings.Split(pvc, "/")
	lvName := sp[0] + "-" + sp[1]
	glog.Infof("delete lv %s", lvName)

	//requeue after

	mountPoint := fmt.Sprintf("%s/%s", c.mountDir, lvName)
	_, err := os.Stat(mountPoint)
	if err == nil {
		glog.Infof("mount point exist, umount and rm this dir")
		err = utils.Run("umount", mountPoint)
		if err != nil {
			glog.Error("umount failed")
			return err
		}

		//err = os.Remove(mountPoint)
		//if err != nil {
		//	return err
		//}

		lvm.DeleteLv(c.volumeGroup, lvName)
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
		if message.messageType == MessageDelete {
			err = c.handleDelete(key)
		}
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
			//err = c.handleDelete(pvc)
		default:
			glog.Error("Invalid Message Type")
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

	controller := NewController(queue, indexer, informer, nodeName, volumeGroup, mountDir, clientset)

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	// Wait forever
	select {}
}
