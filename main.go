package main

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

To use, type: go run main.go
**/

import (
  "log"
  "net/http"
  "golus/lus"
  "strconv"
)

// Main func to get the system up and running.
// Lots of hardcoded params that could do with being moved out.
func main() {
  port := 3000
  var max_lease float64 = 120000 // Arbitrarily set it to 2 minutes (miiliseconds)

  log.Println("LUS Server")
  log.Println("Running on http://localhost:", port)
  request_chan := lus.Start(max_lease)

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { lus.Root_handler(port, w, r) })
  http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) { lus.Register(request_chan, port, w, r) })
  http.HandleFunc("/find", func(w http.ResponseWriter, r *http.Request) { lus.Find(request_chan, w, r) })
  http.HandleFunc(lus.Entry_url(), func(w http.ResponseWriter, r *http.Request) { lus.Entry(request_chan, port, w, r) })
  http.ListenAndServe(":"+strconv.Itoa(port), nil)
}
