package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"path"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/apis"
	"github.com/portainer/k2d/internal/api/core"
	"github.com/portainer/k2d/internal/api/k2d"
	"github.com/portainer/k2d/internal/api/root"
	"github.com/portainer/k2d/internal/config"
	"github.com/portainer/k2d/internal/controller"
	"github.com/portainer/k2d/internal/logging"
	"github.com/portainer/k2d/internal/middleware"
	"github.com/portainer/k2d/internal/openapi"
	"github.com/portainer/k2d/internal/ssl"
	"github.com/portainer/k2d/internal/token"
	"github.com/portainer/k2d/internal/types"
	"github.com/portainer/k2d/pkg/network"
	"github.com/sethvargo/go-envconfig"
)

func getAdvertiseIpAddr(advertiseAddr string) (net.IP, error) {
	if advertiseAddr != "" {
		return network.GetIPv4(advertiseAddr)
	}

	return network.GetLocalIpAddr()
}

func main() {
	ctx := context.Background()

	var cfg config.Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		log.Fatalf("unable to parse configuration: %s", err)
	}

	logger, err := logging.NewLogger(cfg.LogLevel, cfg.LogFormat == "json")
	if err != nil {
		log.Fatalf("unable to initialize logger: %s", err)
	}
	defer logger.Sync()

	// We add the logger to the main context
	ctx = logging.ContextWithLogger(ctx, logger)

	logger.Infow("starting k2d",
		"version", types.Version,
		"config", cfg,
	)

	ip, err := getAdvertiseIpAddr(cfg.AdvertiseAddr)
	if err != nil {
		logger.Fatalf("unable to get advertise IP address: %s", err)
	}

	err = ssl.EnsureTLSCertificatesExist(ctx, cfg.DataPath, ip)
	if err != nil {
		logger.Fatalf("unable to setup TLS certificates: %s", err)
	}

	tokenPath := path.Join(cfg.DataPath, "token")
	encodedSecret, err := token.RetrieveOrCreateEncodedSecret(logger, cfg.Secret, tokenPath)
	if err != nil {
		logger.Fatalf("unable to retrieve or create encoded secret: %s", err)
	}

	serverConfiguration := &types.K2DServerConfiguration{
		ServerIpAddr: ip.String(),
		ServerPort:   cfg.Port,
		CaPath:       ssl.SSLCAPath(cfg.DataPath),
		TokenPath:    tokenPath,
		Secret:       encodedSecret,
	}

	kubeDockerAdapterOptions := &adapter.KubeDockerAdapterOptions{
		K2DConfig:           &cfg,
		Logger:              logger,
		ServerConfiguration: serverConfiguration,
	}

	kubeDockerAdapter, err := adapter.NewKubeDockerAdapter(kubeDockerAdapterOptions)
	if err != nil {
		logger.Fatalf("unable to create docker adapter: %s", err)
	}

	_, err = kubeDockerAdapter.Ping(ctx)
	if err != nil {
		logger.Fatalf("unable to connect to local docker server, make sure the docker socket is reachable at /var/run/docker.sock: %s", err)
	}

	err = kubeDockerAdapter.EnsureRequiredDockerResourcesExist(ctx)
	if err != nil {
		logger.Fatalf("unable to ensure required docker resources exist: %s", err)
	}

	err = kubeDockerAdapter.StoreServiceAccountSecret(tokenPath, ssl.SSLCAPath(cfg.DataPath))
	if err != nil {
		logger.Fatalf("unable to store system secret: %s", err)
	}

	if cfg.PortainerEdgeKey != "" {
		err = kubeDockerAdapter.DeployPortainerEdgeAgent(ctx, cfg.PortainerEdgeKey, cfg.PortainerEdgeID, cfg.PortainerAgentVersion)
		if err != nil {
			logger.Fatalf("unable to deploy portainer edge agent: %s", err)
		}
	}

	operations := make(chan controller.Operation)
	go controller.NewOperationController(logger, kubeDockerAdapter, cfg.OperationBatchMaxSize).StartControlLoop(operations)
	defer close(operations)

	container := restful.NewContainer()

	// We add the logger to the context of the request
	container.Filter(func(r *restful.Request, w *restful.Response, chain *restful.FilterChain) {
		ctx := logging.ContextWithLogger(r.Request.Context(), logger)
		r.Request = r.Request.WithContext(ctx)
		chain.ProcessFilter(r, w)
	})

	container.Filter(middleware.AddTracingHeaders)
	container.Filter(middleware.LogRequests)
	container.Filter(middleware.CheckAuthenticationHeader(encodedSecret))

	// We build the API
	root := root.NewRootAPI()
	// /version
	container.Add(root.Version())
	// /healthz
	container.Add(root.Healthz())

	core := core.NewCoreAPI(kubeDockerAdapter, operations)
	// /api/v1
	container.Add(core.V1())

	apis := apis.NewApisAPI(kubeDockerAdapter, operations)
	// /apis
	container.Add(apis.APIs())
	// /apis/apps
	container.Add(apis.Apps())
	// /apis/events.k8s.io
	container.Add(apis.Events())
	// /apis/authorization.k8s.io
	container.Add(apis.Authorization())

	k2d := k2d.NewK2DAPI(serverConfiguration, kubeDockerAdapter)
	// /k2d/kubeconfig
	container.Add(k2d.Kubeconfig())
	// /k2d/system
	container.Add(k2d.System())

	// We build and host the OpenAPI specs from the API that we have registered
	// This is used by kubectl when using the kubectl apply command
	config := restfulspec.Config{
		WebServices:                   container.RegisteredWebServices(),
		APIPath:                       "/openapi/v2",
		DisableCORS:                   true,
		PostBuildSwaggerObjectHandler: openapi.SwaggerObject,
	}

	openAPIv2, err := openapi.NewOpenAPIService(config)
	if err != nil {
		logger.Fatalf("unable to build OpenAPI web service")
	}

	// /openapi/v2
	container.Add(openAPIv2)

	logger.Infow("starting k2d server on HTTPS port",
		"address", fmt.Sprintf(":%d", cfg.Port),
		"advertise_address", ip.String(),
		"secret", encodedSecret,
	)

	logger.Infoln("use the command below to retrieve the kubeconfig file")
	logger.Infof("curl --insecure -H \"Authorization: Bearer %s\" https://%s:%d/k2d/kubeconfig",
		encodedSecret, serverConfiguration.ServerIpAddr, serverConfiguration.ServerPort)

	err = http.ListenAndServeTLS(
		fmt.Sprintf(":%d", cfg.Port),
		ssl.SSLCertPath(cfg.DataPath),
		ssl.SSLKeyPath(cfg.DataPath),
		container)

	logger.Fatal(err)
}
