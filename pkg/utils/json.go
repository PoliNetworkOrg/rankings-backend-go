package utils

import (
	"encoding/json"
	"reflect"
)

func TestJsonEquals(first, second []byte) (bool, error) {
	var firstDec, secondDec interface{}
	err := json.Unmarshal(first, &firstDec)
	if err != nil  {
		return false, err
	}

	err = json.Unmarshal(second, &secondDec)
	if err != nil  {
		return false, err
	}

	return reflect.DeepEqual(firstDec, secondDec), nil
}
