package controller

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/magaldima/macro/common"
	"github.com/magaldima/macro/pkg/apis/microservice/v1alpha1"
	clientset "github.com/magaldima/macro/pkg/client/clientset/versioned/typed/microservice/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type operationCtx struct {
	// svc is the microservice object
	obj *v1alpha1.MicroService

	// orig is the original microservice object for purposes of creating a patch
	orig *v1alpha1.MicroService

	// updated indicates whether or not the microservice object itself was updated
	// and needs to be persisted back to kubernetes
	updated bool

	// log is an logrus logging context to corralate logs with a microservice
	log *log.Entry

	// controller reference to workflow controller
	controller *MacroController
}

func newMicroServiceOperationCtx(obj *v1alpha1.MicroService, mc *MacroController) *operationCtx {
	oc := operationCtx{
		obj:     obj.DeepCopy(),
		orig:    obj,
		updated: false,
		log: log.WithFields(log.Fields{
			"svc":       obj.Name,
			"namespace": obj.Namespace,
		}),
		controller: mc,
	}

	if oc.obj.Status.Nodes == nil {
		oc.obj.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
	}

	return &oc
}

// operate is the main operator logic of a microservice.
// TODO: an error returned by this method should result in requeuing the microservice to be retried at a
// later time
func (oc *operationCtx) operate() {
	defer oc.persistUpdates()
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(error); ok {
				woc.markWorkflowError(rerr, true)
			} else {
				woc.markWorkflowPhase(wfv1.NodeError, true, fmt.Sprintf("%v", r))
			}
			woc.log.Errorf("Recovered from panic: %+v\n%s", r, debug.Stack())
		}
	}()
	oc.log.Infof("processing microservice")
}

// persistUpdates will update a microservice with any updates made during operation.
// It also labels any pods as completed if we have extracted everything we need from it.
func (oc *operationCtx) persistUpdates() {
	if !oc.updated {
		return
	}
	microClient := oc.controller.clientset.MicroV1alpha1().MicroServices(oc.obj.Namespace)
	_, err := microClient.Update(oc.obj)
	if err != nil {
		oc.log.Warnf("Error updating object: %v", err)
		if !errors.IsConflict(err) {
			return
		}
		oc.log.Info("Re-appying updates on latest version and retrying update")
		err = oc.reapplyUpdate(microClient)
		if err != nil {
			oc.log.Infof("Failed to re-apply update: %+v", err)
			return
		}
	}
	oc.log.Info("object update successful")

	// HACK(jessesuen) after we successfully persist an update to the workflow, the informer's
	// cache is now invalid. It's very common that we will need to immediately re-operate on a
	// workflow due to queuing by the pod workers. The following sleep gives a *chance* for the
	// informer's cache to catch up to the version of the workflow we just persisted. Without
	// this sleep, the next worker to work on this workflow will very likely operate on a stale
	// object and redo work.
	time.Sleep(1 * time.Second)
}

// reapplyUpdate GETs the latest version of the microservice, re-applies the updates and
// retries the UPDATE multiple times. For reasoning behind this technique, see:
// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
func (oc *operationCtx) reapplyUpdate(clientset clientset.MicroServiceInterface) error {
	// First generate the patch
	oldData, err := json.Marshal(oc.orig)
	if err != nil {
		return err
	}
	newData, err := json.Marshal(oc.obj)
	if err != nil {
		return err
	}
	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}
	// Next get latest version of the microservice, apply the patch and retry the Update
	attempt := 1
	for {
		currWf, err := clientset.Get(oc.obj.Name, metav1.GetOptions{})
		if !common.IsRetryableKubeAPIError(err) {
			return err
		}
		currWfBytes, err := json.Marshal(currWf)
		if err != nil {
			return err
		}
		newWfBytes, err := jsonpatch.MergePatch(currWfBytes, patchBytes)
		if err != nil {
			return err
		}
		var newObj v1alpha1.MicroService
		err = json.Unmarshal(newWfBytes, &newObj)
		if err != nil {
			return err
		}
		_, err = clientset.Update(&newObj)
		if err == nil {
			oc.log.Infof("Update retry attempt %d successful", attempt)
			return nil
		}
		attempt++
		oc.log.Warnf("Update retry attempt %d failed: %v", attempt, err)
		if attempt > 5 {
			return err
		}
	}
}
