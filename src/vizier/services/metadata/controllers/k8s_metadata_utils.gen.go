// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package controllers

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	internalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchClient "k8s.io/client-go/tools/watch"

	// Blank import necessary for kubeConfig to work.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	protoutils "pixielabs.ai/pixielabs/src/shared/k8s"
	storepb "pixielabs.ai/pixielabs/src/vizier/services/metadata/storepb"
)

func runtimeObjToPodList(o runtime.Object) *v1.PodList {
	l, ok := o.(*v1.PodList)
	if ok {
		return l
	}

	internalList, ok := o.(*internalversion.List)
	if !ok {
		log.WithField("list", o).Fatal("Received unexpected object type from lister")
	}

	typedList := v1.PodList{}
	for _, i := range internalList.Items {
		item, ok := i.(*v1.Pod)
		if !ok {
			log.WithField("expectedPod", i).Fatal("Received unexpected object type from lister")
		}
		typedList.Items = append(typedList.Items, *item)
	}

	return &typedList
}

// PodWatcher is a resource watcher for Pods.
type PodWatcher struct {
	resourceStr string
	lastRV      string
	updateCh    chan *K8sResourceMessage
	clientset   *kubernetes.Clientset
}

// NewPodWatcher creates a resource watcher for Pods.
func NewPodWatcher(resource string, updateCh chan *K8sResourceMessage, clientset *kubernetes.Clientset) *PodWatcher {
	return &PodWatcher{resourceStr: resource, updateCh: updateCh, clientset: clientset}
}

// Sync syncs the watcher state with the stored updates for Pods.
func (mc *PodWatcher) Sync(storedUpdates []*storepb.K8SResource) error {
	resources, err := listObject(mc.resourceStr, mc.clientset)
	if err != nil {
		return err
	}
	mc.syncPodImpl(storedUpdates, runtimeObjToPodList(resources))
	return nil
}

func (mc *PodWatcher) syncPodImpl(storedUpdates []*storepb.K8SResource, currentState *v1.PodList) {
	activeResources := make(map[string]bool)

	// Send update for currently active resources. This will build our HostIP/PodIP maps.
	for _, o := range currentState.Items {
		pb, err := protoutils.PodToProto(&o)
		if err != nil {
			continue
		}
		activeResources[string(o.ObjectMeta.UID)] = true

		r := &storepb.K8SResource{
			Resource: &storepb.K8SResource_Pod{
				Pod: pb,
			},
		}
		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Modified,
		}
		mc.updateCh <- msg
	}
	mc.lastRV = currentState.ResourceVersion

	// Make a map of resources that we have stored.
	storedResources := make(map[string]*storepb.K8SResource)
	for i := range storedUpdates {
		update := storedUpdates[i].GetPod()
		if update == nil {
			continue
		}

		if update.Metadata.DeletionTimestampNS != 0 {
			// This resource is already terminated, so we don't
			// need to send another termination event.
			delete(storedResources, update.Metadata.UID)
			continue
		}

		storedResources[update.Metadata.UID] = storedUpdates[i]
	}

	// For each resource in our store, determine if it is still running. If not, send a termination update.
	for uid, r := range storedResources {
		if _, ok := activeResources[uid]; ok {
			continue
		}

		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Deleted,
		}
		mc.updateCh <- msg
	}
}

