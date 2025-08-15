package reconciliation

import "fmt"

type State struct {
	Actions []*StateAction `json:"actions"`
}

type StateAction struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

func (me *StateAction) GetStringPayloadDataByKey(key string) (string, error) {
	data, err := me.GetPayloadDataByKey(key)
	if err != nil {
		return "", err
	}

	dataCasted, castOk := data.(string)
	if !castOk {
		return "", fmt.Errorf("failed casting payload data for: %s", key)
	}
	return dataCasted, nil
}

func (me *StateAction) GetIntPayloadDataByKey(key string) (int, error) {
	data, err := me.GetPayloadDataByKey(key)
	if err != nil {
		return 0, err
	}

	dataCasted, castOk := data.(int)
	if !castOk {
		return 0, fmt.Errorf("failed casting payload data for: %s", key)
	}
	return dataCasted, nil
}

func (me *StateAction) GetPayloadDataByKey(key string) (interface{}, error) {
	data, exists := me.Payload[key]
	if !exists {
		return nil, fmt.Errorf("missing %s payload data", key)
	}
	return data, nil
}
