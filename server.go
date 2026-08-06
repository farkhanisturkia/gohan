package gohan

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func Serve(port interface{}) error {
	var portStr string

	switch v := port.(type) {
	case string:
		portStr = v
	case *string:
		if v != nil {
			portStr = *v
		} else {
			portStr = "8080"
		}
	default:
		portStr = fmt.Sprintf("%v", port)
	}

	if !strings.HasPrefix(portStr, ":") {
		portStr = ":" + portStr
	}

	if len(defaultRouter.routes) == 0 {
		SetRoute("GET", "/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Welcome to Gohan Framework!"}`))
		})
	}

	log.Printf("[info] Server running in http://localhost%s\n", portStr)

	return http.ListenAndServe(portStr, defaultRouter)
}