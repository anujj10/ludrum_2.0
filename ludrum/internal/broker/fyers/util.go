package fyers

import "encoding/json"

func decodeMap(raw string) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func stringValue(value interface{}) string {
	text, _ := value.(string)
	return text
}

func jsonUnmarshal[T any](raw string, target *T) error {
	return json.Unmarshal([]byte(raw), target)
}
