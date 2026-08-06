package gohan

import (
	"fmt"
	"net/http"
	"strings"
)

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

	for _, rt := range r.routes {
		if matchPath(rt.path, reqPath) {
			if rt.method != reqMethod {
				Error(w, http.StatusMethodNotAllowed, fmt.Sprintf("Method %s not allowed for %s", reqMethod, reqPath))
				return
			}

			rt.handler(w, req)
			return
		}
	}

	Error(w, http.StatusNotFound, fmt.Sprintf("Route %s %s not found", reqMethod, reqPath))
}

func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := 0; i < len(patternParts); i++ {
		if strings.HasPrefix(patternParts[i], "{") || strings.HasPrefix(patternParts[i], ":") {
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}

	return true
}

func Param(r *http.Request, key string) string {
	val := r.URL.Query().Get(key)
	if val != "" {
		return val
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}