package master

import (
	"errors"
	"net/mail"
	"strings"
)

// ErrNotFound is returned when a record does not exist (or was soft-deleted).
var ErrNotFound = errors.New("not found")

// ConflictError is returned for duplicate values or operations blocked by related records.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string { return e.Message }

func conflict(msg string) error { return &ConflictError{Message: msg} }

// ValidationError carries per-field messages so the frontend can show them next to inputs.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validation failed" }

func invalid(field, msg string) error {
	return &ValidationError{Fields: map[string]string{field: msg}}
}

// validator collects field errors; call err() at the end.
type validator struct {
	fields map[string]string
}

func (v *validator) add(field, msg string) {
	if v.fields == nil {
		v.fields = map[string]string{}
	}
	if _, exists := v.fields[field]; !exists {
		v.fields[field] = msg
	}
}

func (v *validator) err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: v.fields}
}

func validEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return nil
	}
	return &t
}
