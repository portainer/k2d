package system

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
	k2dtypes "github.com/portainer/k2d/internal/types"
)

type SystemService struct {
	serverConfiguration *k2dtypes.K2DServerConfiguration
	adapter             *adapter.KubeDockerAdapter
}

type Diagnostics struct {
	Version             string                           `json:"version"`
	ServerConfiguration *k2dtypes.K2DServerConfiguration `json:"serverConfiguration"`
	OS                  string                           `json:"os"`
	Arch                string                           `json:"arch"`
	DockerInfo          types.Info                       `json:"dockerInfo"`
	DockerVersion       types.Version                    `json:"dockerVersion"`
}

func NewSystemService(cfg *k2dtypes.K2DServerConfiguration, adapter *adapter.KubeDockerAdapter) SystemService {
	return SystemService{
		serverConfiguration: cfg,
		adapter:             adapter,
	}
}

func (svc SystemService) Diagnostics(r *restful.Request, w *restful.Response) {
	authorizationHeader := r.HeaderParameter("Authorization")
	secret := strings.TrimPrefix(authorizationHeader, "Bearer ")

	if secret != svc.serverConfiguration.Secret {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid secret\n"))
		return
	}

	info, version, err := svc.adapter.InfoAndVersion(r.Request.Context())
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to retrieve Docker info: %w", err))
		return
	}

	info.DefaultAddressPools = nil
	svc.serverConfiguration.Secret = "[redacted]"

	diagnostics := Diagnostics{
		Version:             k2dtypes.Version,
		ServerConfiguration: svc.serverConfiguration,
		OS:                  runtime.GOOS,
		Arch:                runtime.GOARCH,
		DockerInfo:          info,
		DockerVersion:       version,
	}

	w.WriteAsJson(diagnostics)
}
