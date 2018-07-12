package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/argoproj/argo/errors"
	unstructutil "github.com/argoproj/argo/util/unstructured"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/magaldima/macro/common"
	"github.com/magaldima/macro/pkg/apis/microservice"
	"github.com/magaldima/macro/pkg/apis/microservice/v1alpha1"
	microclientset "github.com/magaldima/macro/pkg/client/clientset/versioned"
)

// MacroController is the controller for microservice resources
type MacroController struct {
	// ConfigMap is the name of the config map in which to derive configuration of the controller from
	ConfigMap string
	// namespace for config map
	ConfigMapNS string
	// Config is the macro controller's configuration
	Config MacroControllerConfig

	// restConfig is used by controller to send a SIGUSR1 to the wait sidecar using remotecommand.NewSPDYExecutor().
	restConfig    *rest.Config
	kubeclientset kubernetes.Interface
	clientset     microclientset.Interface

	// datastructures to support the processing of microservices and microservice pods
	microInformer cache.SharedIndexInformer
	podInformer   cache.SharedIndexInformer
	microQueue    workqueue.RateLimitingInterface
	podQueue      workqueue.RateLimitingInterface
}

// MacroControllerConfig contain the configuration settings for the macro controller
type MacroControllerConfig struct {
	// Namespace is a label selector filter to limit the controller's watch to a specific namespace
	Namespace string `json:"namespace,omitempty"`

	// InstanceID is a label selector to limit the controller's watch to a specific instance. It
	// contains an arbitrary value that is carried forward into its pod labels, under the key
	// microservices.micro.mu/controller-instanceid, for the purposes of microservice segregation. This
	// enables a controller to only receive microservices and pod events that it is interested about,
	// in order to support multiple controllers in a single cluster, and ultimately allows the
	// controller itself to be bundled as part of a higher level application. If omitted, the
	// controller watches microservices and pods that *are not* labeled with an instance id.
	InstanceID string `json:"instanceID,omitempty"`

	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

const (
	macroResyncPeriod = 20 * time.Minute
	podResyncPeriod   = 30 * time.Minute
)

// NewMacroController instantiates a new MacroController
func NewMacroController(restConfig *rest.Config, kubeclientset kubernetes.Interface, clientset microclientset.Interface, configMap string) *MacroController {
	mc := MacroController{
		restConfig:    restConfig,
		kubeclientset: kubeclientset,
		clientset:     clientset,
		ConfigMap:     configMap,
		microQueue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podQueue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	return &mc
}

// Run starts a microservice resource controller
func (mc *MacroController) Run(ctx context.Context, microWorkers, podWorkers int) {
	defer mc.microQueue.ShutDown()
	defer mc.podQueue.ShutDown()

	log.Infof("Macro Controller starting")
	log.Info("Watch Macro controller config map updates")
	_, err := mc.watchControllerConfigMap(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for controller config map: %v", err)
		return
	}

	mc.microInformer = mc.newMicroserviceInformer()
	mc.podInformer = mc.newPodInformer()
	go mc.microInformer.Run(ctx.Done())
	go mc.podInformer.Run(ctx.Done())

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, informer := range []cache.SharedIndexInformer{mc.microInformer, mc.podInformer} {
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			log.Error("Timed out waiting for caches to sync")
			return
		}
	}

	for i := 0; i < microWorkers; i++ {
		go wait.Until(mc.runWorker, time.Second, ctx.Done())
	}
	for i := 0; i < podWorkers; i++ {
		go wait.Until(mc.podWorker, time.Second, ctx.Done())
	}
	<-ctx.Done()
}

func (mc *MacroController) runWorker() {
	for mc.processNextItem() {
	}
}

// processNextItem is the worker logic for handling workflow updates
func (mc *MacroController) processNextItem() bool {
	key, quit := mc.microQueue.Get()
	if quit {
		return false
	}
	defer mc.microQueue.Done(key)

	obj, exists, err := mc.microInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.Errorf("Failed to get microservice '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		// This happens after a microservice was labeled with completed=true
		// or was deleted, but the work queue still had an entry for it.
		return true
	}
	// The microservice informer receives unstructured objects to deal with the possibility of invalid
	// microservice manifests that are unable to unmarshal to microservice objects
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Warnf("Key '%s' in index is not an unstructured", key)
		return true
	}
	var micro v1alpha1.MicroService
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &micro)
	if err != nil {
		log.Warnf("Failed to unmarshal key '%s' to microservice object: %v", key, err)
		oc := newMicroServiceOperationCtx(&micro, mc)
		oc.markWorkflowFailed(fmt.Sprintf("invalid spec: %s", err.Error()))
		oc.persistUpdates()
		return true
	}

	if micro.ObjectMeta.Labels[common.LabelKeyComplete] == "true" {
		// can get here if we already added the completed=true label,
		// but we are still draining the controller's workflow workqueue
		return true
	}
	oc := newMicroServiceOperationCtx(&micro, mc)
	oc.operate()
	// TODO: operate should return error if it was unable to operate properly
	// so we can requeue the work for a later time
	// See: https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go
	//c.handleErr(err, key)
	return true
}

func (mc *MacroController) podWorker() {
	for mc.processNextPodItem() {
	}
}

