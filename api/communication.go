package api

import (
	"simplesurance/persistence"
	"time"
)

//TODO variable names - the type name is also silly
type communication struct {
	state                persistence.State
	exchangeTimestamp    chan time.Time
	exchangeRequestCount chan persistence.Cache
	exchangePersistence  chan PersistenceData
	exchangeAccumulated  chan int
}

//TODO document purpose of each communication channel
func NewCommunication() communication {
	return communication{
		exchangeTimestamp:    make(chan time.Time),
		exchangeRequestCount: make(chan persistence.Cache),
		exchangePersistence:  make(chan PersistenceData),
		exchangeAccumulated:  make(chan int),
	}
}

//TODO name
//this is the information that the persistence processor requires: a requestCount and a reference timestamp
type PersistenceData struct {
	RequestCount persistence.RequestCount
	Reference    persistence.RequestCount
}

func NewPersistenceData(cache persistence.Cache, timestamp time.Time) PersistenceData {
	return PersistenceData{
		RequestCount: cache.RequestCount,
		Reference: persistence.RequestCount{
			Timestamp: timestamp,
		},
	}
}

func (s *server) startCommunicationProcessor() {
	s.Logger.Print("Starting communication processor...")

	s.Logger.Print("Starting Persistence-Accumulated exchanger...")
	go func(com communication) {
		for {
			persistenceData, ok := <-com.exchangePersistence
			if ok {
				//TODO workflow wrapper?
				com.state.Past = com.state.Past.AppendToTail(persistenceData.RequestCount)
				com.state.Past = com.state.Past.UpdateTotals(persistenceData.Reference, s.persistenceTimeFrame)
				com.exchangeAccumulated <- com.state.Past.TotalAccumulatedRequestCount()
			} else {
				break
			}
		}
	}(s.Communication)

	s.Logger.Print("Starting Timestamp-RequestCount exchanger...")
	go func(com communication) {
		for {
			requestTimestamp, ok := <-com.exchangeTimestamp
			if ok {
				s.Logger.Printf("COM: received new requestTimestamp: '%v'\n", requestTimestamp.Format(time.RFC3339))

				if com.state.Present.Empty() {
					com.state.Present.Timestamp = requestTimestamp
					s.Logger.Print("COM: Initialized cache")
				}

				if com.state.Present.CompareTimestampWithPrecision(requestTimestamp, time.Second) {
					com.state.Present.Increment()
					s.Logger.Printf("COM: Incremented cached requestCount to '%v'\n", com.state.Present.Count)
				} else {
					persistenceUpdate := NewPersistenceData(com.state.Present, requestTimestamp)
					s.Logger.Printf("COM: Sending persistence Update :'%v'\n", persistenceUpdate)

					s.Communication.exchangePersistence <- persistenceUpdate
					totalAccumulated := <-s.Communication.exchangeAccumulated

					s.Logger.Printf("COM: Received new total accumulate of '%v'\n", totalAccumulated)

					com.state.Present = persistence.NewCache(requestTimestamp, totalAccumulated)
					s.Logger.Printf("COM: Updated cache to '%v'\n", com.state.Present)
				}

				com.exchangeRequestCount <- com.state.Present
			} else {
				break
			}
		}
	}(s.Communication)
	s.Logger.Print("Communication processor up and running")
}

func (s *server) CloseChannels() {
	close(s.Communication.exchangeRequestCount)
	close(s.Communication.exchangePersistence)
	close(s.Communication.exchangeAccumulated)
}
