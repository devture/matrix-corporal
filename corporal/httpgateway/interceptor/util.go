package interceptor

import (
	"github.com/sirupsen/logrus"
)

func createInterceptorErrorResponse(loggingContextFields logrus.Fields, errorCode, errorMessage string) InterceptorResponse {
	return InterceptorResponse{
		Result:               InterceptorResultDeny,
		LoggingContextFields: loggingContextFields,
		ErrorCode:            errorCode,
		ErrorMessage:         errorMessage,
	}
}
