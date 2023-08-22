package config

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/k8s"
)

type ConfigService struct {
	caPath     string
	serverAddr string
	secret     string
}

func NewConfigService(caPath, serverAddr, secret string) ConfigService {
	return ConfigService{
		caPath:     caPath,
		serverAddr: serverAddr,
		secret:     secret,
	}
}

func (svc ConfigService) GetKubeconfig(r *restful.Request, w *restful.Response) {
	authorizationHeader := r.HeaderParameter("Authorization")
	secret := strings.TrimPrefix(authorizationHeader, "Bearer ")

	if secret != svc.secret {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid secret\n"))
		return
	}

	kubeconfig, err := k8s.GenerateKubeconfig(svc.caPath, svc.serverAddr, svc.secret)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to generate kubeconfig: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write(kubeconfig)
}
