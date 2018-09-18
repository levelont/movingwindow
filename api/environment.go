package api

import (
	"flag"
	"time"
)

/* Configuration of the application retrieved via environment variables:
- ListenAddress: port on which the server will be listening
- PersistenceFile: destination file on disk for serialization of state upon incoming interrupt signals
- PersistenceTimeFrame: duration of the moving window for which total incoming requests will be calculated
*/
type Environment struct {
	ListenAddress              string
	PersistenceFile            string
	PersistenceTimeFrame       string
	ParsedPersistenceTimeFrame time.Duration
}

/* Parsing of command line flags to set environment values.
If missing, defaults will be provided.
Errors parsing the provided timeframe will crash the server.
*/
func ParseEnvironment() Environment {
	var env Environment
	flag.StringVar(&env.ListenAddress, "listen-address", ":5000", "Server listen address")
	flag.StringVar(&env.PersistenceTimeFrame, "persistence-time-interval", "60s", "Time frame for which request counts will be calculated")
	flag.StringVar(&env.PersistenceFile, "persistence-file", "persistence.bin", "File to which state will be persisted upon server termination")
	flag.Parse()

	var err error
	env.ParsedPersistenceTimeFrame, err = time.ParseDuration(env.PersistenceTimeFrame)
	if err != nil {
		panic(err) //OK: need env variable to be parsable.
	}

	return env
}