// processNextPodItem is the worker logic for handling pod updates.
// For pods updates, this simply means to "wake up" the microservice by
// adding the corresponding microservice key into the microservice workqueue.
func (mc *MacroController) processNextPodItem() bool {
	key, quit := mc.podQueue.Get()
	if quit {
		return false
	}
	defer mc.podQueue.Done(key)

	obj, exists, err := mc.podInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.Errorf("Failed to get pod '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		// we can get here if pod was queued into the pod workqueue,
		// but it was either deleted or labeled completed by the time
		// we dequeued it.
		return true
	}
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		log.Warnf("Key '%s' in index is not a pod", key)
		return true
	}
	if pod.Labels == nil {
		log.Warnf("Pod '%s' did not have labels", key)
		return true
	}
	microName, ok := pod.Labels[common.LabelKeyMicroservice]
	if !ok {
		// Ignore pods unrelated to microservices (this shouldn't happen unless the watch is setup incorrectly)
		log.Warnf("watch returned pod unrelated to any microservice: %s", pod.ObjectMeta.Name)
		return true
	}
	// TODO: currently we reawaken the microservice on *any* pod updates.
	// But this could be be much improved to become smarter by only
	// requeue the microservice when there are changes that we care about.
	mc.microQueue.Add(pod.ObjectMeta.Namespace + "/" + microName)
	return true
}

// ResyncConfig reloads the controller config from the configmap
func (mc *MacroController) ResyncConfig() error {
	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		namespace = common.DefaultControllerNamespace
	}
	cmClient := mc.kubeclientset.CoreV1().ConfigMaps(namespace)
	cm, err := cmClient.Get(mc.ConfigMap, metav1.GetOptions{})
	if err != nil {
		return errors.InternalWrapError(err)
	}
	mc.ConfigMapNS = cm.Namespace
	return mc.updateConfig(cm)
}

func (mc *MacroController) updateConfig(cm *apiv1.ConfigMap) error {
	configStr, ok := cm.Data[common.MacroControllerConfigMapKey]
	if !ok {
		return errors.Errorf(errors.CodeBadRequest, "ConfigMap '%s' does not have key '%s'", mc.ConfigMap, common.MacroControllerConfigMapKey)
	}
	var config MacroControllerConfig
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Printf("macro controller configuration from %s:\n%s", mc.ConfigMap, configStr)
	mc.Config = config
	return nil
}

// instanceIDRequirement returns the label requirement to filter against a controller instance (or not)
func (mc *MacroController) instanceIDRequirement() labels.Requirement {
	var instanceIDReq *labels.Requirement
	var err error
	if mc.Config.InstanceID != "" {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeyMacroControllerInstanceID, selection.Equals, []string{mc.Config.InstanceID})
	} else {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeyMacroControllerInstanceID, selection.DoesNotExist, nil)
	}
	if err != nil {
		panic(err)
	}
	return *instanceIDReq
}

func (mc *MacroController) tweakMicroservicelist(options *metav1.ListOptions) {
	options.FieldSelector = fields.Everything().String()

	// completed notin (true)
	incompleteReq, err := labels.NewRequirement(common.LabelKeyComplete, selection.NotIn, []string{"true"})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(mc.instanceIDRequirement())
	options.LabelSelector = labelSelector.String()
}

// newMicroserviceInformer returns the workflow informer used by the controller. This is actually
// a custom built UnstructuredInformer which is in actuality returning unstructured.Unstructured
// objects. We no longer return WorkflowInformer due to: https://github.com/kubernetes/kubernetes/issues/57705
func (mc *MacroController) newMicroserviceInformer() cache.SharedIndexInformer {
	dynClientPool := dynamic.NewDynamicClientPool(mc.restConfig)
	dclient, err := dynClientPool.ClientForGroupVersionKind(v1alpha1.SchemaGroupVersionKind)
	if err != nil {
		panic(err)
	}
	resource := &metav1.APIResource{
		Name:         microservice.Plural,
		SingularName: microservice.Singular,
		Namespaced:   true,
		Group:        microservice.Group,
		Version:      "v1alpha1",
		ShortNames:   []string{"micro"},
	}
	informer := unstructutil.NewFilteredUnstructuredInformer(
		resource,
		dclient,
		mc.Config.Namespace,
		macroResyncPeriod,
		cache.Indexers{},
		mc.tweakMicroservicelist,
	)
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					mc.microQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					mc.microQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					mc.microQueue.Add(key)
				}
			},
		},
	)
	return informer
}

func (mc *MacroController) watchControllerConfigMap(ctx context.Context) (cache.Controller, error) {
	source := mc.newControllerConfigMapWatch()
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update. Updating the controller config.")
					err := mc.updateConfig(cm)
					if err != nil {
						log.Errorf("Update of config failed due to: %v", err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update. Updating the controller config.")
					err := mc.updateConfig(newCm)
					if err != nil {
						log.Errorf("Update of config failed due to: %v", err)
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (mc *MacroController) newControllerConfigMapWatch() *cache.ListWatch {
	c := mc.kubeclientset.CoreV1().RESTClient()
	resource := "configmaps"
	name := mc.ConfigMap
	namespace := mc.ConfigMapNS
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (mc *MacroController) newMicroPodWatch() *cache.ListWatch {
	c := mc.kubeclientset.CoreV1().RESTClient()
	resource := "pods"
	namespace := mc.Config.Namespace
	fieldSelector := fields.ParseSelectorOrDie("status.phase!=Pending")
	// completed=false
	incompleteReq, _ := labels.NewRequirement(common.LabelKeyComplete, selection.Equals, []string{"false"})
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(mc.instanceIDRequirement())

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (mc *MacroController) newPodInformer() cache.SharedIndexInformer {
	source := mc.newMicroPodWatch()
	informer := cache.NewSharedIndexInformer(source, &apiv1.Pod{}, podResyncPeriod, cache.Indexers{})
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					mc.podQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					mc.podQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					mc.podQueue.Add(key)
				}
			},
		},
	)
	return informer
}
