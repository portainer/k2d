package openapi

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	openapi_v2 "github.com/google/gnostic/openapiv2"
	"github.com/munnerz/goautoneg"
	"google.golang.org/protobuf/proto"
)

// OpenAPIService is the service responsible for serving OpenAPI spec. It has
// the ability to safely change the spec while serving it.
type OpenAPIService struct {
	// rwMutex protects All members of this service.
	rwMutex      sync.RWMutex
	lastModified time.Time
	swagger      *spec.Swagger

	jsonCache  HandlerCache
	protoCache HandlerCache
	etagCache  HandlerCache
}

// NewOpenAPIService returns a new WebService that provides the API documentation of all services
// conform the OpenAPI documentation specifcation.
func NewOpenAPIService(config restfulspec.Config) (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Path(config.APIPath)
	ws.Produces("application/com.github.proto-openapi.spec.v2@v1.0+protobuf", restful.MIME_JSON)

	resource := OpenAPIService{}

	swagger := restfulspec.BuildSwagger(config)
	err := resource.UpdateSpec(swagger)
	if err != nil {
		return nil, err
	}

	resource.swagger = swagger

	ws.Route(ws.GET("/").Filter(EncodingFilter).To(resource.getSwagger))
	return ws, nil
}

func (o *OpenAPIService) UpdateSpec(openapiSpec *spec.Swagger) (err error) {
	o.rwMutex.Lock()
	defer o.rwMutex.Unlock()
	o.jsonCache = o.jsonCache.New(func() ([]byte, error) {
		return json.Marshal(openapiSpec)
	})
	o.protoCache = o.protoCache.New(func() ([]byte, error) {
		json, err := o.jsonCache.Get()
		if err != nil {
			return nil, err
		}
		return toProtoBinary(json)
	})
	o.etagCache = o.etagCache.New(func() ([]byte, error) {
		json, err := o.jsonCache.Get()
		if err != nil {
			return nil, err
		}
		return []byte(computeETag(json)), nil
	})
	o.lastModified = time.Now()

	return nil
}

func (o *OpenAPIService) getSwagger(req *restful.Request, resp *restful.Response) {
	accepted := []struct {
		Type           string
		SubType        string
		GetDataAndETag func() ([]byte, string, time.Time, error)
	}{
		{"application", "json", o.getSwaggerBytes},
		{"application", "com.github.proto-openapi.spec.v2@v1.0+protobuf", o.getSwaggerPbBytes},
	}

	decipherableFormats := req.Request.Header.Get("Accept")
	if decipherableFormats == "" {
		decipherableFormats = "*/*"
	}

	clauses := goautoneg.ParseAccept(decipherableFormats)
	resp.Header().Add("Vary", "Accept")
	for _, clause := range clauses {
		for _, accepts := range accepted {
			if clause.Type != accepts.Type && clause.Type != "*" {
				continue
			}
			if clause.SubType != accepts.SubType && clause.SubType != "*" {
				continue
			}
			// serve the first matching media type in the sorted clause list
			data, etag, lastModified, err := accepts.GetDataAndETag()
			if err != nil {
				log.Printf("Error in OpenAPI handler: %s", err)
				// only return a 503 if we have no older cache data to serve
				if data == nil {
					resp.WriteHeader(http.StatusServiceUnavailable)
					return
				}
			}
			resp.Header().Set("Etag", strconv.Quote(etag))
			resp.Header().Set("Last-Modified", lastModified.UTC().Format(http.TimeFormat))
			resp.Write(data)
		}
	}
}

func toProtoBinary(json []byte) ([]byte, error) {
	document, err := openapi_v2.ParseDocument(json)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(document)
}

func computeETag(data []byte) string {
	if data == nil {
		return ""
	}
	return fmt.Sprintf("%X", sha512.Sum512(data))
}

func (o *OpenAPIService) getSwaggerBytes() ([]byte, string, time.Time, error) {
	o.rwMutex.RLock()
	defer o.rwMutex.RUnlock()
	specBytes, err := o.jsonCache.Get()
	if err != nil {
		return nil, "", time.Time{}, err
	}
	etagBytes, err := o.etagCache.Get()
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return specBytes, string(etagBytes), o.lastModified, nil
}

func (o *OpenAPIService) getSwaggerPbBytes() ([]byte, string, time.Time, error) {
	o.rwMutex.RLock()
	defer o.rwMutex.RUnlock()
	specPb, err := o.protoCache.Get()
	if err != nil {
		return nil, "", time.Time{}, err
	}
	etagBytes, err := o.etagCache.Get()
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return specPb, string(etagBytes), o.lastModified, nil
}

// EncodingFilter Route Filter (defines FilterFunction)
func EncodingFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	// wrap responseWriter into a compressing one
	compress, _ := restful.NewCompressingResponseWriter(resp.ResponseWriter, restful.ENCODING_GZIP)
	resp.ResponseWriter = compress
	defer func() {
		compress.Close()
	}()
	chain.ProcessFilter(req, resp)
}

// SwaggerObject is to add more information to the swagger object
func SwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "k2d API",
			Description: "Manage Docker via Kubernetes API",
			Contact: &spec.ContactInfo{
				ContactInfoProps: spec.ContactInfoProps{
					Name:  "K2D",
					Email: "anthony.lapenna@portainer.io",
					URL:   "https://k2d.io",
				},
			},
			Version: "v1.0.0",
		},
	}
	swo.Tags = []spec.Tag{{TagProps: spec.TagProps{
		Name:        "k2d",
		Description: "Manage Docker via Kubernetes API"}}}
	swo.Swagger = "2.0"
}
