package matrix

import (
	"fmt"
	"strings"
	"time"

	"github.com/matrix-org/gomatrix"
	"github.com/sirupsen/logrus"
)

func IsErrorWithCode(err error, errorCode string) bool {
	if responseHttpError, couldCast := err.(gomatrix.HTTPError); couldCast {
		if responseError, couldCast := responseHttpError.WrappedError.(gomatrix.RespError); couldCast {
			return responseError.ErrCode == errorCode
		}
	}
	return false
}

// ExecuteWithRateLimitRetries executes callback and retries it after some backoff (if it hits a Matrix rate-limit).
//
// If the callback returns an `M_LIMIT_EXCEEDED` error, a few retries (with expontential backoff) are attempted.
// If the callback returns no error or returns another type of error, no retries are attempted.
//
// It seems like Matrix Synapse only rate-limits certain PUT/POST requests (and not GET),
// so those are the ones that would benefit by being wrapped in this.
func ExecuteWithRateLimitRetries(logger *logrus.Logger, requestName string, callback func() error) error {
	var lastCallbackErr error
	maxRetries := 5

	for retry := 0; retry < maxRetries; retry++ {
		lastCallbackErr = callback()
		if lastCallbackErr == nil {
			return nil
		}

		if !IsErrorWithCode(lastCallbackErr, ErrorLimitExceeded) {
			return lastCallbackErr
		}

		retryAfterSeconds := (retry + 1) * 5

		logger.Infof(
			"Request %s hit a rate limit, will retry in %d seconds",
			requestName,
			retryAfterSeconds,
		)

		time.Sleep(time.Duration(retryAfterSeconds) * time.Second)
	}

	logger.Errorf("Request %s failed after %d retries: %s", requestName, maxRetries, lastCallbackErr)

	// Let's preserve the original error instead of wrapping it.
	// Certain callers might want to inspect it.
	//
	// It's not like there's something interesting they would find there though,
	// since this is obviously a rate-limit error and none of our callers handle such errors
	// by themselves (hence them calling us).
	return lastCallbackErr
}

// IsUserDeactivatedAccordingToDisplayName tells if the user account appears to be disabled, judging by the display name.
// The Matrix protocol does not have a notion of enabled/disabled accounts,
// nor a good way to store such data so we're resorting to such hacks.
func IsUserDeactivatedAccordingToDisplayName(displayName string) bool {
	return strings.Contains(displayName, DeactivatedAccountPrefixMarker)
}

func CleanDeactivationMarkerFromDisplayName(displayName string) string {
	return displayName[len(DeactivatedAccountPrefixMarker):]
}

// DetermineFullUserId takes a user id and converts it to a full Matrix user id of the given home server (if not already)
func DetermineFullUserId(userIdLocalOrFull, homeserverDomainName string) (string, error) {
	if userIdLocalOrFull == "" {
		return "", fmt.Errorf("Empty user id")
	}

	if strings.HasPrefix(userIdLocalOrFull, "@") {
		// Somewhat looks like a full user id.
		// We don't care if it's on the same homeserver or not.
		return userIdLocalOrFull, nil
	}

	return fmt.Sprintf("@%s:%s", userIdLocalOrFull, homeserverDomainName), nil
}

// IsFullUserIdOfDomain tells if the given full user id is hosted on the given domain
func IsFullUserIdOfDomain(userIdFull string, homeserverDomainName string) bool {
	return strings.HasSuffix(userIdFull, fmt.Sprintf(":%s", homeserverDomainName))
}
