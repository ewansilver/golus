package lus

/**
  Test that the Client interface works as expected.
**/

import (
	"testing"
	"time"
)

func root_url() string { return "http://localhost:3000/" }

// Test the range of client functions.
func TestClient(t *testing.T) {
	var lease int64 = 10000

	client := NewClient(root_url())
	assert_strings_match("http://localhost:3000/", client.Root_URL())

	// First we need to make sure that the LUS is empty
	a := client.Find(map[string]string{"application": "poller"})
	assert_num_entries("a", a, 0)

	serviceB := NewService(map[string]string{"application": "poller", "environment": "prod", "id": "b"}, lease, "", "b123")
	serviceC := NewService(map[string]string{"application": "poller", "environment": "dev", "id": "c"}, lease, "", "c456")
	// Then we can register a couple of templates
	b := client.Register(serviceB)
	c := client.Register(serviceC)

	// Make sure that calling the specific entry URL gives you the appropriate Service back.
	bb := client.Find(map[string]string{"id": "b"})
	cc := client.Find(map[string]string{"id": "c"})

	assert_num_entries("bb", bb, 1)
	b_entry := bb[0]
	assert_contains("application", "poller", b_entry.Keys)
	assert_contains("environment", "prod", b_entry.Keys)
	assert_contains("id", "b", b_entry.Keys)
	assert_id(b_entry, "b123")

	assert_num_entries("cc", cc, 1)
	c_entry := cc[0]
	assert_contains("application", "poller", c_entry.Keys)
	assert_contains("environment", "dev", c_entry.Keys)
	assert_contains("id", "c", c_entry.Keys)
	assert_id(c_entry, "c456") // This should fail as we are expecting "c456"

	// Then we need to make sure that both entries are present
	d := client.Find(map[string]string{"application": "poller"})
	assert_num_entries("d", d, 2)

	// Check that we can remove one of the entries (we do this by setting its lease to zero which basically means that it gets pulled out of the LUS)
	client.Renew(b.Url, 0)

	// Then we need to make sure that only one entry is present
	e := client.Find(map[string]string{"application": "poller"})
	assert_num_entries("e", e, 1)

	// Finally renew with a zero second lease to make sure that we have removed the entry from the LUS and it does not break any other tests.
	// Need to solve how to start up and shut down LUS services.
	client.Renew(c.Url, 0)

	// Finally we check that we have cleared everything out of the lus
	f := client.Find(map[string]string{"application": "poller"})
	assert_num_entries("f", f, 0)

}

// Test the leases are capped.
func TestCappedLease(t *testing.T) {
	client := NewClient(root_url())

	// First we need to make sure that the LUS is empty
	a := client.Find(map[string]string{"application": "poller"})
	assert_num_entries("a", a, 0)

	serviceB := NewService(map[string]string{"application": "poller", "environment": "prod", "id": "b"}, 99999999999, "", "anID")
	b := client.Register(serviceB)
	client.Renew(b.Url, 0)
	if !(b.Lease == 120000) {
		panic("Lease was not capped.")
	}

}

// Test that the auto renewal function works.
func TestAutoRenewal(t *testing.T) {
	var lease int64 = 1000
	client := NewClient(root_url())

	// First we need to make sure that the LUS is empty
	a := client.Find(map[string]string{"application": "poller"})
	assert_num_entries("a", a, 0)

	serviceB := NewService(map[string]string{"application": "poller", "environment": "prod", "id": "b"}, lease, "", "anID")

	b := client.Register(serviceB)
	client.Auto_renew(b)
	// Lets wait 5 seconds by which time the entry should have timed out. We want to make sure that it is still there and
	// so show that it has been autorenewed
	time.Sleep(2500 * time.Millisecond)

	c := client.Find(map[string]string{"application": "poller"})
	assert_num_entries("c", c, 1)
	// Finally stop renew and check that everything dies out.
	client.Halt_renew(b)
	time.Sleep(1000 * time.Millisecond)

	d := client.Find(map[string]string{"application": "poller"})
	assert_num_entries("d", d, 0)
}

func assert_strings_match(a, b string) {
	if !(a == b) {
		panic("assert_strings_match do not match")
	}
}

func assert_id(entry Service, id string) {
	if !(entry.ID == id) {
		panic("assert_id does not match")
	}
}
