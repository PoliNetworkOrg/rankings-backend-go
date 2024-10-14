package main

import (
	"encoding/json"
	"log/slog"
	"reflect"
	"sort"
	"time"
)

type Brand int

const (
	Mercedes Brand = iota
	Audi
)

var brandName = map[Brand]string{
	Audi:     "audi",
	Mercedes: "mercedes",
}

func (b Brand) String() string {
	return brandName[b]
}

type Car struct {
	Brand       Brand     `json:"brand"`
	Model       string    `json:"model"`
	Optionals   []string  `json:"optionals"`
	Color       string    `json:"color"`
	Cost        uint      `json:"cost"`
	IsAvailable bool      `json:"isAvailable"`
	ReleaseDate time.Time `json:"releaseDate"`
}

func carEquals(a Car, b Car) bool {
	sort.Strings(a.Optionals)
	sort.Strings(b.Optionals)

	return reflect.DeepEqual(a, b)
}

func JsonEncodeDecodeStruct() {
	car1 := Car{
		Brand:       Audi,
		Model:       "A1",
		Optionals:   []string{"a/c", "rear-camera", "ads", "touchscreen"},
		Cost:        18_000,
		Color:       "red",
		IsAvailable: true,
		ReleaseDate: time.Date(2023, 10, 02, 0, 0, 0, 0, time.UTC),
	}

	car2 := Car{
		Color:       "red",
		Model:       "A1",
		Cost:        18_000,
		IsAvailable: true,
		Brand:       Audi,
		ReleaseDate: time.Date(2023, 10, 02, 0, 0, 0, 0, time.UTC),
		Optionals:   []string{"rear-camera", "a/c", "ads", "touchscreen"},
	}

	carB1, err := json.Marshal(car1)
	if err != nil {
		panic(err)
	}

	carB2, err := json.Marshal(car2)
	if err != nil {
		panic(err)
	}

	slog.Info("encode", "car1", string(carB1), "car2", string(carB2))

	carE1 := Car{}
	json.Unmarshal(carB1, &carE1)

	carE2 := Car{}
	json.Unmarshal(carB2, &carE2)

	slog.Info("decode the encoded", "car1", carE1, "car2", carE2, "equals", carEquals(carE1, carE2))
}
