package interceptor

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type InterceptorResult int

const (
	InterceptorResultProxy InterceptorResult = iota
	InterceptorResultDeny
)

type InterceptorResponse struct {
	Result InterceptorResult

	LoggingContextFields logrus.Fields

	ErrorCode    string
	ErrorMessage string
}

type Interceptor interface {
	Intercept(*http.Request) InterceptorResponse
}
