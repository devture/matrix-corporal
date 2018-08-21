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
	data, err := me.getPayloadDataByKey(key)
	if err != nil {
		return "", err
	}

	dataCasted, castOk := data.(string)
	if !castOk {
		return "", fmt.Errorf("Failed casting payload data for: %s", key)
	}
	return dataCasted, nil
}

func (me *StateAction) getPayloadDataByKey(key string) (interface{}, error) {
	data, exists := me.Payload[key]
	if !exists {
		return nil, fmt.Errorf("Missing %s payload data", key)
	}
	return data, nil
}
