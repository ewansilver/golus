package lus

/** A web based LUS based loosely on the Jini LUS that was around back in prehistoric times i.e start of the millenium
(See https://river.apache.org/doc/specs/html/lookup-spec.html or http://www.artima.com/jini/jiniology/lookup.html). It is a toy
example but allows me to learn Go - this is my Hello World on Go!

The core idea is that service providers register a set of name/value pairs that describe themselves with one or more LUS instances by POSTting a JSON
document along with a requested lease time. The LUS will accept the registration and returns a URL and a lease time (in ms) that it will hold onto
the service registration for. The service provider is then responsible for renewing the registration before the lease expires by PUTting a new lease
request. If it does not renew the lease then the LUS will drop the service registration.

Clients who want to make use of the service are able to look up suitably registered services by passing in a set of key/value pairs that describe
the characteristsics that they wish the service to provide. The example in the client.go harness is of two poller applications that register in
different environments (prod and dev). The LUS will return all service registrations that it knows about to any client provided there is a full
match of all the key/value pairs in the clients template and that the service lease has not expired.

This provides quite a nice mechanism for self healing as responsibility for maintaining its registration lies within the service providers own
control - if they die then their lease will not be renewed and it will be removed (eventually) after the lease has expired. To aid this time out
process the LUS may decide to reduce the lease time it actually accepts from the client. This means that if a service tries to ask for a very long lease
then it will likely not get it.

This approach to service discovery is essentially probabilistic in nature. It sits at odds with the current vogue in service discovery which is based
around maintaing a strongly consistent view of the services on the network.

To use, type: go run lus_server.go
**/

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

// Holds a HATEOAS style link relation that we can use later when people call into the webroot to try and use the services.
type LinkRelation struct {
	Rel  string
	Href string
}

// Internal struct to allow us to track when a particular Service will expire.
type entry_state struct {
	expiry  time.Time
	service Service
}

// A shitty internal structure that is overloaded with multiple use cases but represents the various inbound requests and means
// that I don't have to have multiple chans for each request type. Not sure what is most idiomatic here.
type Request struct {
	q                string
	response_channel chan response
	service          Service
	id               string
}

// As with request. It is the return value on all the chans.
type response struct {
	id      string
	lease   int64
	matches []Service
}

// Represents the JSON data struct that lets clients know that a service has been succesfully registered.
type Registration struct {
	Url   string
	Lease int64
}

// Start the Lus server
func Start(max_lease float64) chan Request {
	request_chan := make(chan Request)
	go lus(request_chan, max_lease)
	return request_chan
}

// The core goroutine. Maintains the internal map of entries and orchestrates the various activities.
// Kind of sucks that this has to be written within a for loop. Would much prefer to write it as a tail recursive call
// and pass in all the params but it seems that Go is not optimised for tail recursion. WTF!!!
// (eg see: https://groups.google.com/forum/#!msg/golang-nuts/0oIZPHhrDzY/2nCpUZDKZAAJ)
func lus(c chan Request, max_lease float64) {
	tick_chan := time.Tick(1 * time.Second)
	entries := make(map[string]entry_state)
	var counter int64 = 0

	for {
		select {
		case req := <-c:
			switch req.q {
			case "register": // Handles registration of new services
				id := createUniqueID(counter)
				counter++
				expiry_time, lease_duration := getExpiryAndLease(req.service, max_lease)
				entries[id] = entry_state{service: req.service, expiry: expiry_time}
				req.response_channel <- response{id: id, lease: lease_duration}
			case "renew": // Allows clients to renew service leases
				id := req.id
				e, ok := entries[id]
				if ok {
					expiry_time, lease_duration := getExpiryAndLease(req.service, max_lease)
					entries[id] = entry_state{service: e.service, expiry: expiry_time}
					req.response_channel <- response{id: id, lease: lease_duration}
				} else {
					req.response_channel <- response{} // Send an empty response to indicate nothing happened.
				}
			case "find": // Allows clients to find all the entries that match a particular set of keys.
				req.response_channel <- response{matches: findMatchingEntries(req.service.Keys, entries)}
			case "get_id": // Allows a client to find the specific entry.
				id := req.id
				e, ok := entries[id]
				if ok {
					m := map[string]entry_state{"key": e}
					r := response{matches: convertToServices(m)}
					req.response_channel <- r
				} else {
					req.response_channel <- response{} // Send an empty response to indicate nothing happened.
				}

			default:
				log.Println("**** stateful_routine DEFAULT. Shouldn't be here! :", req)
			}
		case <-tick_chan: // Cleans out stale entries.
			entries = filterBy(removeStaleEntries(), entries)
		}
	}
}

// Returns the new lease and the expiry time based on the requested_lease
func getExpiryAndLease(entry Service, max_lease float64) (time.Time, int64) {
	requested_lease := float64(entry.Lease)
	lease_duration := time.Duration(math.Min(requested_lease, max_lease))
	expiry_time := time.Now().Add(lease_duration * time.Millisecond)
	return expiry_time, inMilliseconds(lease_duration * time.Millisecond)
}

