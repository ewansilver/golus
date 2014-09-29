package lus

/**
The example below simply demo's a couple of  apps being deployed to different environments and then allowing clients to discover those apps. It
also demo's lease time out and renewal.

I haven't spent anytime looking at Go test harnesses yet as this is simply my first app and I wanted to see how it works.

To use: make sure the lus_server is running on the localmachine and then type: go test lus
**/

import (
  "testing"
  "log"
  "time"
  "strings"
  )


func TestMaxLeaseUpdate(t *testing.T) {
  var requested_lease int64 = 999999999
  var max_lease float64 = 10000
  _, lease := get_expiry_and_lease(Entry_struct{Lease:requested_lease}, max_lease)
  assert_int64(lease, int64(max_lease), t)
  }

func TestRequestedLeaseUpdate(t *testing.T) {
  var requested_lease int64 = 1000
  var max_lease float64 = 100000
  _, lease := get_expiry_and_lease(Entry_struct{Lease:requested_lease}, max_lease)
  assert_int64(lease, requested_lease, t)
  }


// Main test func that has a set of hard coded URLs against whihc the lus.go can be run.
// Later on we need to add dynamic service discovery and HATEOAS.
func TestMatcher(t *testing.T) {
  log.Println("LUS Client. Running tests")
  find_url := "http://localhost:3000/find"
  register_url := "http://localhost:3000/register"
  var lease int64 = 10000

  a := <-find_chan(map[string]string{"application": "poller"}, find_url)
  assert_num_entries("a", a, 0)

  b := <-register_chan(map[string]string{"application": "poller", "environment": "prod", "id": "b"}, lease, "", register_url)
  c := <-register_chan(map[string]string{"application": "poller", "environment": "dev", "id": "c"}, lease, "", register_url)

  d := <-find_chan(map[string]string{"application": "poller"}, find_url)
  assert_num_entries("d", d, 2)

  e := <-find_chan(map[string]string{"application": "poller", "environment": "prod"}, find_url)
  assert_num_entries("e", e, 1)

  f := <-find_chan(map[string]string{"application": "poller", "environment": "dev"}, find_url)
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
  g := <-find_chan(map[string]string{"application": "poller"}, find_url)
  assert_num_entries("g", g, 2)
  <-renew_chan(c.Url, lease)

  // Wait for 5 seconds and check that the prod poller has gone and that only the dev poller is still active
  time.Sleep(5 * time.Second)
  h := <-find_chan(map[string]string{"application": "poller"}, find_url)
  assert_num_entries("h", h, 1)

  // Wait for 5 seconds and check that everything has timed out
  time.Sleep(5 * time.Second)
  i := <-find_chan(map[string]string{"application": "poller"}, find_url)
  assert_num_entries("i", i, 0)

  // Check that GETing an expired entry fails.
  bc := <-get(b.Url)
  assert_num_entries("bc", bc, 0)

  // Check that the lease on an entry is capped.

  j := <-register_chan(map[string]string{"application": "some_other_app"}, 99999999999, "", register_url)
  if !(j.Lease == 120000) {
    panic("j_entry lease is not capped")
  }

  log.Println("Everything checks out.")
}


func assert_int64(value ,assertion int64, t *testing.T) {
  if !(value == assertion) {
  t.Fatalf("Value %v is different to assertion %v", value,assertion)
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

func assert_num_entries(test string, entries []Entry_struct, num int) bool {
  ok := num == len(entries)
  log.Println("assert_num_entries:", ok)
  if !ok {
    log.Println("***** Num entries does not match what it is supposed to. Test:", test)
    panic("Assert_num-entries failed.")
  }
  return ok
}
