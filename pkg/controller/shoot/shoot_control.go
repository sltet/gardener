// Copyright 2018 The Gardener Authors.
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

package shoot

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gardener/gardener/pkg/apis/componentconfig"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	gardeninformers "github.com/gardener/gardener/pkg/client/garden/informers/externalversions/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/operation"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func (c *Controller) shootAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Logger.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	c.shootQueue.Add(key)
}

func (c *Controller) shootUpdate(oldObj, newObj interface{}) {
	var (
		oldShoot        = oldObj.(*gardenv1beta1.Shoot)
		newShoot        = newObj.(*gardenv1beta1.Shoot)
		oldShootJSON, _ = json.Marshal(oldShoot)
		newShootJSON, _ = json.Marshal(newShoot)
		shootLogger     = logger.NewShootLogger(logger.Logger, newShoot.ObjectMeta.Name, newShoot.ObjectMeta.Namespace, "")
	)
	shootLogger.Debugf(string(oldShootJSON))
	shootLogger.Debugf(string(newShootJSON))

	// If the generation did not change for an update event (i.e., no changes to the .spec section have
	// been made), we do not want to add the Shoot to th queue. The period reconciliation is handled
	// elsewhere by adding the Shoot to the queue to dedicated times.
	if newShoot.Generation == newShoot.Status.ObservedGeneration {
		shootLogger.Debug("Do not need to do anything as the Update event occurred due to .status field changes")
		return
	}

	c.shootAdd(newObj)
}

func (c *Controller) shootDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Logger.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}

	c.shootQueue.Add(key)
}

func (c *Controller) reconcileShootKey(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	shoot, err := c.shootLister.Shoots(namespace).Get(name)
	if apierrors.IsNotFound(err) {
		logger.Logger.Debugf("[SHOOT RECONCILE] %s - skipping because Shoot has been deleted", key)
		return nil
	}
	if err != nil {
		logger.Logger.Infof("[SHOOT RECONCILE] %s - unable to retrieve object from store: %v", key, err)
		return err
	}

	shootLogger := logger.NewShootLogger(logger.Logger, shoot.ObjectMeta.Name, shoot.ObjectMeta.Namespace, "")
	if shoot.DeletionTimestamp != nil && !sets.NewString(shoot.Finalizers...).Has(gardenv1beta1.GardenerName) {
		shootLogger.Debug("Do not need to do anything as the Shoot does not have my finalizer")
		c.shootQueue.Forget(key)
		return nil
	}

	var (
		reconcileErr       = c.control.ReconcileShoot(shoot, key)
		durationToNextSync = scheduleNextSync(shoot.ObjectMeta, reconcileErr != nil, c.config.Controllers.Shoot)
	)

	c.shootQueue.AddAfter(key, durationToNextSync)
	shootLogger.Infof("Scheduled next reconciliation for Shoot '%s' in %s", key, durationToNextSync)
	return nil
}

func scheduleNextSync(objectMeta metav1.ObjectMeta, errorOccured bool, config componentconfig.ShootControllerConfiguration) time.Duration {
	if errorOccured {
		return (*config.RetrySyncPeriod).Duration
	}

	var (
		syncPeriod                 = config.SyncPeriod
		respectSyncPeriodOverwrite = *config.RespectSyncPeriodOverwrite

		currentTimeNano  = time.Now().UnixNano()
		creationTimeNano = objectMeta.CreationTimestamp.UnixNano()
	)

	if syncPeriodOverwrite, ok := objectMeta.Annotations[common.ShootSyncPeriod]; ok && respectSyncPeriodOverwrite {
		if syncPeriodAnnotation, err := time.ParseDuration(syncPeriodOverwrite); err == nil {
			if syncPeriodAnnotation >= time.Minute {
				syncPeriod = metav1.Duration{Duration: syncPeriodAnnotation}
			}
		}
	}

	var (
		syncPeriodNano = syncPeriod.Nanoseconds()
		nextSyncNano   = currentTimeNano - (currentTimeNano-creationTimeNano)%syncPeriodNano + syncPeriodNano
	)

	return time.Duration(nextSyncNano - currentTimeNano)
}

// ControlInterface implements the control logic for updating Shoots. It is implemented as an interface to allow
// for extensions that provide different semantics. Currently, there is only one implementation.
type ControlInterface interface {
	// ReconcileShoot implements the control logic for Shoot creation, update, and deletion.
	// If an implementation returns a non-nil error, the invocation will be retried using a rate-limited strategy.
	// Implementors should sink any errors that they do not wish to trigger a retry, and they may feel free to
	// exit exceptionally at any point provided they wish the update to be re-run at a later point in time.
	ReconcileShoot(shoot *gardenv1beta1.Shoot, key string) error
}

// NewDefaultControl returns a new instance of the default implementation ControlInterface that
// implements the documented semantics for Shoots. updater is the UpdaterInterface used
// to update the status of Shoots. You should use an instance returned from NewDefaultControl() for any
// scenario other than testing.
func NewDefaultControl(k8sGardenClient kubernetes.Client, k8sGardenInformers gardeninformers.Interface, secrets map[string]*corev1.Secret, imageVector imagevector.ImageVector, identity *gardenv1beta1.Gardener, config *componentconfig.ControllerManagerConfiguration, gardenerNamespace string, recorder record.EventRecorder, updater UpdaterInterface) ControlInterface {
	return &defaultControl{k8sGardenClient, k8sGardenInformers, secrets, imageVector, identity, config, gardenerNamespace, recorder, updater}
}

type defaultControl struct {
	k8sGardenClient    kubernetes.Client
	k8sGardenInformers gardeninformers.Interface
	secrets            map[string]*corev1.Secret
	imageVector        imagevector.ImageVector
	identity           *gardenv1beta1.Gardener
	config             *componentconfig.ControllerManagerConfiguration
	gardenerNamespace  string
	recorder           record.EventRecorder
	updater            UpdaterInterface
}

