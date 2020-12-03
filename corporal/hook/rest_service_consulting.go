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

// RESTServiceConsultingRequest reprents as request payload to be sent to a REST service.
//
// It contains various fields holding information about the Matrix Client-Server API request
// that was intercepted and is sent to the REST service for "consulting".
//
// See RESTServiceConsultor.
type RESTServiceConsultingRequest struct {
	RequestURI string `json:"requestURI"`

	Method string `json:"method"`

	Headers map[string]string `json:"headers"`

	Payload string `json:"payload"`

	// If the Matrix Client-Server API request is for an authenticated user, this holds the ID for it.
	// Whether this is set depends on the hook event type.
	//
	// For example, hooks of type `EventTypeBeforeAnyRequest` run very early on,
	// before authentiation information has been figured out yet.
	AuthenticatedMatrixUserID *string `json:"authenticatedMatrixUserId"`
}

// RESTServiceConsultor is a helper which consults a REST API about a specific Matrix Client-Server API request.
//
// The API can in turn log or analyze the request payload and decide how it should be handled.
// It can answer with any valid hook data (rejection, response, passthrough, another REST service call, etc.).
//
// The payload sent to the API is seen in RESTServiceConsultingRequest.
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
func (me *RESTServiceConsultor) Consult(request *http.Request, hook Hook, logger *logrus.Entry) (*Hook, error) {
	consultingHTTPRequest, err := prepareConsultingHTTPRequest(request, hook, me.defaultTimeoutDuration)

	logger.Debugf("RESTServiceConsultor: calling %s %s", consultingHTTPRequest.Method, consultingHTTPRequest.URL)

	resp, err := me.httpClient.Do(consultingHTTPRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Non-200 response fetching from URL %s: %d", *hook.RESTServiceURL, resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed reading HTTP response body at %s: %s", *hook.RESTServiceURL, err)
	}

	var response Hook
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("Failed parsing JSON out of response at %s: %s", *hook.RESTServiceURL, err)
	}

	return &response, nil
}

func prepareConsultingHTTPRequest(request *http.Request, hook Hook, defaultTimeoutDuration time.Duration) (*http.Request, error) {
	if hook.RESTServiceURL == nil || *hook.RESTServiceURL == "" {
		return nil, fmt.Errorf("Cannot use NewRESTServiceConsultor with an empty RESTServiceURL")
	}

	consultingRequestPayload, err := prepareConsultingHTTPRequestPayload(request, hook)
	if err != nil {
		return nil, fmt.Errorf("Could not prepare request payload to be sent to the REST service: %s", err)
	}

	consultingRequestBytes, err := json.Marshal(consultingRequestPayload)
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
	ctx, _ := context.WithTimeout(context.Background(), timeoutDuration)

	consultingHTTPRequest, err := http.NewRequestWithContext(
		ctx,
		consultingRequestMethod,
		*hook.RESTServiceURL,
		ioutil.NopCloser(bytes.NewReader(consultingRequestBytes)),
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
}

func prepareConsultingHTTPRequestPayload(request *http.Request, hook Hook) (*RESTServiceConsultingRequest, error) {
	consultingRequest := RESTServiceConsultingRequest{}
	consultingRequest.RequestURI = request.RequestURI
	consultingRequest.Method = request.Method

	consultingRequest.Headers = map[string]string{}
	for headerName, headerValuesList := range request.Header {
		// Go from []string{"gzip, deflate"} to `"gzip, deflate"`
		headerValue := strings.Join(headerValuesList, ", ")
		consultingRequest.Headers[headerName] = headerValue
	}

	payloadBytes, err := httphelp.GetRequestBody(request)
	if err != nil {
		return nil, fmt.Errorf("Failed reading request body")
	}
	consultingRequest.Payload = string(payloadBytes)

	matrixUserIDInterface := request.Context().Value("userId")
	if matrixUserIDInterface != nil {
		matrixUserIDString := matrixUserIDInterface.(string)
		consultingRequest.AuthenticatedMatrixUserID = &matrixUserIDString
	}

	return &consultingRequest, nil
}