// StartWatcher starts a watcher for Pods.
func (mc *PodWatcher) StartWatcher(quitCh chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		watcher := cache.NewListWatchFromClient(mc.clientset.CoreV1().RESTClient(), mc.resourceStr, v1.NamespaceAll, fields.Everything())
		retryWatcher, err := watchClient.NewRetryWatcher(mc.lastRV, watcher)
		if err != nil {
			log.WithError(err).Fatal("Could not start watcher for k8s resource: " + mc.resourceStr)
		}

		resCh := retryWatcher.ResultChan()
		runWatcher := true
		for runWatcher {
			select {
			case <-quitCh:
				return
			case c := <-resCh:
				s, ok := c.Object.(*metav1.Status)
				if ok && s.Status == metav1.StatusFailure {
					if s.Reason == metav1.StatusReasonGone {
						log.WithField("resource", mc.resourceStr).Info("Requested resource version too old, no longer stored in K8S API")
						runWatcher = false
						break
					}
					// Ignore and let the retry watcher retry.
					log.WithField("resource", mc.resourceStr).WithField("object", c.Object).Info("Failed to read from k8s watcher")
					continue
				}

				// Update the lastRV, so that if the watcher restarts, it starts at the correct resource version.
				o, ok := c.Object.(*v1.Pod)
				if !ok {
					continue
				}

				mc.lastRV = o.ObjectMeta.ResourceVersion

				pb, err := protoutils.PodToProto(o)
				if err != nil {
					continue
				}
				r := &storepb.K8SResource{
					Resource: &storepb.K8SResource_Pod{
						Pod: pb,
					},
				}

				msg := &K8sResourceMessage{
					Object:     r,
					ObjectType: mc.resourceStr,
					EventType:  c.Type,
				}
				mc.updateCh <- msg
			}
		}

		log.WithField("resource", mc.resourceStr).Info("K8s watcher channel closed. Retrying")
		runWatcher = true

		// Wait 5 minutes before retrying, however if stop is called, just return.
		select {
		case <-quitCh:
			return
		case <-time.After(5 * time.Minute):
			continue
		}
	}
}

func runtimeObjToServiceList(o runtime.Object) *v1.ServiceList {
	l, ok := o.(*v1.ServiceList)
	if ok {
		return l
	}

	internalList, ok := o.(*internalversion.List)
	if !ok {
		log.WithField("list", o).Fatal("Received unexpected object type from lister")
	}

	typedList := v1.ServiceList{}
	for _, i := range internalList.Items {
		item, ok := i.(*v1.Service)
		if !ok {
			log.WithField("expectedService", i).Fatal("Received unexpected object type from lister")
		}
		typedList.Items = append(typedList.Items, *item)
	}

	return &typedList
}

// ServiceWatcher is a resource watcher for Services.
type ServiceWatcher struct {
	resourceStr string
	lastRV      string
	updateCh    chan *K8sResourceMessage
	clientset   *kubernetes.Clientset
}

// NewServiceWatcher creates a resource watcher for Services.
func NewServiceWatcher(resource string, updateCh chan *K8sResourceMessage, clientset *kubernetes.Clientset) *ServiceWatcher {
	return &ServiceWatcher{resourceStr: resource, updateCh: updateCh, clientset: clientset}
}

// Sync syncs the watcher state with the stored updates for Services.
func (mc *ServiceWatcher) Sync(storedUpdates []*storepb.K8SResource) error {
	resources, err := listObject(mc.resourceStr, mc.clientset)
	if err != nil {
		return err
	}
	mc.syncServiceImpl(storedUpdates, runtimeObjToServiceList(resources))
	return nil
}

func (mc *ServiceWatcher) syncServiceImpl(storedUpdates []*storepb.K8SResource, currentState *v1.ServiceList) {
	activeResources := make(map[string]bool)

	// Send update for currently active resources. This will build our HostIP/PodIP maps.
	for _, o := range currentState.Items {
		pb, err := protoutils.ServiceToProto(&o)
		if err != nil {
			continue
		}
		activeResources[string(o.ObjectMeta.UID)] = true

		r := &storepb.K8SResource{
			Resource: &storepb.K8SResource_Service{
				Service: pb,
			},
		}
		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Modified,
		}
		mc.updateCh <- msg
	}
	mc.lastRV = currentState.ResourceVersion

	// Make a map of resources that we have stored.
	storedResources := make(map[string]*storepb.K8SResource)
	for i := range storedUpdates {
		update := storedUpdates[i].GetService()
		if update == nil {
			continue
		}

		if update.Metadata.DeletionTimestampNS != 0 {
			// This resource is already terminated, so we don't
			// need to send another termination event.
			delete(storedResources, update.Metadata.UID)
			continue
		}

		storedResources[update.Metadata.UID] = storedUpdates[i]
	}

	// For each resource in our store, determine if it is still running. If not, send a termination update.
	for uid, r := range storedResources {
		if _, ok := activeResources[uid]; ok {
			continue
		}

		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Deleted,
		}
		mc.updateCh <- msg
	}
}

