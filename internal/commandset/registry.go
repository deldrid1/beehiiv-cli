package commandset

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Multiple    bool   `json:"multiple"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type Operation struct {
	OperationID           string      `json:"operation_id"`
	Command               []string    `json:"command"`
	Method                string      `json:"method"`
	Path                  string      `json:"path"`
	RequiresPublicationID bool        `json:"requires_publication_id"`
	PathParams            []string    `json:"path_params"`
	QueryParams           []Parameter `json:"query_params"`
	Body                  bool        `json:"body"`
	List                  bool        `json:"list"`
	Pagination            string      `json:"pagination"`
	Summary               string      `json:"summary"`
	Description           string      `json:"description"`
}

//go:embed operations.json
var operationsJSON []byte

var (
	loadOnce    sync.Once
	loadErr     error
	operations  []Operation
	groups      []string
	groupToOps  map[string][]Operation
	commandToOp map[string]Operation
)

func mustLoad() {
	loadOnce.Do(func() {
		loadErr = json.Unmarshal(operationsJSON, &operations)
		if loadErr != nil {
			return
		}

		groupToOps = make(map[string][]Operation, len(operations))
		commandToOp = make(map[string]Operation, len(operations))

		groupSet := make(map[string]struct{})
		for _, operation := range operations {
			if len(operation.Command) != 2 {
				loadErr = fmt.Errorf("operation %q has unsupported command depth %d", operation.OperationID, len(operation.Command))
				return
			}

			group := operation.Command[0]
			commandKey := strings.Join(operation.Command, " ")
			if _, exists := commandToOp[commandKey]; exists {
				loadErr = fmt.Errorf("duplicate command %q", commandKey)
				return
			}

			groupSet[group] = struct{}{}
			groupToOps[group] = append(groupToOps[group], operation)
			commandToOp[commandKey] = operation
		}

		groups = make([]string, 0, len(groupSet))
		for group := range groupSet {
			groups = append(groups, group)
		}
		sort.Strings(groups)

		for group, ops := range groupToOps {
			sort.SliceStable(ops, func(i, j int) bool {
				return ops[i].Command[1] < ops[j].Command[1]
			})
			groupToOps[group] = ops
		}
	})
}

func All() ([]Operation, error) {
	mustLoad()
	if loadErr != nil {
		return nil, loadErr
	}

	cloned := make([]Operation, len(operations))
	copy(cloned, operations)
	return cloned, nil
}

func Groups() ([]string, error) {
	mustLoad()
	if loadErr != nil {
		return nil, loadErr
	}

	cloned := make([]string, len(groups))
	copy(cloned, groups)
	return cloned, nil
}

func OperationsForGroup(group string) ([]Operation, error) {
	mustLoad()
	if loadErr != nil {
		return nil, loadErr
	}

	ops := groupToOps[group]
	cloned := make([]Operation, len(ops))
	copy(cloned, ops)
	return cloned, nil
}

func Find(group, action string) (Operation, bool, error) {
	mustLoad()
	if loadErr != nil {
		return Operation{}, false, loadErr
	}

	operation, ok := commandToOp[group+" "+action]
	return operation, ok, nil
}

func GroupExists(group string) (bool, error) {
	mustLoad()
	if loadErr != nil {
		return false, loadErr
	}

	_, ok := groupToOps[group]
	return ok, nil
}
