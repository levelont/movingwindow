package persistence

import (
	"time"
)

/* The cache structure is in charge of keeping track of:
- all incoming requests for the current timestamp and within the precision of the algorithm - currently hard-coded to
  one second. This is done with the fields of the RequestCount structure.
- the total accumulated requests within the persistence timeframe - configurable via the environment variable.
  This is done with the additional 'GlobalCount' field.
*/
type Cache struct {
	RequestCount
	GlobalCount int
}

/* A new cache will be created when a new request comes in with a timestamp that differs by at least one unit of the
current precision of the algorithm. When that occurs, a new cache shall hold the new timestamp and the global count of
requests within the persistence timeframe, using the new timestamp as a reference. Refer to 'UpdateTotals()' for details.
*/
func NewCache(timestamp time.Time, totalAccumulated int) Cache {
	requestCount := RequestCount{Timestamp: timestamp}
	requestCount.Increment()
	return Cache{
		RequestCount: requestCount,
		GlobalCount:  totalAccumulated + 1,
	}
}

/* Counters for the current timestamp and the global ammount of requests are handled independently of each other
 */
func (c *Cache) Increment() {
	c.RequestCount.Increment()
	c.GlobalCount++
}
