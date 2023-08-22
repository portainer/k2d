package middleware

import (
	"net/http"
	"strings"

	restful "github.com/emicklei/go-restful/v3"
)

// CheckAuthenticationHeader returns a restful.FilterFunction that checks the Authorization header of a request.
// The header should contain a "Bearer" token, which is compared with the given encodedSecret parameter.
// If the token does not match the encodedSecret, the filter responds with an HTTP 401 Unauthorized status code and stops processing the request.
// If the token matches, the filter calls the next filter in the chain.
func CheckAuthenticationHeader(encodedSecret string) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		authorizationHeader := req.HeaderParameter("Authorization")
		secret := strings.TrimPrefix(authorizationHeader, "Bearer ")

		if secret != encodedSecret {
			resp.WriteHeader(http.StatusUnauthorized)
			resp.Write([]byte("invalid secret\n"))
			return
		}

		chain.ProcessFilter(req, resp)
	}
}
