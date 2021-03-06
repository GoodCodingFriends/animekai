package errors

import "github.com/morikuni/failure"

const (
	Canceled         failure.StringCode = "Canceled"
	DeadlineExceeded failure.StringCode = "DeadlineExceeded"
	InvalidArgument  failure.StringCode = "InvalidArgument"
	Internal         failure.StringCode = "Internal"
	Unauthenticated  failure.StringCode = "Unauthenticated"
)
