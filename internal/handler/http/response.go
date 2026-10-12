package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

const maxBodyBytes = 1 << 20

type Response struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, statusCode int, data any) {
	write(w, statusCode, Response{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, statusCode int, code, message string) {
	write(w, statusCode, Response{
		Success: false,
		Error:   &ErrorInfo{Code: code, Message: message},
	})
}

func write(w http.ResponseWriter, statusCode int, body Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

var errInvalidBody = errors.New("invalid request body")

// decodeJSON reads a size-limited body and rejects unknown fields, so typos
// in the payload fail loudly instead of being silently ignored.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errInvalidBody
	}
	return nil
}

func pathID(r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	return id, err == nil
}
