package persistence

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
)

type State struct {
	Past    RequestCountDoublyLinkedList
	Present Cache
}

/*Fields need be exported for encoding purposes
 */
type internalState struct {
	Past    requestCountList
	Present Cache
}

func (s State) encode() ([]byte, error) {
	internalState := internalState{
		Past:    s.Past.getNodes(),
		Present: s.Present,
	}
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)
	err := e.Encode(internalState)
	if err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}

func decodeState(buffer []byte) (State, error) {
	var decodedInternalState internalState
	d := gob.NewDecoder(bytes.NewBuffer(buffer))
	err := d.Decode(&decodedInternalState)
	if err != nil {
		return State{}, err
	}

	decodedState := State{
		Past:    decodedInternalState.Past.BuildDoublyLinkedList(),
		Present: decodedInternalState.Present,
	}

	return decodedState, nil
}

func (s State) WriteToFile(path string) error {
	//TODO WriteFile gets a 'fileName'. Should that not be an absolute path?
	bytes, err := s.encode()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, bytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

func ReadFromFile(path string) (State, error) {
	//TODO WriteFile gets a 'fileName'. Should that not be an absolute path?
	readBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return State{}, err
	}

	state, err := decodeState(readBytes)
	if err != nil {
		return State{}, err
	}

	return state, nil
}
