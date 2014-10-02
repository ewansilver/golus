package lus

/**
The example below simply demo's a couple of  apps being deployed to different environments and then allowing clients to discover those apps. It
also demo's lease time out and renewal.

I haven't spent anytime looking at Go test harnesses yet as this is simply my first app and I wanted to see how it works.

To use: make sure the lus_server is running on the localmachine and then type: go test lus
**/

import (
	"log"
	"strings"
	"testing"
)

func TestMaxLeaseUpdate(t *testing.T) {
	var requested_lease int64 = 999999999
	var max_lease float64 = 10000
	_, lease := getExpiryAndLease(Service{Lease: requested_lease}, max_lease)
	assert_int64(lease, int64(max_lease), t)
}

func TestRequestedLeaseUpdate(t *testing.T) {
	var requested_lease int64 = 1000
	var max_lease float64 = 100000
	_, lease := getExpiryAndLease(Service{Lease: requested_lease}, max_lease)
	assert_int64(lease, requested_lease, t)
}

func assert_int64(value, assertion int64, t *testing.T) {
	if !(value == assertion) {
		t.Fatalf("Value %v is different to assertion %v", value, assertion)
	}
}

//A simple test routine to check that we are getting valid responses.
func assert_contains(key string, value string, m map[string]string) bool {
	v, _ := m[key]
	if !strings.EqualFold(v, value) {
		panic("Assert contains failed.")
	}
	return true
}

func assert_num_entries(test string, entries []Service, num int) bool {
	ok := num == len(entries)
	log.Println("assert_num_entries:", ok)
	if !ok {
		log.Println("***** Num entries does not match what it is supposed to. Test:", test)
		panic("Assert_num-entries failed.")
	}
	return ok
}
