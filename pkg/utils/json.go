package utils

import (
	"encoding/json"
	"os"
	"reflect"
)

func TestJsonEquals(firstPath string, secondPath string) (bool, error) {
	first, err := os.ReadFile(firstPath)
	if err != nil  {
		return false, err
	}

	second, err := os.ReadFile(secondPath)
	if err != nil  {
		return false, err
	}

	var firstDec, secondDec interface{}
	err = json.Unmarshal(first, &firstDec)
	if err != nil  {
		return false, err
	}

	err = json.Unmarshal(second, &secondDec)
	if err != nil  {
		return false, err
	}

	return reflect.DeepEqual(firstDec, secondDec), nil
}
