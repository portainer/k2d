package apis

import (
	restful "github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/apis/apps"
	"github.com/portainer/k2d/internal/api/apis/authorization.k8s.io"
	"github.com/portainer/k2d/internal/api/apis/batch"
	"github.com/portainer/k2d/internal/api/apis/events.k8s.io"
	"github.com/portainer/k2d/internal/api/apis/storage.k8s.io"
	"github.com/portainer/k2d/internal/controller"
)

type (
	ApisAPI struct {
		apps          apps.AppsService
		batch         batch.BatchService
		events        events.EventsService
		authorization authorization.AuthorizationService
		storage       storage.StorageService
	}
)

func NewApisAPI(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) *ApisAPI {
	return &ApisAPI{
		apps:          apps.NewAppsService(operations, adapter),
		batch:         batch.NewBatchService(operations, adapter),
		events:        events.NewEventsService(adapter),
		authorization: authorization.NewAuthorizationService(),
		storage:       storage.NewStorageService(adapter),
	}
}

// /apis
// Used by Kubernetes clients to discover available APIs
func (api ApisAPI) APIs() *restful.WebService {
	routes := new(restful.WebService).
		Path("/apis").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	routes.Route(routes.GET("").
		To(ListAPIGroups))

	return routes
}

// /apis/storage.k8s.io
func (api ApisAPI) Storages() *restful.WebService {
	routes := new(restful.WebService).
		Path("/apis/storage.k8s.io").
		Produces(restful.MIME_JSON)

	// which versions are served by this api
	routes.Route(routes.GET("").
		To(api.storage.GetAPIVersions))

	// which resources are available under /apis/storage.k8s.io/v1
	routes.Route(routes.GET("/v1").
		To(api.storage.ListAPIResources))

	api.storage.RegisterStorageAPI(routes)
	return routes
}

// /apis/events.k8s.io
func (api ApisAPI) Events() *restful.WebService {
	routes := new(restful.WebService).
		Path("/apis/events.k8s.io").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	// which versions are served by this api
	routes.Route(routes.GET("").
		To(api.events.GetAPIVersions))

	// which resources are available under /apis/events.k8s.io/v1
	routes.Route(routes.GET("/v1").
		To(api.events.ListAPIResources))

	api.events.RegisterEventAPI(routes)
	return routes
}

// /apis/authorization.k8s.io
func (api ApisAPI) Authorization() *restful.WebService {
	routes := new(restful.WebService).
		Path("/apis/authorization.k8s.io").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	// which versions are served by this api
	routes.Route(routes.GET("").
		To(api.authorization.GetAPIVersions))

	// which resources are available under /apis/authorization.k8s.io/v1
	routes.Route(routes.GET("/v1").
		To(api.authorization.ListAPIResources))

	api.authorization.RegisterAuthorizationAPI(routes)
	return routes
}

// /apis/apps
func (api ApisAPI) Apps() *restful.WebService {
	routes := new(restful.WebService).
		Path("/apis/apps").
		Consumes(restful.MIME_JSON, "application/yml", "application/json-patch+json", "application/merge-patch+json", "application/strategic-merge-patch+json").
		Produces(restful.MIME_JSON)

	// which versions are served by this api
	routes.Route(routes.GET("").
		To(api.apps.GetAPIVersions))

	// which resources are available under /apis/apps/v1
	routes.Route(routes.GET("/v1").
		To(api.apps.ListAPIResources))

	api.apps.RegisterAppsAPI(routes)
	return routes
}

// /apis/batch
func (api ApisAPI) Batch() *restful.WebService {
	routes := new(restful.WebService).
		Path("/apis/batch").
		Consumes(restful.MIME_JSON, "application/yml", "application/json-patch+json", "application/merge-patch+json", "application/strategic-merge-patch+json").
		Produces(restful.MIME_JSON)

	// which versions are served by this api
	routes.Route(routes.GET("").
		To(api.batch.GetAPIVersions))

	// which resources are available under /apis/batch/v1
	routes.Route(routes.GET("/v1").
		To(api.batch.ListAPIResources))

	api.batch.RegisterBatchAPI(routes)
	return routes
}
