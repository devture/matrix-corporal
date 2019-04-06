package policy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type PermissionAssertment struct {
	Type               string                 `json:"type"`
	Payload            map[string]interface{} `json:"payload"`
	Allowed            bool                   `json:"allowed"`
	ExpectationComment string                 `json:"expectationComment"`
}

type TestData struct {
	Policy                Policy                 `json:"policy"`
	PermissionAssertments []PermissionAssertment `json:"permissionAssertments"`
}

func TestPolicyPermissionAssertment(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.json")
	if err != nil {
		panic(err)
	}

	checker := NewChecker()

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

			err = determinePolicyPermissionError(testData.Policy, checker, testData.PermissionAssertments)
			if err != nil {
				t.Errorf(
					"Policy permission failure in %s: %s",
					testPath,
					err,
				)
				return
			}
		})
	}
}

func determinePolicyPermissionError(policy Policy, checker *Checker, assertments []PermissionAssertment) error {
	for _, assertment := range assertments {
		err := checkAssertment(policy, checker, assertment)
		if err != nil {
			return fmt.Errorf("%s (rule expecation: %s)", err, assertment.ExpectationComment)
		}
	}
	return nil
}

func checkAssertment(policy Policy, checker *Checker, assertment PermissionAssertment) error {
	if assertment.Type == "leaveRoom" {
		userId := assertment.Payload["userId"].(string)
		roomId := assertment.Payload["roomId"].(string)

		allowed := checker.CanUserLeaveRoom(policy, userId, roomId)

		if allowed == assertment.Allowed {
			return nil
		}

		return fmt.Errorf("Expected %t status for user %s being able to leave room %s", assertment.Allowed, userId, roomId)
	}

	if assertment.Type == "leaveCommunity" {
		userId := assertment.Payload["userId"].(string)
		communityId := assertment.Payload["communityId"].(string)

		allowed := checker.CanUserLeaveCommunity(policy, userId, communityId)

		if allowed == assertment.Allowed {
			return nil
		}

		return fmt.Errorf("Expected %t status for user %s being able to leave community %s", assertment.Allowed, userId, communityId)
	}

	return fmt.Errorf("Unknown policy assertment type: %s", assertment.Type)
}
