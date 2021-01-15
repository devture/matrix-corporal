package handler

import (
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/policy"
	"devture-matrix-corporal/corporal/policy/provider"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type PolicyApiHandlerRegistrator struct {
	policyStore    *policy.Store
	policyProvider provider.Provider
}

func NewPolicyApiHandlerRegistrator(
	policyStore *policy.Store,
	policyProvider provider.Provider,
) *PolicyApiHandlerRegistrator {
	return &PolicyApiHandlerRegistrator{
		policyStore:    policyStore,
		policyProvider: policyProvider,
	}
}

func (me *PolicyApiHandlerRegistrator) RegisterRoutesWithRouter(router *mux.Router) {
	router.HandleFunc("/_matrix/corporal/policy", me.actionPolicyGet).Methods("GET")
	router.HandleFunc("/_matrix/corporal/policy", me.actionPolicyPut).Methods("PUT")
	router.HandleFunc("/_matrix/corporal/policy/provider/reload", me.actionPolicyProviderReload).Methods("POST")
}

func (me *PolicyApiHandlerRegistrator) actionPolicyGet(w http.ResponseWriter, r *http.Request) {
	// May be nil
	policy := me.policyStore.Get()

	Respond(w, http.StatusOK, map[string]interface{}{
		"policy": policy,
	})
}

func (me *PolicyApiHandlerRegistrator) actionPolicyPut(w http.ResponseWriter, r *http.Request) {
	var policy policy.Policy

	err := httphelp.GetJsonFromRequestBody(r, &policy)
	if err != nil {
		Respond(w, http.StatusBadRequest, ApiResponseError{
			ErrorCode:    ErrorCodeBadJson,
			ErrorMessage: "Bad body payload",
		})
		return
	}

	err = me.policyStore.Set(&policy)
	if err != nil {
		Respond(w, http.StatusOK, ApiResponseError{
			ErrorCode:    ErrorCodeUnknown,
			ErrorMessage: fmt.Sprintf("Failed to set policy: %s", err),
		})
		return
	}

	Respond(w, http.StatusOK, map[string]interface{}{})
}

func (me *PolicyApiHandlerRegistrator) actionPolicyProviderReload(w http.ResponseWriter, r *http.Request) {
	go me.policyProvider.Reload()

	Respond(w, http.StatusOK, map[string]interface{}{})
}

// Ensure interface is implemented
var _ httphelp.HandlerRegistrator = &PolicyApiHandlerRegistrator{}
