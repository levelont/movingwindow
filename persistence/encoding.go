package main

import (
	"bytes"
	"encoding/gob"
	"log"
)

func encode(requestCounts map[string]int) []byte {
	b := new(bytes.Buffer)

	e := gob.NewEncoder(b)

	// Encoding the map
	err := e.Encode(requestCounts)
	if err != nil {
		//TODO Logging + error handling
		log.Fatal(err)
	}

	return b.Bytes()
}

func decode(buffer []byte) map[string]int {
	var decodedMap map[string]int
	d := gob.NewDecoder(bytes.NewBuffer(buffer))

	// Decoding the serialized data
	err := d.Decode(&decodedMap)
	if err != nil {
		//TODO Logging + error handling
		log.Fatal(err)
	}

	return decodedMap
}
