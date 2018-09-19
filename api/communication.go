package api

import (
	"simplesurance/persistence"
	"time"
)

/*A member of the server struct, the communication struct contains the state object that keeps track of all
requests and all necessary channels to interact with it:
- exchangeTimestamp: used by the index handler to send timestamps of new incoming requests to the communication processor
- exchangeRequestCount: used by the communication processor to notify the index handler of computed request totals
- exchangePersistence: used internally by the communication processor
- exchangeAccumulated: used internally by the communication processor
*/
type communication struct {
	state                persistence.State
	exchangeTimestamp    chan time.Time
	exchangeRequestCount chan persistence.Cache
	exchangePersistence  chan persistenceData
	exchangeAccumulated  chan int
}

func NewCommunication() communication {
	return communication{
		exchangeTimestamp:    make(chan time.Time),
		exchangeRequestCount: make(chan persistence.Cache),
		exchangePersistence:  make(chan persistenceData),
		exchangeAccumulated:  make(chan int),
	}
}

/* The communication processor uses PersistenceData internally as a means to exchange information between its goroutines.
- RequestCount: accumulated request count for the last unit of time
- Reference: object containing the timestamp that will be used for calculation of request counts within the persistence timeframe.
*/
type persistenceData struct {
	RequestCount persistence.RequestCount
	Reference    persistence.RequestCount
}

func NewPersistenceData(cache persistence.Cache, timestamp time.Time) persistenceData {
	return persistenceData{
		RequestCount: cache.RequestCount,
		Reference: persistence.RequestCount{
			Timestamp: timestamp,
		},
	}
}

/* The communication processor acts as a synchronizer for all attempts to modify request counts held in memory, so that they
are serialized and served correctly. For that purpose, it spawns two goroutines, the Persistence-Accumulated exchanger and
the Timestamp-RequestCount exchanger. For every new request that comes in, the flow goes as follows:

- Client sends a request

  ->  IndexHandler sends the timestamp

        -> Timestamp-RequestCount exchanger compares timestamp with cache on basis of the algorithm precision

	    -> If cache and incoming timestamp are considered to be in the same point in time by the algorithm precision,
	       the cache is increased on top without any further calculations. The result of the increase is returned to the
	    <- IndexHandler

	    -> Else, the received timestamp is passed to the Persistence-Accumulated exchanger along with the previous
	       cache values to calculate new request counts within the persistence time frame

	        -> Persistence-Accumulated exchanger sends old cache values and new timestamp to the request count calculator
	           and the total request count for the persistence timeframe - taking the new timestamp as the reference,
	        <- back to the Timestamp-RequestCount exchanger

	   Timestamp-RequestCount initializes a new cache with the timestamp and the received total request count and sends
        <- this values back to the indexHandler

  <-  IndexHandler produces the response and sends it to the client

- Client receives the response
*/
func (s *server) startCommunicationProcessor() {
	s.Logger.Print("Starting communication processor...")

	s.Logger.Print("Starting Persistence-Accumulated exchanger...")
	go func() {
		for {
			persistenceData, ok := <-s.Communication.exchangePersistence
			if ok {
				s.Communication.state.Past = s.Communication.state.Past.AppendToTail(persistenceData.RequestCount)
				s.Communication.state.Past = s.Communication.state.Past.UpdateTotals(persistenceData.Reference, s.persistenceTimeFrame, s.precision)
				s.Communication.exchangeAccumulated <- s.Communication.state.Past.TotalAccumulatedRequestCount()
			} else {
				break
			}
		}
	}()

	s.Logger.Print("Starting Timestamp-RequestCount exchanger...")
	go func() {
		for {
			requestTimestamp, ok := <-s.Communication.exchangeTimestamp
			if ok {
				s.Logger.Printf("COM: received new requestTimestamp: '%v'\n", requestTimestamp.Format(time.RFC3339))

				if s.Communication.state.Present.Empty() {
					s.Communication.state.Present.Timestamp = requestTimestamp
					s.Logger.Print("COM: Initialized cache")
				}

				if s.Communication.state.Present.CompareTimestampWithPrecision(requestTimestamp, s.precision) {
					s.Communication.state.Present.Increment()
					s.Logger.Printf("COM: Incremented cached requestCount to '%v'\n", s.Communication.state.Present.Count)
				} else {
					persistenceUpdate := NewPersistenceData(s.Communication.state.Present, requestTimestamp)
					s.Logger.Printf("COM: Sending persistence Update :'%v'\n", persistenceUpdate)

					s.Communication.exchangePersistence <- persistenceUpdate
					totalAccumulated := <-s.Communication.exchangeAccumulated

					s.Logger.Printf("COM: Received new total accumulate of '%v'\n", totalAccumulated)

					s.Communication.state.Present = persistence.NewCache(requestTimestamp, totalAccumulated)
					s.Logger.Printf("COM: Updated cache to '%v'\n", s.Communication.state.Present)
				}

				s.Communication.exchangeRequestCount <- s.Communication.state.Present
			} else {
				break
			}
		}
	}()
	s.Logger.Print("Communication processor up and running")
}

/* Cleanup for shutdown of the server.
 */
func (s *server) CloseChannels() {
	close(s.Communication.exchangeRequestCount)
	close(s.Communication.exchangePersistence)
	close(s.Communication.exchangeAccumulated)
}