// StartWatcher starts a watcher for Services.
func (mc *ServiceWatcher) StartWatcher(quitCh chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		watcher := cache.NewListWatchFromClient(mc.clientset.CoreV1().RESTClient(), mc.resourceStr, v1.NamespaceAll, fields.Everything())
		retryWatcher, err := watchClient.NewRetryWatcher(mc.lastRV, watcher)
		if err != nil {
			log.WithError(err).Fatal("Could not start watcher for k8s resource: " + mc.resourceStr)
		}

		resCh := retryWatcher.ResultChan()
		runWatcher := true
		for runWatcher {
			select {
			case <-quitCh:
				return
			case c := <-resCh:
				s, ok := c.Object.(*metav1.Status)
				if ok && s.Status == metav1.StatusFailure {
					if s.Reason == metav1.StatusReasonGone {
						log.WithField("resource", mc.resourceStr).Info("Requested resource version too old, no longer stored in K8S API")
						runWatcher = false
						break
					}
					// Ignore and let the retry watcher retry.
					log.WithField("resource", mc.resourceStr).WithField("object", c.Object).Info("Failed to read from k8s watcher")
					continue
				}

				// Update the lastRV, so that if the watcher restarts, it starts at the correct resource version.
				o, ok := c.Object.(*v1.Service)
				if !ok {
					continue
				}

				mc.lastRV = o.ObjectMeta.ResourceVersion

				pb, err := protoutils.ServiceToProto(o)
				if err != nil {
					continue
				}
				r := &storepb.K8SResource{
					Resource: &storepb.K8SResource_Service{
						Service: pb,
					},
				}

				msg := &K8sResourceMessage{
					Object:     r,
					ObjectType: mc.resourceStr,
					EventType:  c.Type,
				}
				mc.updateCh <- msg
			}
		}

		log.WithField("resource", mc.resourceStr).Info("K8s watcher channel closed. Retrying")
		runWatcher = true

		// Wait 5 minutes before retrying, however if stop is called, just return.
		select {
		case <-quitCh:
			return
		case <-time.After(5 * time.Minute):
			continue
		}
	}
}

func runtimeObjToNamespaceList(o runtime.Object) *v1.NamespaceList {
	l, ok := o.(*v1.NamespaceList)
	if ok {
		return l
	}

	internalList, ok := o.(*internalversion.List)
	if !ok {
		log.WithField("list", o).Fatal("Received unexpected object type from lister")
	}

	typedList := v1.NamespaceList{}
	for _, i := range internalList.Items {
		item, ok := i.(*v1.Namespace)
		if !ok {
			log.WithField("expectedNamespace", i).Fatal("Received unexpected object type from lister")
		}
		typedList.Items = append(typedList.Items, *item)
	}

	return &typedList
}

// NamespaceWatcher is a resource watcher for Namespaces.
type NamespaceWatcher struct {
	resourceStr string
	lastRV      string
	updateCh    chan *K8sResourceMessage
	clientset   *kubernetes.Clientset
}

// NewNamespaceWatcher creates a resource watcher for Namespaces.
func NewNamespaceWatcher(resource string, updateCh chan *K8sResourceMessage, clientset *kubernetes.Clientset) *NamespaceWatcher {
	return &NamespaceWatcher{resourceStr: resource, updateCh: updateCh, clientset: clientset}
}

// Sync syncs the watcher state with the stored updates for Namespaces.
func (mc *NamespaceWatcher) Sync(storedUpdates []*storepb.K8SResource) error {
	resources, err := listObject(mc.resourceStr, mc.clientset)
	if err != nil {
		return err
	}
	mc.syncNamespaceImpl(storedUpdates, runtimeObjToNamespaceList(resources))
	return nil
}

func (mc *NamespaceWatcher) syncNamespaceImpl(storedUpdates []*storepb.K8SResource, currentState *v1.NamespaceList) {
	activeResources := make(map[string]bool)

	// Send update for currently active resources. This will build our HostIP/PodIP maps.
	for _, o := range currentState.Items {
		pb, err := protoutils.NamespaceToProto(&o)
		if err != nil {
			continue
		}
		activeResources[string(o.ObjectMeta.UID)] = true

		r := &storepb.K8SResource{
			Resource: &storepb.K8SResource_Namespace{
				Namespace: pb,
			},
		}
		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Modified,
		}
		mc.updateCh <- msg
	}
	mc.lastRV = currentState.ResourceVersion

	// Make a map of resources that we have stored.
	storedResources := make(map[string]*storepb.K8SResource)
	for i := range storedUpdates {
		update := storedUpdates[i].GetNamespace()
		if update == nil {
			continue
		}

		if update.Metadata.DeletionTimestampNS != 0 {
			// This resource is already terminated, so we don't
			// need to send another termination event.
			delete(storedResources, update.Metadata.UID)
			continue
		}

		storedResources[update.Metadata.UID] = storedUpdates[i]
	}

	// For each resource in our store, determine if it is still running. If not, send a termination update.
	for uid, r := range storedResources {
		if _, ok := activeResources[uid]; ok {
			continue
		}

		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Deleted,
		}
		mc.updateCh <- msg
	}
}

