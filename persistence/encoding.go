package persistence

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
)

/* The total amount of requests of the system can only be obtained together with the counter - which keeps past data
within the persistence time frame, and the current cached data - which keeps accumulated, request counts for the present
point in time according to the precision of the algorithm.
*/
type State struct {
	Past    RequestCounter
	Present Cache
}

/*A request counter is, in terms of data, just two pointers. However, they represent a list of nodes. When serialising
state, the data of all those nodes need to be extracted from memory and persisted to disk. This internal structure
acts as an intermediate step during serialization of state to ensure that all data is writen to the destination file.
Fields need be exported for encoding purposes
*/
type internalState struct {
	Past    requestCountList
	Present Cache
}

/* Converts state to its internalState representation and encodes it into a stream of bytes.
 */
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

/* Decodes the provided buffer into the intermediate internalState representation and builds a requestCounter with the
resulting data.
*/
func decodeState(buffer []byte) (State, error) {
	var decodedInternalState internalState
	d := gob.NewDecoder(bytes.NewBuffer(buffer))
	err := d.Decode(&decodedInternalState)
	if err != nil {
		return State{}, err
	}

	decodedState := State{
		Past:    decodedInternalState.Past.ToRequestCounter(),
		Present: decodedInternalState.Present,
	}

	return decodedState, nil
}

/* Resulting file will only be readable and writable by the current user
 */
func (s State) WriteToFile(path string) error {
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