func (c *defaultControl) ReconcileShoot(shootObj *gardenv1beta1.Shoot, key string) error {
	key, err := cache.MetaNamespaceKeyFunc(shootObj)
	if err != nil {
		return err
	}

	var (
		shoot         = shootObj.DeepCopy()
		operationID   = utils.GenerateRandomString(8)
		shootLogger   = logger.NewShootLogger(logger.Logger, shoot.Name, shoot.Namespace, operationID)
		lastOperation = shoot.Status.LastOperation
	)

	logger.Logger.Infof("[SHOOT RECONCILE] %s", key)
	shootJSON, _ := json.Marshal(shoot)
	shootLogger.Debugf(string(shootJSON))

	operation, err := operation.New(shoot, shootLogger, c.k8sGardenClient, c.k8sGardenInformers, c.identity, c.secrets, c.imageVector)
	if err != nil {
		shootLogger.Errorf("Could not initialize a new operation: %s", err.Error())
		return err
	}

	// We check whether the Shoot's last operation status field indicates that the last operation failed (i.e. the operation
	// will not be retried unless the shoot generation changes).
	if lastOperation != nil && lastOperation.State == gardenv1beta1.ShootLastOperationStateFailed && shoot.Generation == shoot.Status.ObservedGeneration {
		shootLogger.Infof("Will not reconcile as the last operation has been set to '%s' and the generation has not changed since then.", gardenv1beta1.ShootLastOperationStateFailed)
		return nil
	}

	// When a Shoot clusters deletion timestamp is set we need to delete the cluster and must not trigger a new reconciliation operation.
	if shoot.DeletionTimestamp != nil {
		// In order to protect users from accidential/undesired deletion we check whether there is an annotation whose value is equal to the
		// deletion timestamp itself. If the annotation is missing or the value does not match the deletion timestamp then we skip the deletion
		// until it gets confirmed (by correctly putting the annotation).
		if !metav1.HasAnnotation(shoot.ObjectMeta, common.ConfirmationDeletionTimestamp) || !common.CheckConfirmationDeletionTimestampValid(shoot.ObjectMeta) {
			shootLogger.Infof("Shoot cluster's deletionTimestamp is set but the confirmation annotation '%s' is missing. Skipping.", common.ConfirmationDeletionTimestamp)
		} else {
			// If we reach this line then the deletion timestamp is set, the confirmation annotation exists and its value matches the deletion
			// timestamp itself. Consequently, it's safe to trigger the Shoot cluster deletion here.
			c.recorder.Eventf(shoot, corev1.EventTypeNormal, gardenv1beta1.ShootEventDeleting, "[%s] Deleting Shoot cluster", operationID)
			if updateErr := c.updateShootStatusDeleteStart(operation); updateErr != nil {
				shootLogger.Errorf("Could not update the Shoot status after deletion start: %+v", updateErr)
				return updateErr
			}
			deleteErr := c.deleteShoot(operation)
			if deleteErr != nil {
				c.recorder.Eventf(shoot, corev1.EventTypeWarning, gardenv1beta1.ShootEventDeleteError, "[%s] %s", operationID, deleteErr.Description)
				if updateErr := c.updateShootStatusDeleteError(operation, deleteErr); updateErr != nil {
					shootLogger.Errorf("Could not update the Shoot status after deletion error: %+v", updateErr)
					return updateErr
				}
				return errors.New(deleteErr.Description)
			}
			c.recorder.Eventf(shoot, corev1.EventTypeNormal, gardenv1beta1.ShootEventDeleted, "[%s] Deleted Shoot cluster", operationID)
			if updateErr := c.updateShootStatusDeleteSuccess(operation); updateErr != nil {
				shootLogger.Errorf("Could not update the Shoot status after deletion success: %+v", updateErr)
				return updateErr
			}
		}

		return nil
	}

	operationType := gardenv1beta1.ShootLastOperationTypeReconcile
	if lastOperation == nil || (lastOperation.Type == gardenv1beta1.ShootLastOperationTypeCreate && lastOperation.State != gardenv1beta1.ShootLastOperationStateSucceeded) {
		operationType = gardenv1beta1.ShootLastOperationTypeCreate
	}

	c.recorder.Eventf(shoot, corev1.EventTypeNormal, gardenv1beta1.ShootEventReconciling, "[%s] Reconciling Shoot cluster state", operationID)
	if updateErr := c.updateShootStatusReconcileStart(operation, operationType); updateErr != nil {
		shootLogger.Errorf("Could not update the Shoot status after reconciliation start: %+v", updateErr)
		return updateErr
	}
	reconcileErr := c.reconcileShoot(operation, operationType)
	if reconcileErr != nil {
		c.recorder.Eventf(shoot, corev1.EventTypeWarning, gardenv1beta1.ShootEventReconcileError, "[%s] %s", operationID, reconcileErr.Description)
		if updateErr := c.updateShootStatusReconcileError(operation, operationType, reconcileErr); updateErr != nil {
			shootLogger.Errorf("Could not update the Shoot status after reconciliation error: %+v", updateErr)
			return updateErr
		}
		return errors.New(reconcileErr.Description)
	}
	c.recorder.Eventf(shoot, corev1.EventTypeNormal, gardenv1beta1.ShootEventReconciled, "[%s] Reconciled Shoot cluster state", operationID)
	if updateErr := c.updateShootStatusReconcileSuccess(operation, operationType); updateErr != nil {
		shootLogger.Errorf("Could not update the Shoot status after reconciliation success: %+v", updateErr)
		return updateErr
	}

	return nil
}
