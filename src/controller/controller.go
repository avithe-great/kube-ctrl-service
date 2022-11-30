package controller

import (
	"fmt"
	"time"

	"github.com/avithe-great/kube-ctrl-service/src/kube"
	"github.com/sirupsen/logrus"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type event struct {
	key          string
	eventType    string
	resourceType string
}

type controller struct {
	client   kubernetes.Interface
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
}

func Start(config *string) {
	kc, err := kube.GetClient(config)
	if err != nil {
		logrus.Fatal(err)
	}

	factory := informers.NewSharedInformerFactory(kc, 0)
	informer := factory.Core().V1().Pods().Informer()

	c := newController(kc, informer)
	stopCh := make(chan struct{})
	defer close(stopCh)

	c.Run(stopCh)

}

func newController(kc kubernetes.Interface, informer cache.SharedIndexInformer) *controller {
	q := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var event event
	var err error
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event.key, err = cache.MetaNamespaceKeyFunc(obj)
			event.eventType = "create"
			if err == nil {
				q.Add(event)
			}
			logrus.Infof("Event received of type [%s] for [%s]", event.eventType, event.key)
		},
		UpdateFunc: func(old, new interface{}) {
			event.key, err = cache.MetaNamespaceKeyFunc(old)
			event.eventType = "update"
			if err == nil {
				q.Add(event)
			}
			logrus.Infof("Event received of type [%s] for [%s]", event.eventType, event.key)
		},
		DeleteFunc: func(obj interface{}) {
			event.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			event.eventType = "delete"
			if err == nil {
				q.Add(event)
			}
			logrus.Infof("Event received of type [%s] for [%s]", event.eventType, event.key)
		},
	})

	return &controller{
		client:   kc,
		informer: informer,
		queue:    q,
	}
}

func (c *controller) Run(stopper <-chan struct{}) {

	defer utilruntime.HandleCrash() //this will handle panic and won't crash the process
	defer c.queue.ShutDown()        //shutdown all workqueue and terminate all workers

	logrus.Info("Starting Chronos...")

	go c.informer.Run(stopper)

	logrus.Info("Synchronizing events...")

	//synchronize the cache before starting to process events
	if !cache.WaitForCacheSync(stopper, c.informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		logrus.Info("synchronization failed...")
		return
	}

	logrus.Info("synchronization complete!")
	logrus.Info("Ready to process events")

	wait.Until(c.runWorker, time.Second, stopper)
}

func (c *controller) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

func (c *controller) processNextItem() bool {
	e, term := c.queue.Get()

	if term {
		return false
	}

	err := c.processItem(e.(event))
	if err == nil {
		c.queue.Forget(e)
		return true
	}
	return true
}

var abc []interface{}

func (c *controller) processItem(e event) error {
	obj, _, err := c.informer.GetIndexer().GetByKey(e.key)
	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store: %v", e.key, err)
	}

	//Use a switch clause instead and process the events based on the type
	//logrus.Infof("Chronos has processed 1 event of type [%s] for object [%s]", e.eventType, obj)
	switch e.eventType {
	case "create":
		mObj := obj.(v1.Object)
		logrus.Printf("New Pod Added to Store: %s", mObj.GetName())
		logrus.Printf("New Added Pod belongs to Namespaces: %s", mObj.GetNamespace())
		abc = append(abc, mObj.GetName())
		logrus.Print("abc lenght", len(abc))
	case "update":
	case "delete":
	default:
		logrus.Print("Invalid event type")
	}

	return nil
}
