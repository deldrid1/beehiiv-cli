package commandset

import "testing"

func TestRegistryLoadsWithoutDuplicateCommands(t *testing.T) {
	t.Parallel()

	ops, err := All()
	if err != nil {
		t.Fatalf("All returned error: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("All returned no operations")
	}
}

func TestExpectedNormalizedCommandsExist(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		group       string
		action      string
		operationID string
	}{
		{group: "publications", action: "get", operationID: "publications_show"},
		{group: "poll-responses", action: "list", operationID: "polls_list_responses"},
		{group: "workspaces", action: "publications-by-subscription-email", operationID: "workspaces_publications-by-subscription-email"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.group+"_"+testCase.action, func(t *testing.T) {
			operation, found, err := Find(testCase.group, testCase.action)
			if err != nil {
				t.Fatalf("Find returned error: %v", err)
			}
			if !found {
				t.Fatalf("operation %s %s not found", testCase.group, testCase.action)
			}
			if operation.OperationID != testCase.operationID {
				t.Fatalf("operation id = %q, want %q", operation.OperationID, testCase.operationID)
			}
		})
	}
}
