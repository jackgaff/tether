package respond

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Data  any        `json:"data,omitempty"`
	Meta  any        `json:"meta,omitempty"`
	Error *ErrorBody `json:"error,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, data any, meta any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	payload := Envelope{
		Data: data,
		Meta: meta,
	}

	_ = json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	payload := Envelope{
		Error: &ErrorBody{
			Code:    code,
			Message: message,
		},
	}

	_ = json.NewEncoder(w).Encode(payload)
}