// StartWatcher starts a watcher for Namespaces.
func (mc *NamespaceWatcher) StartWatcher(quitCh chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		watcher := cache.NewListWatchFromClient(mc.clientset.CoreV1().RESTClient(), mc.resourceStr, v1.NamespaceAll, fields.Everything())
		retryWatcher, err := watchClient.NewRetryWatcher(mc.lastRV, watcher)
		if err != nil {
			log.WithError(err).Fatal("Could not start watcher for k8s resource: " + mc.resourceStr)
		}

		resCh := retryWatcher.ResultChan()
		runWatcher := true
		for runWatcher {
			select {
			case <-quitCh:
				return
			case c := <-resCh:
				s, ok := c.Object.(*metav1.Status)
				if ok && s.Status == metav1.StatusFailure {
					if s.Reason == metav1.StatusReasonGone {
						log.WithField("resource", mc.resourceStr).Info("Requested resource version too old, no longer stored in K8S API")
						runWatcher = false
						break
					}
					// Ignore and let the retry watcher retry.
					log.WithField("resource", mc.resourceStr).WithField("object", c.Object).Info("Failed to read from k8s watcher")
					continue
				}

				// Update the lastRV, so that if the watcher restarts, it starts at the correct resource version.
				o, ok := c.Object.(*v1.Namespace)
				if !ok {
					continue
				}

				mc.lastRV = o.ObjectMeta.ResourceVersion

				pb, err := protoutils.NamespaceToProto(o)
				if err != nil {
					continue
				}
				r := &storepb.K8SResource{
					Resource: &storepb.K8SResource_Namespace{
						Namespace: pb,
					},
				}

				msg := &K8sResourceMessage{
					Object:     r,
					ObjectType: mc.resourceStr,
					EventType:  c.Type,
				}
				mc.updateCh <- msg
			}
		}

		log.WithField("resource", mc.resourceStr).Info("K8s watcher channel closed. Retrying")
		runWatcher = true

		// Wait 5 minutes before retrying, however if stop is called, just return.
		select {
		case <-quitCh:
			return
		case <-time.After(5 * time.Minute):
			continue
		}
	}
}

func runtimeObjToEndpointsList(o runtime.Object) *v1.EndpointsList {
	l, ok := o.(*v1.EndpointsList)
	if ok {
		return l
	}

	internalList, ok := o.(*internalversion.List)
	if !ok {
		log.WithField("list", o).Fatal("Received unexpected object type from lister")
	}

	typedList := v1.EndpointsList{}
	for _, i := range internalList.Items {
		item, ok := i.(*v1.Endpoints)
		if !ok {
			log.WithField("expectedEndpoints", i).Fatal("Received unexpected object type from lister")
		}
		typedList.Items = append(typedList.Items, *item)
	}

	return &typedList
}

// EndpointsWatcher is a resource watcher for Endpointss.
type EndpointsWatcher struct {
	resourceStr string
	lastRV      string
	updateCh    chan *K8sResourceMessage
	clientset   *kubernetes.Clientset
}

// NewEndpointsWatcher creates a resource watcher for Endpointss.
func NewEndpointsWatcher(resource string, updateCh chan *K8sResourceMessage, clientset *kubernetes.Clientset) *EndpointsWatcher {
	return &EndpointsWatcher{resourceStr: resource, updateCh: updateCh, clientset: clientset}
}

// Sync syncs the watcher state with the stored updates for Endpointss.
func (mc *EndpointsWatcher) Sync(storedUpdates []*storepb.K8SResource) error {
	resources, err := listObject(mc.resourceStr, mc.clientset)
	if err != nil {
		return err
	}
	mc.syncEndpointsImpl(storedUpdates, runtimeObjToEndpointsList(resources))
	return nil
}

