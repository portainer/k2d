package controller

import (
	"context"
	"sync"
	"time"

	"github.com/portainer/k2d/internal/adapter"
	"go.uber.org/zap"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type (
	OperationController struct {
		adapter      *adapter.KubeDockerAdapter
		logger       *zap.SugaredLogger
		maxBatchSize int
	}

	Operation struct {
		Priority  OperationPriority
		Operation interface{}
		RequestID string
	}

	OperationBatch struct {
		HighPriorityOperations   []Operation
		MediumPriorityOperations []Operation
		LowPriorityOperations    []Operation
	}
)

type OperationPriority int

const (
	HighPriorityOperation OperationPriority = iota
	MediumPriorityOperation
	LowPriorityOperation
)

func (p OperationPriority) String() string {
	switch p {
	case HighPriorityOperation:
		return "High Priority Operation"
	case MediumPriorityOperation:
		return "Medium Priority Operation"
	case LowPriorityOperation:
		return "Low Priority Operation"
	default:
		return "Unknown Priority Operation"
	}
}

func NewOperation(operation interface{}, priority OperationPriority, requestID string) Operation {
	return Operation{
		Priority:  priority,
		Operation: operation,
		RequestID: requestID,
	}
}

func NewOperationController(logger *zap.SugaredLogger, adapter *adapter.KubeDockerAdapter, maxBatchSize int) *OperationController {
	return &OperationController{
		adapter:      adapter,
		logger:       logger,
		maxBatchSize: maxBatchSize,
	}
}

// StartControlLoop initializes and controls a loop to handle incoming operations.
// It creates a queue with a maximum size of 25 and then processes these operations
// in a separate goroutine every 3 seconds.
// If the queue is full, it will wait until the current batch of operations is processed
// before creating a new queue and continuing to process incoming operations.
// The function uses a mutex to ensure thread-safety when creating the queue and
// adding operations to it.
// It ensures that all operations received from the input channel will be processed
// and none will be missed.
// The loop continues until the ops channel is closed and all operations have been processed.
func (controller *OperationController) StartControlLoop(ops chan Operation) {
	var queue chan Operation
	var mu sync.Mutex

	for num := range ops {
		mu.Lock()
		if queue == nil {
			queue = make(chan Operation, controller.maxBatchSize)
			go func(q chan Operation) {
				time.AfterFunc(3*time.Second, func() {
					mu.Lock()
					close(q)
					controller.processOperationQueue(q)
					queue = nil
					mu.Unlock()
				})
			}(queue)
		}

		if len(queue) < cap(queue) {
			queue <- num
		} else {
			// The queue is full. Wait for it to empty and create a new one.
			mu.Unlock()
			time.Sleep(3 * time.Second)
			mu.Lock()
			queue = make(chan Operation, 25)
			queue <- num
		}
		mu.Unlock()
	}
}

func newOperationBatch(operations []Operation) OperationBatch {
	return OperationBatch{
		HighPriorityOperations:   filterOperationsByPriority(operations, HighPriorityOperation),
		MediumPriorityOperations: filterOperationsByPriority(operations, MediumPriorityOperation),
		LowPriorityOperations:    filterOperationsByPriority(operations, LowPriorityOperation),
	}
}

func filterOperationsByPriority(operations []Operation, priority OperationPriority) []Operation {
	var filteredOperations []Operation

	for _, op := range operations {
		if op.Priority == priority {
			filteredOperations = append(filteredOperations, op)
		}
	}

	return filteredOperations
}

func (controller *OperationController) processOperationQueue(queue chan Operation) {
	operations := collectOperations(queue)

	controller.logger.Debugw("processing operation batch",
		"batch_size", len(operations),
	)

	batch := newOperationBatch(operations)

	controller.processPriorityOperations(batch.HighPriorityOperations, HighPriorityOperation)
	controller.processPriorityOperations(batch.MediumPriorityOperations, MediumPriorityOperation)
	controller.processPriorityOperations(batch.LowPriorityOperations, LowPriorityOperation)
}

func collectOperations(queue chan Operation) []Operation {
	operations := []Operation{}
	for op := range queue {
		operations = append(operations, op)
	}
	return operations
}

func (controller *OperationController) processPriorityOperations(ops []Operation, priority OperationPriority) {
	var wg sync.WaitGroup

	controller.logger.Debugw("processing operations",
		"operation_count", len(ops),
		"priority", priority.String(),
	)

	for _, op := range ops {
		wg.Add(1)
		go controller.processOperation(op, &wg)
	}

	wg.Wait()
}

func (controller *OperationController) processOperation(op Operation, wg *sync.WaitGroup) {
	defer wg.Done()

	switch op.Operation.(type) {
	case *corev1.Pod:
		err := controller.createPod(op)
		if err != nil {
			controller.logger.Errorw("unable to create pod",
				"error", err,
				"request_id", op.RequestID,
			)
		}
	case *appsv1.Deployment:
		err := controller.createDeployment(op)
		if err != nil {
			controller.logger.Errorw("unable to create deployment",
				"error", err,
				"request_id", op.RequestID,
			)
		}
	case *appsv1.StatefulSet:
		err := controller.createStatefulSet(op)
		if err != nil {
			controller.logger.Errorw("unable to create statefulset",
				"error", err,
				"request_id", op.RequestID,
			)
		}
	case *appsv1.DaemonSet:
		err := controller.createDaemonSet(op)
		if err != nil {
			controller.logger.Errorw("unable to create daemonset",
				"error", err,
				"request_id", op.RequestID,
			)
		}
	case *corev1.ConfigMap:
		err := controller.createConfigMap(op)
		if err != nil {
			controller.logger.Errorw("unable to create configmap",
				"error", err,
			)
		}
	case *corev1.Secret:
		err := controller.createSecret(op)
		if err != nil {
			controller.logger.Errorw("unable to create secret",
				"error", err,
			)
		}
	case *corev1.Service:
		err := controller.createService(op)
		if err != nil {
			controller.logger.Errorw("unable to update container",
				"error", err,
				"request_id", op.RequestID,
			)
		}
	}
}

func (controller *OperationController) createPod(op Operation) error {
	pod := op.Operation.(*corev1.Pod)
	return controller.adapter.CreateContainerFromPod(context.TODO(), pod)
}

func (controller *OperationController) createDeployment(op Operation) error {
	deployment := op.Operation.(*appsv1.Deployment)
	return controller.adapter.CreateContainerFromDeployment(context.TODO(), deployment)
}

func (controller *OperationController) createStatefulSet(op Operation) error {
	statefulSet := op.Operation.(*appsv1.StatefulSet)
	return controller.adapter.CreateContainerFromStatefulSet(context.TODO(), statefulSet)
}

func (controller *OperationController) createDaemonSet(op Operation) error {
	daemonSet := op.Operation.(*appsv1.DaemonSet)
	return controller.adapter.CreateContainerFromDaemonSet(context.TODO(), daemonSet)
}

func (controller *OperationController) createService(op Operation) error {
	service := op.Operation.(*corev1.Service)
	return controller.adapter.CreateContainerFromService(context.TODO(), service)
}

func (controller *OperationController) createConfigMap(op Operation) error {
	configMap := op.Operation.(*corev1.ConfigMap)
	return controller.adapter.CreateConfigMap(configMap)
}

func (controller *OperationController) createSecret(op Operation) error {
	secret := op.Operation.(*corev1.Secret)
	return controller.adapter.CreateSecret(secret)
}
