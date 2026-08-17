package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/RamDass1/test-api/internal/domain"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.Invalid("malformed request body")
	}
	if err := dec.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		return domain.Invalid("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type errorBody struct {
	Code    domain.Code `json:"code"`
	Message string      `json:"message"`
}

func writeError(w http.ResponseWriter, err error) {
	var de *domain.Error
	if !errors.As(err, &de) {
		de = domain.Internal(err, "unexpected error")
	}
	status := http.StatusInternalServerError
	msg := de.Message
	switch de.Code {
	case domain.CodeValidation:
		status = http.StatusBadRequest
	case domain.CodeUnauthorized:
		status = http.StatusUnauthorized
	case domain.CodeNotFound:
		status = http.StatusNotFound
	case domain.CodeConflict:
		status = http.StatusConflict
	case domain.CodeInternal:
		msg = "internal server error"
	}
	writeJSON(w, status, map[string]any{"error": errorBody{Code: de.Code, Message: msg}})
}
