package provider

import (
	"devture-matrix-corporal/corporal/policy"
	"encoding/json"
)

func createPolicyFromJsonBytes(data []byte) (*policy.Policy, error) {
	var policy policy.Policy
	err := json.Unmarshal(data, &policy)
	if err != nil {
		return nil, err
	}

	return &policy, nil
}
