package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/febrianj/go-task-management-api/internal/apperr"
)

const maxBodyBytes = 1 << 20 // MiB

type validatable interface {
	Validate() []apperr.FieldError
}

func decodeAndValidate[T validatable](w http.ResponseWriter, r *http.Request, dst *T) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return apperr.BadRequest("request body must not be empty")
		}
		return apperr.BadRequest("malformed JSON body").WithErr(err)
	}

	if details := (*dst).Validate(); len(details) > 0 {
		return apperr.Validation(details)
	}

	return nil
}
