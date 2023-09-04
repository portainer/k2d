package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/adapter/filters"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/batch"
)

type JobLogOptions struct {
	Timestamps bool
	Follow     bool
	Tail       string
}

func (adapter *KubeDockerAdapter) CreateContainerFromJob(ctx context.Context, job *batchv1.Job) error {
	opts := ContainerCreationOptions{
		containerName: job.Name,
		namespace:     job.Namespace,
		jobSpec:       job.Spec,
		labels:        job.Labels,
	}

	if job.Labels["app.kubernetes.io/managed-by"] == "Helm" {
		jobData, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("unable to marshal job: %w", err)
		}
		job.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(jobData)
	}

	opts.lastAppliedConfiguration = job.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]

	return adapter.createContainerFromJobSpec(ctx, opts)
}

// The GetJob implementation is using a filtered list approach as the Docker API provide different response types
// when inspecting a container and listing containers.
// The logic used to build a job from a container is based on the type returned by the list operation (types.Container)
// and not the inspect operation (types.ContainerJSON).
// This is because using the inspect operation everywhere would be more expensive overall.
func (adapter *KubeDockerAdapter) GetJob(ctx context.Context, jobName string, namespace string) (*batchv1.Job, error) {
	filter := filters.ByPod(namespace, jobName) // NOTE: I am not sure about this, should work?
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filter})
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %w", err)
	}

	var container *types.Container

	containerName := buildContainerName(jobName, namespace)
	for _, cntr := range containers {
		if cntr.Names[0] == "/"+containerName {
			container = &cntr
			break
		}
	}

	if container == nil {
		adapter.logger.Errorf("unable to find container for job %s in namespace %s", jobName, namespace)
		return nil, errors.ErrResourceNotFound
	}

	job, err := adapter.buildJobFromContainer(*container)
	if err != nil {
		return nil, fmt.Errorf("unable to get job: %w", err)
	}

	versionedJob := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(job, &versionedJob)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	return &versionedJob, nil
}

func (adapter *KubeDockerAdapter) GetJobLogs(ctx context.Context, namespace string, jobName string, opts JobLogOptions) (io.ReadCloser, error) {
	containerName := buildContainerName(jobName, namespace)
	container, err := adapter.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("unable to inspect container: %w", err)
	}

	return adapter.cli.ContainerLogs(ctx, container.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: opts.Timestamps,
		Follow:     opts.Follow,
		Tail:       opts.Tail,
	})
}

func (adapter *KubeDockerAdapter) GetJobTable(ctx context.Context, namespace string) (*metav1.Table, error) {
	jobList, err := adapter.listJobs(ctx, namespace)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list jobs: %w", err)
	}

	return k8s.GenerateTable(&jobList)
}

func (adapter *KubeDockerAdapter) ListJobs(ctx context.Context, namespace string) (batch.JobList, error) {
	jobList, err := adapter.listJobs(ctx, namespace)
	if err != nil {
		return batch.JobList{}, fmt.Errorf("unable to list jobs: %w", err)
	}

	versionedJobList := batch.JobList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "JobList",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&jobList, &versionedJobList)
	if err != nil {
		return batch.JobList{}, fmt.Errorf("unable to convert internal JobList to versioned JobList: %w", err)
	}

	return versionedJobList, nil
}

func (adapter *KubeDockerAdapter) buildJobFromContainer(container types.Container) (*batch.Job, error) {
	job := adapter.converter.ConvertContainerToJob(container)

	if container.Labels[k2dtypes.JobLastAppliedConfigLabelKey] != "" {
		internalJobSpecData := container.Labels[k2dtypes.JobLastAppliedConfigLabelKey]
		jobSpec := batch.JobSpec{}

		err := json.Unmarshal([]byte(internalJobSpecData), &jobSpec)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal job spec: %w", err)
		}

		job.Spec = jobSpec
	}

	return &job, nil
}

func (adapter *KubeDockerAdapter) listJobs(ctx context.Context, namespace string) (batch.JobList, error) {
	filter := filters.ByNamespace(namespace)
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filter})
	if err != nil {
		return batch.JobList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	jobs := []batch.Job{}

	for _, container := range containers {
		job, err := adapter.buildJobFromContainer(container)
		if err != nil {
			return batch.JobList{}, fmt.Errorf("unable to get jobs: %w", err)
		}

		jobs = append(jobs, *job)
	}

	return batch.JobList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "JobList",
			APIVersion: "v1",
		},
		Items: jobs,
	}, nil
}
