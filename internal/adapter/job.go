package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/adapter/filters"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/batch"
)

func (adapter *KubeDockerAdapter) CreateContainerFromJob(ctx context.Context, job *batchv1.Job) error {
	opts := ContainerCreationOptions{
		containerName: job.Name,
		namespace:     job.Namespace,
		podSpec:       job.Spec.Template.Spec,
		labels:        job.Spec.Template.Labels,
	}

	if opts.labels == nil {
		opts.labels = make(map[string]string)
	}

	opts.labels[k2dtypes.WorkloadLabelKey] = k2dtypes.JobWorkloadType

	if job.Labels["app.kubernetes.io/managed-by"] == "Helm" {
		jobData, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("unable to marshal job: %w", err)
		}
		job.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(jobData)
	}

	// kubectl create job does not pass the last-applied-configuration annotation
	// so we need to add it manually
	if job.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] == "" {
		jobData, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("unable to marshal job: %w", err)
		}
		opts.labels[k2dtypes.WorkloadLastAppliedConfigLabelKey] = string(jobData)
	}

	opts.lastAppliedConfiguration = job.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]

	return adapter.createContainerFromPodSpec(ctx, opts)
}

func (adapter *KubeDockerAdapter) getContainerFromJobName(ctx context.Context, jobName, namespace string) (types.Container, error) {
	filter := filters.ByJob(namespace, jobName)
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filter})
	if err != nil {
		return types.Container{}, fmt.Errorf("unable to list containers: %w", err)
	}

	if len(containers) == 0 {
		return types.Container{}, adaptererr.ErrResourceNotFound
	}

	if len(containers) > 1 {
		return types.Container{}, fmt.Errorf("multiple containers were found with the associated job name %s", jobName)
	}

	return containers[0], nil
}

func (adapter *KubeDockerAdapter) GetJob(ctx context.Context, jobName string, namespace string) (*batchv1.Job, error) {
	container, err := adapter.getContainerFromJobName(ctx, jobName, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to get container from job name: %w", err)
	}

	job, err := adapter.buildJobFromContainer(ctx, container)
	if err != nil {
		return nil, fmt.Errorf("unable to get job: %w", err)
	}

	versionedJob := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
	}

	err = adapter.ConvertK8SResource(job, &versionedJob)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	return &versionedJob, nil
}

func (adapter *KubeDockerAdapter) GetJobTable(ctx context.Context, namespace string) (*metav1.Table, error) {
	jobList, err := adapter.listJobs(ctx, namespace)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list jobs: %w", err)
	}

	return k8s.GenerateTable(&jobList)
}

func (adapter *KubeDockerAdapter) ListJobs(ctx context.Context, namespace string) (batchv1.JobList, error) {
	jobList, err := adapter.listJobs(ctx, namespace)
	if err != nil {
		return batchv1.JobList{}, fmt.Errorf("unable to list jobs: %w", err)
	}

	versionedJobList := batchv1.JobList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "JobList",
			APIVersion: "batch/v1",
		},
	}

	err = adapter.ConvertK8SResource(&jobList, &versionedJobList)
	if err != nil {
		return batchv1.JobList{}, fmt.Errorf("unable to convert internal JobList to versioned JobList: %w", err)
	}

	return versionedJobList, nil
}

func (adapter *KubeDockerAdapter) buildJobFromContainer(ctx context.Context, container types.Container) (*batch.Job, error) {
	if container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey] == "" {
		return nil, fmt.Errorf("unable to build job, missing %s label on container %s", k2dtypes.WorkloadLastAppliedConfigLabelKey, container.Names[0])
	}

	jobData := container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey]

	versionedJob := batchv1.Job{}

	err := json.Unmarshal([]byte(jobData), &versionedJob)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal versioned job: %w", err)
	}

	job := batch.Job{}
	err = adapter.ConvertK8SResource(&versionedJob, &job)
	if err != nil {
		return nil, fmt.Errorf("unable to convert versioned job spec to internal job spec: %w", err)
	}

	containerInspect, err := adapter.cli.ContainerInspect(ctx, container.ID)
	if err != nil {
		return nil, fmt.Errorf("unable to inspect the container: %w", err)
	}

	adapter.converter.UpdateJobFromContainerInfo(&job, container, containerInspect)

	return &job, nil
}

func (adapter *KubeDockerAdapter) listJobs(ctx context.Context, namespace string) (batch.JobList, error) {
	filter := filters.AllJobs(namespace)
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filter})
	if err != nil {
		return batch.JobList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	jobs := []batch.Job{}

	for _, container := range containers {
		job, err := adapter.buildJobFromContainer(ctx, container)
		if err != nil {
			return batch.JobList{}, fmt.Errorf("unable to get job: %w", err)
		}

		if job != nil {
			jobs = append(jobs, *job)
		}
	}

	return batch.JobList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "JobList",
			APIVersion: "batch/v1",
		},
		Items: jobs,
	}, nil
}
