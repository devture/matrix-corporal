package matrix

const (
	ErrorBadJson          = "M_BAD_JSON"
	ErrorForbidden        = "M_FORBIDDEN"
	ErrorMissingToken     = "M_MISSING_TOKEN"
	ErrorUnknown          = "M_UNKNOWN"
	ErrorUnknownToken     = "M_UNKNOWN_TOKEN"
	ErrorUserInUse        = "M_USER_IN_USE"
	ErrorInvalidUsername  = "M_INVALID_USERNAME"
	ErrorUserDeactivated  = "M_USER_DEACTIVATED"
	ErrorLimitExceeded    = "M_LIMIT_EXCEEDED"
	ErrorMissingParameter = "M_MISSING_PARAM"
	ErrorNotFound         = "M_NOT_FOUND"
)

const (
	// DeactivatedAccountPrefixMarker is the prefix added to user account display names
	// when those accounts are marked as disabled.
	DeactivatedAccountPrefixMarker = "[x] "
)

const (
	LoginTypePassword = "m.login.password"
	LoginTypeToken    = "m.login.token"

	RegistrationTypeSharedSecret = "org.matrix.login.shared_secret"
)
