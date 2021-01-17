package hook

import (
	"bytes"
	"context"
	"devture-matrix-corporal/corporal/httphelp"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type httpRequestFactory func() (*http.Request, error)

// restServiceConsultingRequest reprents as request payload to be sent to a REST service.
//
// It contains various fields holding information about the Matrix Client-Server API request
// that was intercepted and is sent to the REST service for "consulting".
//
// See RESTServiceConsultor.
type restServiceConsultingRequest struct {
	Meta restServiceConsultingRequestMetaInformation `json:"meta"`

	Request restServiceConsultingRequestRequestInformation `json:"request"`

	// Response contains the upstream response information (if available).
	// This is only available for `after*` hooks.
	Response *restServiceConsultingRequestResponseInformation `json:"response"`
}

// restServiceConsultingRequestMetaInformation represents the meta information about an HTTP request we're consulting about.
type restServiceConsultingRequestMetaInformation struct {
	// HookID contains the name of the hook that provoked this consultation.
	HookID string `json:"hookId"`

	// AuthenticatedMatrixUserID contains the full Matrix User ID (MXID) of the user that made the request.
	// It might be null for unauthenticated requests.
	AuthenticatedMatrixUserID *string `json:"authenticatedMatrixUserId"`
}

// restServiceConsultingRequestRequestInformation represents the information about an HTTP request we're consulting about
type restServiceConsultingRequestRequestInformation struct {
	// URI is the raw request URI (corresponds to request.RequestURI).
	// It contains escape sequences, etc., and a query string.
	// Example: `/_matrix/client/r0/rooms/!AbCdEF%3Aexample.com/invite?something=here`
	URI string `json:"URI"`

	// Path is the path parsed out of the raw request URI (corresponds to request.URL.Path).
	// Example: `/_matrix/client/r0/rooms/!AbCdEF:example.com/invite`
	Path string `json:"path"`

	Method string `json:"method"`

	Headers map[string]string `json:"headers"`

	Payload string `json:"payload"`
}

// restServiceConsultingRequestResponseInformation represents the information about an upstream HTTP response we're consulting about
type restServiceConsultingRequestResponseInformation struct {
	StatusCode int `json:"statusCode"`

	Headers map[string]string `json:"headers"`

	Payload string `json:"payload"`
}

// RESTServiceConsultor is a helper which consults a REST API about a specific Matrix Client-Server API request.
//
// The API can in turn log or analyze the request payload and decide how it should be handled.
// It can answer with any valid hook data (rejection, response, passthrough, another REST service call, etc.).
//
// The payload sent to the API is seen in restServiceConsultingRequest.
type RESTServiceConsultor struct {
	defaultTimeoutDuration time.Duration

	httpClient *http.Client
}

func NewRESTServiceConsultor(defaultTimeoutDuration time.Duration) *RESTServiceConsultor {
	return &RESTServiceConsultor{
		defaultTimeoutDuration: defaultTimeoutDuration,

		httpClient: &http.Client{},
	}
}

// Consult consults the specified REST service and returns a new Hook containing the response.
// The result-Hook defines some other action to take (pass, reject, consult another REST service, etc).
func (me *RESTServiceConsultor) Consult(request *http.Request, response *http.Response, hook Hook, logger *logrus.Entry) (*Hook, error) {
	// We use a factory, because:
	// - each time we retry, we need to use a new http.Request.
	//    - The request.Body reader can only be used once.
	//    - Timeouts in the context also need to be reset.
	// - we'd rather prepare the factory now (for async requests), because we can't guarantee what happens with the original
	//   request's body in the future (once we exit this function). Something somewhere may consume it, making us unable
	//   to build a proper payload for the request we'll send to the REST service.
	consultingHTTPRequestFactory, err := prepareConsultingHTTPRequestFactory(request, response, hook, me.defaultTimeoutDuration)
	if err != nil {
		return nil, err
	}

	if hook.RESTServiceAsync {
		// We do the same thing we do synchronously. We just do it in the background and don't care what happens.
		// Still, logging, etc., is done.
		go me.callRestServiceWithRetries(consultingHTTPRequestFactory, hook, logger)

		if hook.RESTServiceAsyncResultHook != nil {
			return hook.RESTServiceAsyncResultHook, nil
		}

		return &Hook{Action: ActionPassUnmodified}, nil
	}

	responseHook, err := me.callRestServiceWithRetries(consultingHTTPRequestFactory, hook, logger)
	if err != nil {
		if hook.RESTServiceContingencyHook == nil {
			// No contingency. We have no choice but to error-out.
			return nil, err
		}

		logger.Warnf("Swallowing REST service error and responding with contingency hook: %s", err)

		return hook.RESTServiceContingencyHook, nil
	}

	return responseHook, nil
}

func (me *RESTServiceConsultor) callRestServiceWithRetries(
	requestFactory httpRequestFactory,
	hook Hook,
	logger *logrus.Entry,
) (*Hook, error) {
	attemptsCount := uint(1)
	if hook.RESTServiceRetryAttempts != nil {
		attemptsCount += *hook.RESTServiceRetryAttempts
	}

	var restError error

	for attemptNumber := uint(1); attemptNumber <= attemptsCount; attemptNumber++ {
		requestToSend, err := requestFactory()
		if err != nil {
			logger.Errorf("RESTServiceConsultor: failed preparing HTTP Request: %s", err)
			return nil, err
		}

		logger = logger.WithFields(logrus.Fields{
			"RESTRrequestMethod": requestToSend.Method,
			"RESTRrequestURL":    requestToSend.URL,
			"RESTRequestAttempt": attemptNumber,
		})

		if attemptNumber > 1 {
			// All attempts after the first one are potentially delayed.
			if hook.RESTServiceRetryWaitTimeMilliseconds != nil {
				logger.Debugf("Waiting %d ms before retrying\n", *hook.RESTServiceRetryWaitTimeMilliseconds)

				t := time.NewTimer(time.Duration(*hook.RESTServiceRetryWaitTimeMilliseconds) * time.Millisecond)
				defer t.Stop()
				<-t.C
			}
		}

		logger.Debugf("RESTServiceConsultor: making HTTP request")

		resp, err := me.httpClient.Do(requestToSend)
		if err != nil {
			restError = fmt.Errorf("Error fetching from URL: %s", err)
			logger.Warnf("RESTServiceConsultor: failed: %s", restError)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			restError = fmt.Errorf("Non-200 response: %d", resp.StatusCode)
			logger.Warnf("RESTServiceConsultor: failed: %s", restError)
			continue
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// This is probably an error on our side, so retrying may be silly.
			restError = fmt.Errorf("Failed reading HTTP response body: %s", err)
			logger.Warnf("RESTServiceConsultor: failed: %s", restError)
			continue
		}

		var responseHook Hook
		err = json.Unmarshal(bodyBytes, &responseHook)
		if err != nil {
			restError = fmt.Errorf("Failed parsing JSON out of response: %s", err)
			logger.Warnf("RESTServiceConsultor: failed: %s", restError)
			continue
		}

		return &responseHook, nil
	}

	err := fmt.Errorf(
		"Failed after trying %d times. Last error: %s",
		attemptsCount,
		restError,
	)

	logger.Warnf("RESTServiceConsultor: ultimately failed: %s", restError)

	return nil, err
}

func prepareConsultingHTTPRequestFactory(
	request *http.Request,
	response *http.Response,
	hook Hook,
	defaultTimeoutDuration time.Duration,
) (httpRequestFactory, error) {
	if hook.RESTServiceURL == nil || *hook.RESTServiceURL == "" {
		return nil, fmt.Errorf("Cannot use NewRESTServiceConsultor with an empty RESTServiceURL")
	}

	// We extract the payload once, when making the factory, because that's when we know it's there
	// and it's safe for us to operate on.
	//
	// It also makes sense to do it just once and reuse it for retries as well.
	//
	// Extracting the payload for each request is wasteful and may also not be possible to do,
	// if we do it later on for `RESTServiceAsync = true` requests.
	consultingRequestPayload, err := prepareConsultingHTTPRequestPayload(request, response, hook)
	if err != nil {
		return nil, fmt.Errorf("Could not prepare request payload to be sent to the REST service: %s", err)
	}

	consultingRequestPayloadBytes, err := json.Marshal(consultingRequestPayload)
	if err != nil {
		return nil, fmt.Errorf("Could not serialize request payload to be sent to the REST service: %s", err)
	}

	consultingRequestMethod := "POST"
	if hook.RESTServiceRequestMethod != nil {
		consultingRequestMethod = *hook.RESTServiceRequestMethod
	}

	timeoutDuration := defaultTimeoutDuration
	if hook.RESTServiceRequestTimeoutMilliseconds != nil {
		timeoutDuration = time.Duration(*hook.RESTServiceRequestTimeoutMilliseconds) * time.Millisecond
	}

	return func() (*http.Request, error) {
		// This needs to be done each time, because it uses absolute time inside.
		ctx, _ := context.WithTimeout(context.Background(), timeoutDuration)

		consultingHTTPRequest, err := http.NewRequestWithContext(
			ctx,
			consultingRequestMethod,
			*hook.RESTServiceURL,
			bytes.NewReader(consultingRequestPayloadBytes),
		)
		if err != nil {
			return nil, err
		}

		consultingHTTPRequest.Header.Set("Content-Type", "application/json")
		if hook.RESTServiceRequestHeaders != nil {
			for k, v := range *hook.RESTServiceRequestHeaders {
				consultingHTTPRequest.Header.Set(k, v)
			}
		}

		return consultingHTTPRequest, nil
	}, nil
}

func prepareConsultingHTTPRequestPayload(request *http.Request, response *http.Response, hook Hook) (*restServiceConsultingRequest, error) {
	consultingRequest := restServiceConsultingRequest{}
	consultingRequest.Request.URI = request.RequestURI
	consultingRequest.Request.Path = request.URL.Path
	consultingRequest.Request.Method = request.Method

	consultingRequest.Request.Headers = map[string]string{}
	for headerName, headerValuesList := range request.Header {
		consultingRequest.Request.Headers[headerName] = httpHeaderListToHeaderValue(headerValuesList)
	}

	payloadBytes, err := httphelp.GetRequestBody(request)
	if err != nil {
		return nil, fmt.Errorf("Failed reading request body: %s", err)
	}
	consultingRequest.Request.Payload = string(payloadBytes)

	consultingRequest.Meta.HookID = hook.ID
	matrixUserIDInterface := request.Context().Value("userId")
	if matrixUserIDInterface != nil {
		matrixUserIDString := matrixUserIDInterface.(string)
		consultingRequest.Meta.AuthenticatedMatrixUserID = &matrixUserIDString
	}

	if response != nil {
		consultingRequest.Response = &restServiceConsultingRequestResponseInformation{
			StatusCode: response.StatusCode,
		}

		consultingRequest.Response.Headers = map[string]string{}
		for headerName, headerValuesList := range response.Header {
			consultingRequest.Response.Headers[headerName] = httpHeaderListToHeaderValue(headerValuesList)
		}

		responseBytes, err := httphelp.GetResponseBody(response)
		if err != nil {
			return nil, fmt.Errorf("Failed reading response body: %s", err)
		}

		consultingRequest.Response.Payload = string(responseBytes)
	}

	return &consultingRequest, nil
}

func httpHeaderListToHeaderValue(headerValuesList []string) string {
	// Go from []string{"gzip, deflate"} to `"gzip, deflate"`
	return strings.Join(headerValuesList, ", ")
}
