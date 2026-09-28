package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type contextKey string

const paramsKey contextKey = "routeParams"

type HandlerFunc http.HandlerFunc

type route struct {
	method  string
	path    string
	handler http.HandlerFunc
}

type Router struct {
	routes []route
}

var defaultRouter = &Router{}

func SetRoute(method, path string, handler http.HandlerFunc) {
	defaultRouter.SetRoute(method, path, handler)
}

func (r *Router) SetRoute(method, path string, handler http.HandlerFunc) {
	method = strings.ToUpper(strings.TrimSpace(method))
	r.routes = append(r.routes, route{
		method:  method,
		path:    path,
		handler: handler,
	})
}

func Get(path string, handler http.HandlerFunc) {
	SetRoute("GET", path, handler)
}

func Post(path string, handler http.HandlerFunc) {
	SetRoute("POST", path, handler)
}

func Put(path string, handler http.HandlerFunc) {
	SetRoute("PUT", path, handler)
}

func Patch(path string, handler http.HandlerFunc) {
	SetRoute("PATCH", path, handler)
}

func Delete(path string, handler http.HandlerFunc) {
	SetRoute("DELETE", path, handler)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	reqPath := req.URL.Path
	reqMethod := req.Method

	pathMatched := false

	for _, rt := range r.routes {
		params, match := matchAndExtractParams(rt.path, reqPath)
		if match {
			pathMatched = true

			if reqMethod == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if rt.method == reqMethod {
				ctx := context.WithValue(req.Context(), paramsKey, params)
				rt.handler(w, req.WithContext(ctx))
				return
			}
		}
	}

	if pathMatched {
		Error(w, http.StatusMethodNotAllowed, fmt.Sprintf("Method %s not allowed for %s", reqMethod, reqPath))
		return
	}

	Error(w, http.StatusNotFound, fmt.Sprintf("Route %s %s not found", reqMethod, reqPath))
}

func matchAndExtractParams(pattern, path string) (map[string]string, bool) {
	if pattern == path {
		return nil, true
	}

	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return nil, false
	}

	params := make(map[string]string)

	for i := 0; i < len(patternParts); i++ {
		pPart := patternParts[i]
		valPart := pathParts[i]

		if strings.HasPrefix(pPart, "{") && strings.HasSuffix(pPart, "}") {
			paramName := pPart[1 : len(pPart)-1]
			params[paramName] = valPart
		} else if strings.HasPrefix(pPart, ":") {
			paramName := pPart[1:]
			params[paramName] = valPart
		} else if pPart != valPart {
			return nil, false
		}
	}

	return params, true
}

func Param(r *http.Request, key string) string {
	if params, ok := r.Context().Value(paramsKey).(map[string]string); ok {
		if val, exists := params[key]; exists {
			return val
		}
	}

	return r.URL.Query().Get(key)
}