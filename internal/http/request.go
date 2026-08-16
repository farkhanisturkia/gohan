package http

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func BindJSON(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body empty")
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}

	return nil
}