package hook

import (
	"devture-matrix-corporal/corporal/util"
	"fmt"
	"net/http"
	"regexp"
)

var (
	// HookMatchRuleTypeURLPath is a match rule type that requires a match against the incoming HTTP request's HTTP Method (GET, POST, etc)
	HookMatchRuleTypeHTTPMethod = "method"

	// HookMatchRuleTypeURLPath is a match rule type that requires a match against the incoming HTTP request's URI.
	//
	// Matching is done against the parsed path of the request URI (no query string).
	// Example:
	// - original request URI: `/_matrix/client/r0/rooms/!AbCdEF%3Aexample.com/invite?something=here`
	// - parsed path: `/_matrix/client/r0/rooms/!AbCdEF:example.com/invite`
	HookMatchRuleTypeURLPath = "route"

	// HookMatchRuleTypeURLPath is a match rule type that requires a match against the full Matrix ID of the authenticated user.
	HookMatchRuleTypeMatrixUserID = "matrixUserID"
)

var knownHookMatchRuleTypes = []string{
	HookMatchRuleTypeHTTPMethod,
	HookMatchRuleTypeURLPath,
	HookMatchRuleTypeMatrixUserID,
}

type HookMatchRule struct {
	// Type specifies the type of this hook match rule.
	// This determines the value that matching will happen against.
	Type string `json:"type"`

	// Regex specifies the regular expression that needs to match.
	// To invert it (that is, to make the rule pass when there is no match), specify Invert = true
	Regex         string `json:"regex,omitempty"`
	regexCompiled *regexp.Regexp

	// Invert specifies whether this rule passes if we get a match or if we don't.
	// By default (Invert = false), it's a pass if there is a match.
	Invert bool `json:"invert"`
}

func (me *HookMatchRule) MatchesRequest(request *http.Request) bool {
	isMatch, err := me.matchRequestAgainstRules(request)
	if err != nil {
		// This should have been run during policy validation.
		// Now there's nothing we can do but fail hard.
		panic(err)
	}

	if me.Invert {
		isMatch = !isMatch
	}

	return isMatch
}

func (me *HookMatchRule) matchRequestAgainstRules(request *http.Request) (bool, error) {
	err := me.ensureInitialized()
	if err != nil {
		return false, err
	}

	if me.Type == HookMatchRuleTypeHTTPMethod {
		if !me.regexCompiled.MatchString(request.Method) {
			return false, nil
		}
	}

	if me.Type == HookMatchRuleTypeURLPath {
		if !me.regexCompiled.MatchString(request.URL.Path) {
			return false, nil
		}
	}

	if me.Type == HookMatchRuleTypeMatrixUserID {
		matrixUserIDInterface := request.Context().Value("userId")
		if matrixUserIDInterface != nil {
			matrixUserIDString := matrixUserIDInterface.(string)
			if !me.regexCompiled.MatchString(matrixUserIDString) {
				return false, nil
			}
		}
	}

	return true, nil
}

func (me *HookMatchRule) validate() error {
	if !util.IsStringInArray(me.Type, knownHookMatchRuleTypes) {
		return fmt.Errorf("%s is an invalid hook match rule type", me.Type)
	}

	err := me.ensureInitialized()
	if err != nil {
		return fmt.Errorf("Failed initialization for hook match rule (%s): %s", me, err)
	}

	return nil
}

func (me *HookMatchRule) ensureInitialized() error {
	if me.regexCompiled == nil {
		regex, err := regexp.Compile(me.Regex)
		if err != nil {
			return err
		}
		me.regexCompiled = regex
	}

	return nil
}
