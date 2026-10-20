package procurement

import "errors"

// Kind classifies an application error so the HTTP layer can map it to a
// status code without leaking internal (database) error messages.
type Kind int

const (
	KindValidation Kind = iota + 1 // 400 - the request itself is wrong
	KindNotFound                   // 404 - the target record does not exist
	KindConflict                   // 409 - the record is in the wrong state
)

// AppError is a safe-to-show error.
type AppError struct {
	Kind Kind
	Msg  string
}

func (e *AppError) Error() string { return e.Msg }

func invalid(msg string) error  { return &AppError{Kind: KindValidation, Msg: msg} }
func notFound(msg string) error { return &AppError{Kind: KindNotFound, Msg: msg} }
func conflict(msg string) error { return &AppError{Kind: KindConflict, Msg: msg} }

// AsAppError reports whether err is (or wraps) an AppError.
func AsAppError(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
