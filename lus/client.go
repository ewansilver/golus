package lus

/**
  A dummy test client that just allows me to test the basic functionality of the LUS. Longer term this code needs to be extracted into
  some kind of LookupLocator (https://river.apache.org/doc/api/net/jini/core/discovery/LookupLocator.html) and corresponding LookupDiscoveryService
  (https://river.apache.org/doc/api/net/jini/discovery/LookupDiscoveryService.html)
**/

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

// Represents the JSON data struct that lets clients ask to extend a lease registration.
type Renew_request struct {
	Lease int64
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
