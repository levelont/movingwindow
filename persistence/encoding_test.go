package main

import (
	"testing"
	"io/ioutil"
	"log"
	"reflect"
)

func TestEncoding(t *testing.T) {
	var expected = map[string]int{"one":1, "two":2, "three":3}
	bytes := encode(expected)

	fileName := "RequestCounts.txt"
	err := ioutil.WriteFile(fileName, bytes, 0600)
	if err != nil {
		log.Fatal(err)
	}

	readBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
	}

	result := decode(readBytes)

	if ! reflect.DeepEqual(expected, result) {
		t.Errorf("Expected maps to be equal, found '%+v' != '%+v'\n", expected, result)
	}
}
