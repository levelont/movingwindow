# Moving window request counter

*Using only the standard library, create a Go HTTP server that on each request responds with a counter of the total number of requests that it has received during the previous 60 seconds (moving window).*
*The server should continue to the return the correct numbers after restarting it.*

# How to run

After cloning the repository, run from the project directory:

    Lak@Lak-PC MINGW64 ~/go/src/simplesurance (master)
    $ go run main.go
    http: 2018/09/19 03:48:28 Server is starting...
    http: 2018/09/19 03:48:28 Server is ready to handle requests at :5000
    
# Configuration

You can use the following flags to configure the functionality of the application:

    --listen-address:        Server will be available at this listen address
                             Default: ":5000"
    --persistence-timeframe: Time frame of the moving window for which total request counts will be calculated
                             Default: "60s"                   
    --precision:             Server precision. Timestamps that differ by this amount will be considered to be equal. This enhances caching.
                             Default: "100ms"
                             
For details on the format of `--persistence-timeframe` and `--precision`, please refer to the [Golang documentation on ParseDuration](https://golang.org/pkg/time/#ParseDuration).

# Responses

All requests will be handled by the same handler and will return a `requestCount` value encoded in JSON:

    Lak@Lak-PC MINGW64 ~/go/src/simplesurance (master)
    $ curl -s -X GET http://localhost:5000/
    {"requestCount":4}

# Testing

Most of the functions and functionality have tests covering them.
The `http_test` might be the most interesting of them all, allowing to test the handling of thousands of requests simultaneously and model different scenarios for moving window requests.
For details, refer to the file itself: it has extensive documentation.

# Implementation and design details

## Initial thoughts

At the very beginning, the first solution concept involved as simple a list of all incoming timestamps, calculating which of them were within the persistence timeframe of a reference - the most recent timestamp, and calculating the totals by traversing the entire list.
After considering a high-performance scenario in which thousands of requests could reach the server at the same time, the need for caching and reducing the number of times that the entire list would be traversed seemed to be a must.
That is when I came up with the concept of precision: if two timestamps were to be considered equal by a precision p, then it would not be necessary to traverse the list for them. A local cache could be simply increased.
As soon as a timestamp would arrive that is seen as different from the cache, then the calculation would take place. The precision could then play a huge role in delaying calculations and saving computations.
As to the calculation of totals itself, I realised that with every new incoming request, the amount of data kept in memory can be trimmed to save up usage. The new request will determine the new persistence time frame. Depending on how long it took the new request to arrive, it may be that several requests kept in memory are no longer within that timeframe and should be removed.

## Caching

Going back to the analogy of the incoming timestamps to be a simple list and considering that it would hold request timestamps in chronological order of arrival, one can see that the two operations modify different ends of the structure: adding a new request is an append operation to the end of the list, while trimming the list pops out elements from the front.
It is with that realisation that I decided to hold the data in a doubly linked list; not only because it provides constant O(1) access to both the head and the tail, but also because there are no additional considerations to be made in terms of allocating capacity for new items, or reshuffling elements once the list has been modified by either removing old requests or allocating more space.
Once I settled for the structure type, I decided that each of its nodes would represent an already processed request. For that purpose, it should hold three bits of information:

- timestamp of request arrival
- accumulated request count for requests that 'arrive at the same time' - that is, that are interpreted by the program as being the same timestamp after considering the precision factor
- accumulated request count for all request, starting from a reference, all the way down to this node.

Wrapped into a `RequestCount` structure, this forms the basis of both the doubly linked list nodes for accumulated totals and the cache for requests that arrive at a single point in time.
For proper caching, both are required: requests within the precision range will be held in a 'local' cache structure that needs no further calculations than simple counter increases. As soon as a request comes in with an effective new timestamp (taking into account the precision factor), the cache will be transformed into the tail of the doubly linked list, and all of the structure's data will be updated from tail to head, stopping at the first node that no longer falls within the new persistence time frame - which is now defined by the last received timestamp.
The local cache, then, holds the request counts for the 'Present', defined by the last received timestamp and the precision factor, whereas the doubly linked list holds the 'Past'
The first draft of the algorithm in pseudo-code can be found in the drafter.md file.

An important question to address is which precision factor provides a good balance between caching and 'real-time' results. I settled for 100ms, which is the default value for the flag.

## Testing

Another motivation for configurable precision in the program was testing: if the precision could be set to a relatively large duration for tests, the modelling behaviours of incoming requests with delays in between could be done reliably.
Failing to do so would be very fragile, as delays in the time magnitude of milliseconds are very susceptible to load spikes, which makes the tests unreliable.
A set of consistent tests can be found in `http_test.go`. Please refer to its documentation for details on how the scenarios are modelled and can be extended, and how those tests are turned into choreographed requests and corresponding assertions.
FYI: the test suite takes 20s to complete.

## Synchronization

With the initial hypothesis that thousands of requests could come in at the very same time, we need proper synchronization.
The concurrency model of golang makes it easy to serialize all accesses to the application's state: no (explicit) mutexes are required.
A communication processor takes care of handling these exchanges. Refer to the documentation at `api/communication.go` for details.

## Persistence

A web server is meant to run forever, but interruptions may occur. A signal manager - implemented as a goroutine forever running in the background and spawned by the main goroutine, detects interruptions and triggers serialization of the application's state.
During server initialization, this file - if found, will be read to bring back the old state to a new runtime environment.
As indicated by the corresponding function in the `server.go` file:

    Keep in mind: by the time the server has been restarted, the persisted values read and a new request is to be handled, the persisted request counts might no longer be in the persistence time frame. In that case, they will be discarded for the next request count computation.

# Future work

- [ ] better logging
- [ ] automated persistence tests
- [ ] signal handling test
- [ ] message queue between goroutines for easy testing of their interactions