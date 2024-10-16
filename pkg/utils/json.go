package utils

import (
	"encoding/json"
	"log/slog"
	"reflect"
)

func TestJsonEquals(first, second []byte) (bool, error) {
	slog.Warn("TestJsonEquals => this method is not reliable if data is not equally sorted, because it performs a DeepEqual which compares item one-by-one. Use it only if you are sure that data is sorted equally.")
	var firstDec, secondDec interface{}
	err := json.Unmarshal(first, &firstDec)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(second, &secondDec)
	if err != nil {
		return false, err
	}

	return reflect.DeepEqual(firstDec, secondDec), nil
}
