package errors

import "errors"

type ErrorKind int

const (
	Transient ErrorKind = iota
	Permanent
)

type JobError struct {
	Kind    ErrorKind
	Message string
	Cause   error
}

func (e *JobError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *JobError) Unwrap() error {
	return e.Cause
}

func NewTransient(msg string, cause error) *JobError {
	return &JobError{Kind: Transient, Message: msg, Cause: cause}
}

func NewPermanent(msg string, cause error) *JobError {
	return &JobError{Kind: Permanent, Message: msg, Cause: cause}
}

func IsTransient(err error) bool {
	var jobErr *JobError
	if errors.As(err, &jobErr) {
		return jobErr.Kind == Transient
	}
	// unknown errors are treated as transient by default
	return true
}

func IsPermanent(err error) bool {
	return !IsTransient(err)
}
