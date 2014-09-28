package main

/**
  A dummy test client that just allows me to test the basic functionality of the LUS. Longer term this code needs to be extracted into
  some kind of LookupLocator (https://river.apache.org/doc/api/net/jini/core/discovery/LookupLocator.html) and corresponding LookupDiscoveryService
  (https://river.apache.org/doc/api/net/jini/discovery/LookupDiscoveryService.html)

  The example below simply demo's a couple of  apps being deployed to different environments and then allowing clients to discover those apps. It
  also demo's lease time out and renewal.

  I haven't spent anytime looking at Go test harnesses yet as this is simply my first app and I wanted to see how it works.

  To use: make sure the lus_server is running on the localmachine and then type: go run lus_client.go
**/

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

// Represents the JSON data structure that is being passed over the wire to register a service.
type Entry_struct struct {
	Lease int64
	Data  string
	Keys  map[string]string
}

// Represents the JSON data struct that lets clients know that a service has been succesfully registered.
type Register_response struct {
	Url   string
	Lease int64
}

// Represents the JSON data struct that lets clients ask to extend a lease registration.
type Renew_request struct {
	Lease int64
}

// Main test func that has a set of hard coded URLs against whihc the lus.go can be run.
// Later on we need to add dynamic service discovery and HATEOAS.
func main() {
	log.Println("LUS Client. Running tests")
	find_url := "http://localhost:3000/find"
	register_url := "http://localhost:3000/register"
	var lease int64 = 10000

	a := <-find(map[string]string{"application": "poller"}, find_url)
	assert_num_entries("a", a, 0)

	b := <-register(map[string]string{"application": "poller", "environment": "prod", "id": "b"}, lease, "", register_url)
	c := <-register(map[string]string{"application": "poller", "environment": "dev", "id": "c"}, lease, "", register_url)

	d := <-find(map[string]string{"application": "poller"}, find_url)
	assert_num_entries("d", d, 2)

	e := <-find(map[string]string{"application": "poller", "environment": "prod"}, find_url)
	assert_num_entries("e", e, 1)

	f := <-find(map[string]string{"application": "poller", "environment": "dev"}, find_url)
	assert_num_entries("f", f, 1)

	// Make sure that calling the specific entry URL gives you the appropriate Entry_struct back.
	bb := <-get(b.Url)
	assert_num_entries("bb", bb, 1)
	b_entry := bb[0]
	assert_contains("application", "poller", b_entry.Keys)
	assert_contains("environment", "prod", b_entry.Keys)
	assert_contains("id", "b", b_entry.Keys)

	// Wait for 5 seconds, renew the dev poller, check that there are still 2 entries
	time.Sleep(5 * time.Second)
	g := <-find(map[string]string{"application": "poller"}, find_url)
	assert_num_entries("g", g, 2)
	<-renew(c.Url, lease)

	// Wait for 5 seconds and check that the prod poller has gone and that only the dev poller is still active
	time.Sleep(5 * time.Second)
	h := <-find(map[string]string{"application": "poller"}, find_url)
	assert_num_entries("h", h, 1)

	// Wait for 5 seconds and check that everything has timed out
	time.Sleep(5 * time.Second)
	i := <-find(map[string]string{"application": "poller"}, find_url)
	assert_num_entries("i", i, 0)

	// Check that GETing an expired entry fails.
	bc := <-get(b.Url)
	assert_num_entries("bc", bc, 0)

	// Check that the lease on an entry is capped.

	j := <-register(map[string]string{"application": "some_other_app"}, 99999999999, "", register_url)
	if !(j.Lease == 120000) {
		panic("j_entry lease is not capped")
	}

	log.Println("Everything checks out.")
}

//A simple test routine to check that we are getting valid responses.
func assert_contains(key string, value string, m map[string]string) bool {
	v, _ := m[key]
	if !strings.EqualFold(v, value) {
		panic("Assert contains failed.")
	}
	return true
}

func assert_num_entries(test string, entries []Entry_struct, num int) bool {
	ok := num == len(entries)
	log.Println("assert_num_entries:", ok)
	if !ok {
		log.Println("***** Num entries does not match what it is supposed to. Test:", test)
		panic("Assert_num-entries failed.")
	}
	return ok
}

func get(url string) chan []Entry_struct {
	response_channel := make(chan []Entry_struct)
	go get_http(response_channel, url)
	return response_channel
}

func renew(url string, lease int64) chan Register_response {
	response_channel := make(chan Register_response)
	r := Renew_request{Lease: lease}
	go renew_http(r, response_channel, url)
	return response_channel
}

func register(keys map[string]string, lease int64, data string, url string) chan Register_response {
	response_channel := make(chan Register_response)
	e := Entry_struct{Keys: keys, Lease: lease, Data: data}
	go register_http(e, response_channel, url)
	return response_channel
}

func find(keys map[string]string, url string) chan []Entry_struct {
	response_channel := make(chan []Entry_struct)
	e := Entry_struct{Keys: keys}
	go find_http(e, response_channel, url)
	return response_channel
}

func get_http(response_channel chan []Entry_struct, url string) {
	body := get_to_server(url)
	response_channel <- get_entries(body)
}

func renew_http(r Renew_request, response_channel chan Register_response, url string) {
	json, _ := json.Marshal(r)
	body := json_to_server(bytes.NewBuffer(json), url, "PUT")
	response_channel <- get_register_response(body)
}

func register_http(e Entry_struct, response_channel chan Register_response, url string) {
	json, _ := json.Marshal(e)
	body := json_to_server(bytes.NewBuffer(json), url, "POST")
	response_channel <- get_register_response(body)
}

func find_http(e Entry_struct, response_channel chan []Entry_struct, url string) {
	json, _ := json.Marshal(e)
	body := json_to_server(bytes.NewBuffer(json), url, "POST")
	response_channel <- get_entries(body)
}

func get_to_server(url string) []byte {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}

func json_to_server(json io.Reader, url string, method string) []byte {
	req, err := http.NewRequest(method, url, json)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}

func get_register_response(body []byte) Register_response {
	response := Register_response{}
	json.Unmarshal(body, &response)
	return response
}

func get_entries(body []byte) []Entry_struct {
	entries := []Entry_struct{}
	json.Unmarshal(body, &entries)
	return entries
}
