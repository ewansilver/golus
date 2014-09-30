package lus

/**
  Test that the Client interface works as expected.
**/

import (
	"testing"
)

func root_url() string { return "http://localhost:3000/" }

func TestClient(t *testing.T) {
	var lease int64 = 10000

	client := NewClient(root_url())
	assert_strings_match("http://localhost:3000/", client.Root_URL())

	// First we need to make sure that the LUS is empty
	a := client.Find(map[string]string{"application": "poller"})
	assert_num_entries("a", a, 0)

	// Then we can register a couple of templates
	b := client.Register(map[string]string{"application": "poller", "environment": "prod", "id": "b"}, lease, "")
	c := client.Register(map[string]string{"application": "poller", "environment": "dev", "id": "c"}, lease, "")

	// Make sure that calling the specific entry URL gives you the appropriate Entry_struct back.
	bb := client.Find(map[string]string{"id": "b"})
	cc := client.Find(map[string]string{"id": "c"})

	assert_num_entries("bb", bb, 1)
	b_entry := bb[0]
	assert_contains("application", "poller", b_entry.Keys)
	assert_contains("environment", "prod", b_entry.Keys)
	assert_contains("id", "b", b_entry.Keys)

	assert_num_entries("cc", cc, 1)
	c_entry := cc[0]
	assert_contains("application", "poller", c_entry.Keys)
	assert_contains("environment", "dev", c_entry.Keys)
	assert_contains("id", "c", c_entry.Keys)

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

func assert_strings_match(a, b string) {
	if !(a == b) {
		panic("assert_strings_match do not match")
	}
}