func (mc *EndpointsWatcher) syncEndpointsImpl(storedUpdates []*storepb.K8SResource, currentState *v1.EndpointsList) {
	activeResources := make(map[string]bool)

	// Send update for currently active resources. This will build our HostIP/PodIP maps.
	for _, o := range currentState.Items {
		pb, err := protoutils.EndpointsToProto(&o)
		if err != nil {
			continue
		}
		activeResources[string(o.ObjectMeta.UID)] = true

		r := &storepb.K8SResource{
			Resource: &storepb.K8SResource_Endpoints{
				Endpoints: pb,
			},
		}
		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Modified,
		}
		mc.updateCh <- msg
	}
	mc.lastRV = currentState.ResourceVersion

	// Make a map of resources that we have stored.
	storedResources := make(map[string]*storepb.K8SResource)
	for i := range storedUpdates {
		update := storedUpdates[i].GetEndpoints()
		if update == nil {
			continue
		}

		if update.Metadata.DeletionTimestampNS != 0 {
			// This resource is already terminated, so we don't
			// need to send another termination event.
			delete(storedResources, update.Metadata.UID)
			continue
		}

		storedResources[update.Metadata.UID] = storedUpdates[i]
	}

	// For each resource in our store, determine if it is still running. If not, send a termination update.
	for uid, r := range storedResources {
		if _, ok := activeResources[uid]; ok {
			continue
		}

		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Deleted,
		}
		mc.updateCh <- msg
	}
}

// StartWatcher starts a watcher for Endpointss.
func (mc *EndpointsWatcher) StartWatcher(quitCh chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		watcher := cache.NewListWatchFromClient(mc.clientset.CoreV1().RESTClient(), mc.resourceStr, v1.NamespaceAll, fields.Everything())
		retryWatcher, err := watchClient.NewRetryWatcher(mc.lastRV, watcher)
		if err != nil {
			log.WithError(err).Fatal("Could not start watcher for k8s resource: " + mc.resourceStr)
		}

		resCh := retryWatcher.ResultChan()
		runWatcher := true
		for runWatcher {
			select {
			case <-quitCh:
				return
			case c := <-resCh:
				s, ok := c.Object.(*metav1.Status)
				if ok && s.Status == metav1.StatusFailure {
					if s.Reason == metav1.StatusReasonGone {
						log.WithField("resource", mc.resourceStr).Info("Requested resource version too old, no longer stored in K8S API")
						runWatcher = false
						break
					}
					// Ignore and let the retry watcher retry.
					log.WithField("resource", mc.resourceStr).WithField("object", c.Object).Info("Failed to read from k8s watcher")
					continue
				}

				// Update the lastRV, so that if the watcher restarts, it starts at the correct resource version.
				o, ok := c.Object.(*v1.Endpoints)
				if !ok {
					continue
				}

				mc.lastRV = o.ObjectMeta.ResourceVersion

				pb, err := protoutils.EndpointsToProto(o)
				if err != nil {
					continue
				}
				r := &storepb.K8SResource{
					Resource: &storepb.K8SResource_Endpoints{
						Endpoints: pb,
					},
				}

				msg := &K8sResourceMessage{
					Object:     r,
					ObjectType: mc.resourceStr,
					EventType:  c.Type,
				}
				mc.updateCh <- msg
			}
		}

		log.WithField("resource", mc.resourceStr).Info("K8s watcher channel closed. Retrying")
		runWatcher = true

		// Wait 5 minutes before retrying, however if stop is called, just return.
		select {
		case <-quitCh:
			return
		case <-time.After(5 * time.Minute):
			continue
		}
	}
}

func runtimeObjToNodeList(o runtime.Object) *v1.NodeList {
	l, ok := o.(*v1.NodeList)
	if ok {
		return l
	}

	internalList, ok := o.(*internalversion.List)
	if !ok {
		log.WithField("list", o).Fatal("Received unexpected object type from lister")
	}

	typedList := v1.NodeList{}
	for _, i := range internalList.Items {
		item, ok := i.(*v1.Node)
		if !ok {
			log.WithField("expectedNode", i).Fatal("Received unexpected object type from lister")
		}
		typedList.Items = append(typedList.Items, *item)
	}

	return &typedList
}

// NodeWatcher is a resource watcher for Nodes.
type NodeWatcher struct {
	resourceStr string
	lastRV      string
	updateCh    chan *K8sResourceMessage
	clientset   *kubernetes.Clientset
}

