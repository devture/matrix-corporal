package policy

import (
	"devture-matrix-corporal/corporal/matrix"
	"fmt"
)

type Validator struct {
	homeserverDomainName string
}

func NewValidator(homeserverDomainName string) *Validator {
	return &Validator{
		homeserverDomainName: homeserverDomainName,
	}
}

func (me *Validator) Validate(policy *Policy) error {
	if policy.SchemaVerson != 1 {
		return fmt.Errorf("Found policy with schema version (%d) that we do not support", policy.SchemaVerson)
	}

	for _, userId := range policy.GetManagedUserIds() {
		if !matrix.IsFullUserIdOfDomain(userId, me.homeserverDomainName) {
			return fmt.Errorf(
				"Policy user `%s` is not hosted on the managed homeserver domain (%s)",
				userId,
				me.homeserverDomainName,
			)
		}
	}

	for idx, userPolicy := range policy.User {
		err := userPolicy.Validate()
		if err != nil {
			return fmt.Errorf(
				"User policy validation for `%s` (index %d) failed: %s",
				userPolicy.Id,
				idx,
				err,
			)
		}
	}

	hookIDToIndexMap := make(map[string]int)

	for idx, hook := range policy.Hooks {
		existingIndex, exists := hookIDToIndexMap[hook.ID]
		if exists {
			return fmt.Errorf(
				"Hook at index `%d` (ID = %s) has the same ID as the hook at index %d. Assign unique hook IDs to prevent confusion",
				idx,
				hook.ID,
				existingIndex,
			)
		}

		err := hook.Validate()
		if err != nil {
			return fmt.Errorf(
				"Hook at index `%d` (ID = %s) is invalid: %s",
				idx,
				hook.ID,
				err,
			)
		}

		hookIDToIndexMap[hook.ID] = idx
	}

	return nil
}