// Find the Services that match the supplied templates
func findMatchingEntries(templates map[string]string, entries map[string]entry_state) []Service {
	for k, v := range templates {
		entries = filterBy(matchesEntryState(k, v), entries)
	}
	return convertToServices(entries)
}

// Helper func that allows us to hack in a unique ID for every entry. Obviously this is deterministic but it is my first Go app so give me a break!
func createUniqueID(counter int64) string {
	data := strconv.AppendInt([]byte("Some random stuff that isn't really random but will do for our purposes...."), counter, 10)
	hasher := sha1.New()
	hasher.Write(data)
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

//
func convertToServices(entries map[string]entry_state) []Service {
	array := make([]Service, 0, len(entries))
	now := time.Now()

	for _, entry := range entries {
		expiry_time := entry.expiry
		remaining_lease := inMilliseconds(expiry_time.Sub(now)) // Get the remaining lease in milliseconds
		if remaining_lease > 0 {
			array = append(array, Service{Lease: remaining_lease, Data: entry.service.Data, Keys: entry.service.Keys, ID: entry.service.ID})
		}
	}
	return array
}

// Get a Duration in milliseconds.
func inMilliseconds(d time.Duration) int64 {
	return d.Nanoseconds() / 1e6
}

// Remove any entries that have expired. Is passed into filterBy
func removeStaleEntries() func(e entry_state) bool {
	now := time.Now()
	return func(e entry_state) bool {
		is_alive := now.Before(e.expiry)
		return is_alive
	}
}

// Finds all the entries that match the supplier key/value pair. Is passed into filterBy
func matchesEntryState(key string, value string) func(e entry_state) bool {
	return func(s entry_state) bool {
		service := s.service
		v, ok := service.Keys[key]
		if ok {
			return v == value
		}
		return false
	}
}

// Functional filter to return a new map of entries that match the supplier filter fun.
func filterBy(filter func(e entry_state) bool, maps map[string]entry_state) map[string]entry_state {
	response := make(map[string]entry_state)
	for key, entry := range maps {
		if filter(entry) {
			response[key] = entry
		}
	}
	return response
}

// Extract the Service that was passed over the wire as JSON from the http.Request.
func getService(b *http.Request) Service {
	body, err := ioutil.ReadAll(b.Body)
	if err != nil {
		panic("l")
	}
	var s Service
	err = json.Unmarshal(body, &s)
	if err != nil {
		panic(err)
	}
	return s
}

// The wrapper func that is called when clients want to register a new entry.
func Register(request_channel chan Request, port int, w http.ResponseWriter, r *http.Request) {
	service := getService(r)
	response_chan := make(chan response)
	request_struct := Request{q: "register", response_channel: response_chan, service: service}
	request_channel <- request_struct
	response := <-response_chan

	b, _ := json.Marshal(Registration{Url: "http://localhost:" + strconv.Itoa(port) + Entry_url() + response.id, Lease: response.lease})
	w.Write(b)
}

// The wrapper func that is called when clients either want to update (via PUT) or examine (via GET) a specific entry
func Entry(request_channel chan Request, port int, w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		path := r.URL.Path
		id := path[7:len(path)]
		response_chan := make(chan response)
		request_channel <- Request{q: "renew", response_channel: response_chan, service: getService(r), id: id}
		response := <-response_chan
		b, _ := json.Marshal(Registration{Url: "http://localhost:" + strconv.Itoa(port) + Entry_url() + response.id, Lease: response.lease})
		w.Write(b)
	} else if r.Method == "GET" {
		path := r.URL.Path
		id := path[7:len(path)]
		response_chan := make(chan response)
		request_channel <- Request{q: "get_id", response_channel: response_chan, id: id}
		response := <-response_chan
		b, _ := json.Marshal(response.matches)
		w.Write(b)
	} else {
		panic("Wrong method")
	}
}

// Wrapper func that is called to allow clients to find all Entries that match the supplied Entry JSON.
func Find(request_channel chan Request, w http.ResponseWriter, r *http.Request) {
	response_chan := make(chan response)
	request_channel <- Request{q: "find", response_channel: response_chan, service: getService(r)}
	response := <-response_chan
	b, _ := json.Marshal(response.matches)
	w.Write(b)
}

// An example of a HATEOAS webroot that will allow us to alter the exact URLS called for register etc in a later iteration.
func Root_handler(port int, w http.ResponseWriter, r *http.Request) {
	rels := []LinkRelation{LinkRelation{Href: "http://localhost:" + strconv.Itoa(port) + "/register", Rel: "http://rels.ewansilver.com/v1/lus/register"}, LinkRelation{Href: "http://localhost:" + strconv.Itoa(port) + "/find", Rel: "http://rels.ewansilver.com/v1/lus/find"}}
	b, _ := json.Marshal(rels)
	w.Write(b)
}

// Helper func to allow us to replace all the entry urls easily.
func Entry_url() string {
	return "/entry/"
}
