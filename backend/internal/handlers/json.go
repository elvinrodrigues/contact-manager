package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"contact-manager/internal/utils"
)

// decodeJSON reads exactly one JSON object from the request body into dst.
//
// It is stricter than a bare json.Decoder in three ways that each hid a class of
// client bug: the Content-Type must actually be JSON, unknown fields are an
// error rather than silently dropped (so a misspelled field name fails loudly),
// and trailing content after the object is rejected.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return apiProblem{http.StatusUnsupportedMediaType, utils.CodeBadRequest,
				"Content-Type must be application/json"}
		}
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apiProblem{http.StatusBadRequest, utils.CodeBadRequest,
			"request body must contain a single JSON object"}
	}
	return nil
}

func decodeError(err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var maxBytesErr *http.MaxBytesError

	switch {
	case errors.As(err, &maxBytesErr):
		return apiProblem{http.StatusRequestEntityTooLarge, utils.CodePayloadTooLarge,
			fmt.Sprintf("request body must not exceed %d bytes", maxBytesErr.Limit)}

	case errors.As(err, &syntaxErr):
		return apiProblem{http.StatusBadRequest, utils.CodeBadRequest,
			fmt.Sprintf("malformed JSON at byte %d", syntaxErr.Offset)}

	case errors.As(err, &typeErr):
		return apiProblem{http.StatusBadRequest, utils.CodeValidation,
			fmt.Sprintf("field %q expects a %s", typeErr.Field, typeErr.Type)}

	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return apiProblem{http.StatusBadRequest, utils.CodeBadRequest, "request body is empty or truncated"}

	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return apiProblem{http.StatusBadRequest, utils.CodeBadRequest,
			fmt.Sprintf("unknown field %s", field)}
	}

	return apiProblem{http.StatusBadRequest, utils.CodeBadRequest, "invalid request body"}
}

// apiProblem is an error that already knows how it should be reported.
type apiProblem struct {
	status  int
	code    string
	message string
}

func (p apiProblem) Error() string { return p.message }

// writeProblem renders an apiProblem, falling back to a generic 400.
func writeProblem(w http.ResponseWriter, err error) {
	var p apiProblem
	if errors.As(err, &p) {
		utils.WriteError(w, p.status, p.code, p.message)
		return
	}
	utils.WriteError(w, http.StatusBadRequest, utils.CodeBadRequest, "invalid request body")
}
