package persistence

import (
	"time"
)

//TODO name
//this is the information that the persistence processor requires: a requestCount and a reference timestamp
type PersistenceData struct {
	RequestCount RequestCount
	Reference    RequestCount
}

func NewPersistenceData(cache Cache, timestamp time.Time) PersistenceData {
	return PersistenceData{
		RequestCount: cache.RequestCount,
		Reference: RequestCount{
			Timestamp: timestamp,
		},
	}
}
