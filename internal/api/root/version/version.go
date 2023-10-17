package version

import (
	"runtime"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/version"
)

type VersionService struct{}

func NewVersionService() VersionService {
	return VersionService{}
}

func (svc VersionService) Version(r *restful.Request, w *restful.Response) {
	version := version.Info{
		Major:      "1",
		Minor:      "28",
		GitVersion: "v1.28.2-k2d",
		GoVersion:  runtime.Version(),
		Compiler:   "gc",
		Platform:   "linux/amd64",
	}

	w.WriteAsJson(version)
}
