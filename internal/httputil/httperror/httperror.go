package httperror

import "fmt"

type Error struct {
	Status int
	Body   any
}

func (e *Error) Error() string {
	return fmt.Sprintf("http error: status=%d", e.Status)
}

func New(status int, body any) *Error {
	return &Error{Status: status, Body: body}
}
