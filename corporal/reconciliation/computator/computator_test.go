package computator

import (
	"devture-matrix-corporal/corporal/connector"
	"devture-matrix-corporal/corporal/policy"
	"devture-matrix-corporal/corporal/reconciliation"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

type TestData struct {
	CurrentState        connector.CurrentState `json:"currentState"`
	Policy              policy.Policy          `json:"policy"`
	ReconciliationState reconciliation.State   `json:"reconciliationState"`
}

func TestReconciliationStateComputation(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.json")
	if err != nil {
		panic(err)
	}

	logger := logrus.New()
	logger.Out = ioutil.Discard

	reconciliationComputator := NewReconciliationStateComputator(logger)

	for _, testPath := range matches {
		testPath := testPath //make local
		fileName := filepath.Base(testPath)
		testTitle := strings.Replace(fileName, ".json", "", 1)

		t.Run(testTitle, func(t *testing.T) {
			t.Parallel()

			f, err := os.Open(testPath)
			if err != nil {
				t.Errorf("Failed to open file: %s: %s", testPath, err)
				return
			}
			defer f.Close()

			bytes, err := ioutil.ReadAll(f)
			if err != nil {
				t.Errorf("Failed reading from file: %s: %s", testPath, err)
				return
			}

			var testData TestData
			err = json.Unmarshal(bytes, &testData)
			if err != nil {
				t.Errorf("Failed to decode JSON from file: %s: %s", testPath, err)
				return
			}

			computedReconciliationState, err := reconciliationComputator.Compute(
				&testData.CurrentState,
				&testData.Policy,
			)
			if err != nil {
				t.Errorf("Failed to compute reconciliation state for file: %s: %s", testPath, err)
				return
			}

			err = determineReconciliationStateMismatchError(&testData.ReconciliationState, computedReconciliationState)
			if err != nil {
				t.Errorf(
					"Unexpected reconciliation state for: %s: %s. \nExpected:\n%#v\n\nComputed:\n%#v",
					testPath,
					err,
					testData.ReconciliationState.Actions,
					computedReconciliationState.Actions,
				)
				return
			}
		})
	}
}

func determineReconciliationStateMismatchError(expected, computed *reconciliation.State) error {
	for idx, expectedAction := range expected.Actions {
		if len(computed.Actions)-1 < idx {
			return fmt.Errorf(
				"Expected a %s action at index %d, but got nothing",
				expectedAction.Type,
				idx,
			)
		}

		computedAction := computed.Actions[idx]

		err := determineUserActionMismatchError(expectedAction, computedAction)
		if err != nil {
			return fmt.Errorf(
				"Action mismatch at index %d: %s",
				idx,
				err,
			)
		}
	}

	//There's another possibility. That we've computed more actions than were expected.
	if len(computed.Actions) > len(expected.Actions) {
		return fmt.Errorf(
			"Expected %d actions, but computed %d",
			len(expected.Actions),
			len(computed.Actions),
		)
	}

	return nil
}

func determineUserActionMismatchError(expected, computed *reconciliation.StateAction) error {
	if expected.Type != computed.Type {
		return fmt.Errorf(
			"Expected type (%s) different than computed (%s)",
			expected.Type,
			computed.Type,
		)
	}

	if expected.Type == reconciliation.ActionUserCreate {
		passwordComputed, err := computed.GetStringPayloadDataByKey("password")
		if err != nil {
			return fmt.Errorf("Did not expect computed %s action to not have a password payload: %s", reconciliation.ActionUserCreate, err)
		}

		passwordExpected, err := expected.GetStringPayloadDataByKey("password")
		if err != nil {
			return fmt.Errorf("Did not expect expected %s action to not have a password payload: %s", reconciliation.ActionUserCreate, err)
		}

		if passwordExpected == "__RANDOM__" {
			if len(passwordComputed) != 128 {
				return fmt.Errorf("Expected a randomly-generated 128-character password, got: %s", passwordComputed)
			}
			return nil
		}

		if passwordExpected != passwordComputed {
			return fmt.Errorf("Expected password %s, got %s", passwordExpected, passwordComputed)
		}

		return nil
	}

	// TODO - we can validate other actions in more detail

	return nil
}