// NewNodeWatcher creates a resource watcher for Nodes.
func NewNodeWatcher(resource string, updateCh chan *K8sResourceMessage, clientset *kubernetes.Clientset) *NodeWatcher {
	return &NodeWatcher{resourceStr: resource, updateCh: updateCh, clientset: clientset}
}

// Sync syncs the watcher state with the stored updates for Nodes.
func (mc *NodeWatcher) Sync(storedUpdates []*storepb.K8SResource) error {
	resources, err := listObject(mc.resourceStr, mc.clientset)
	if err != nil {
		return err
	}
	mc.syncNodeImpl(storedUpdates, runtimeObjToNodeList(resources))
	return nil
}

func (mc *NodeWatcher) syncNodeImpl(storedUpdates []*storepb.K8SResource, currentState *v1.NodeList) {
	activeResources := make(map[string]bool)

	// Send update for currently active resources. This will build our HostIP/PodIP maps.
	for _, o := range currentState.Items {
		pb, err := protoutils.NodeToProto(&o)
		if err != nil {
			continue
		}
		activeResources[string(o.ObjectMeta.UID)] = true

		r := &storepb.K8SResource{
			Resource: &storepb.K8SResource_Node{
				Node: pb,
			},
		}
		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Modified,
		}
		mc.updateCh <- msg
	}
	mc.lastRV = currentState.ResourceVersion

	// Make a map of resources that we have stored.
	storedResources := make(map[string]*storepb.K8SResource)
	for i := range storedUpdates {
		update := storedUpdates[i].GetNode()
		if update == nil {
			continue
		}

		if update.Metadata.DeletionTimestampNS != 0 {
			// This resource is already terminated, so we don't
			// need to send another termination event.
			delete(storedResources, update.Metadata.UID)
			continue
		}

		storedResources[update.Metadata.UID] = storedUpdates[i]
	}

	// For each resource in our store, determine if it is still running. If not, send a termination update.
	for uid, r := range storedResources {
		if _, ok := activeResources[uid]; ok {
			continue
		}

		msg := &K8sResourceMessage{
			Object:     r,
			ObjectType: mc.resourceStr,
			EventType:  watch.Deleted,
		}
		mc.updateCh <- msg
	}
}

// StartWatcher starts a watcher for Nodes.
func (mc *NodeWatcher) StartWatcher(quitCh chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		watcher := cache.NewListWatchFromClient(mc.clientset.CoreV1().RESTClient(), mc.resourceStr, v1.NamespaceAll, fields.Everything())
		retryWatcher, err := watchClient.NewRetryWatcher(mc.lastRV, watcher)
		if err != nil {
			log.WithError(err).Fatal("Could not start watcher for k8s resource: " + mc.resourceStr)
		}

		resCh := retryWatcher.ResultChan()
		runWatcher := true
		for runWatcher {
			select {
			case <-quitCh:
				return
			case c := <-resCh:
				s, ok := c.Object.(*metav1.Status)
				if ok && s.Status == metav1.StatusFailure {
					if s.Reason == metav1.StatusReasonGone {
						log.WithField("resource", mc.resourceStr).Info("Requested resource version too old, no longer stored in K8S API")
						runWatcher = false
						break
					}
					// Ignore and let the retry watcher retry.
					log.WithField("resource", mc.resourceStr).WithField("object", c.Object).Info("Failed to read from k8s watcher")
					continue
				}

				// Update the lastRV, so that if the watcher restarts, it starts at the correct resource version.
				o, ok := c.Object.(*v1.Node)
				if !ok {
					continue
				}

				mc.lastRV = o.ObjectMeta.ResourceVersion

				pb, err := protoutils.NodeToProto(o)
				if err != nil {
					continue
				}
				r := &storepb.K8SResource{
					Resource: &storepb.K8SResource_Node{
						Node: pb,
					},
				}

				msg := &K8sResourceMessage{
					Object:     r,
					ObjectType: mc.resourceStr,
					EventType:  c.Type,
				}
				mc.updateCh <- msg
			}
		}

		log.WithField("resource", mc.resourceStr).Info("K8s watcher channel closed. Retrying")
		runWatcher = true

		// Wait 5 minutes before retrying, however if stop is called, just return.
		select {
		case <-quitCh:
			return
		case <-time.After(5 * time.Minute):
			continue
		}
	}
}