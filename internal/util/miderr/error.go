package miderr

import (
	"fmt"
	"net/http"
)

// error which can diferentiate between error types (user/internal).
// Err is satisfies the error interface, you can return Err instead of err
type Err interface {
	Error() string
	Type() errorType
	HTTPCode() int
}

func BadRequest(s string) errorImpl {
	return errorImpl{"Bad Request: " + s, requestError}
}

func BadRequestF(format string, a ...interface{}) errorImpl {
	return BadRequest(fmt.Sprintf(format, a...))
}

func InternalErr(s string) errorImpl {
	return errorImpl{"Internal Error: " + s, internalError}
}

func InternalErrE(e error) errorImpl {
	return InternalErr(e.Error())
}
func InternalErrF(format string, a ...interface{}) errorImpl {
	return InternalErr(fmt.Sprintf(format, a...))
}

type errorType int

const (
	requestError errorType = iota
	internalError
)

type errorImpl struct {
	s string
	t errorType
}

func (me errorImpl) Error() string {
	return me.s
}

func (me errorImpl) Type() errorType {
	return me.t
}

var httpCodes = map[errorType]int{
	requestError:  http.StatusBadRequest,
	internalError: http.StatusInternalServerError,
}

func (me errorImpl) HTTPCode() int {
	return httpCodes[me.t]
}
