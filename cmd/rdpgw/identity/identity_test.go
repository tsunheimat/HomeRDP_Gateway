package identity

import (
	"log"
	"reflect"
	"testing"
)

func TestMarshalling(t *testing.T) {
	u := NewUser()
	u.SetUserName("ANAME")
	u.SetAuthenticated(true)
	u.SetDomain("DOMAIN")

	c := NewUser()
	data, err := u.Marshal()
	if err != nil {
		log.Fatalf("Cannot marshal %s", err)
	}

	err = c.Unmarshal(data)
	if err != nil {
		t.Fatalf("Error while unmarshalling: %s", err)
	}

	if u.UserName() != c.UserName() || u.Authenticated() != c.Authenticated() || u.Domain() != c.Domain() {
		t.Fatalf("identities not equal: %+v != %+v", u, c)
	}
}

func TestGroupsRoundTrip(t *testing.T) {
	u := NewUser()
	u.SetGroups([]string{"rdpgw-admins", "homelab-users", "rdpgw-admins", ""})

	if !u.InGroup("rdpgw-admins") {
		t.Fatalf("expected user to be in group rdpgw-admins")
	}

	expected := []string{"homelab-users", "rdpgw-admins"}
	if !reflect.DeepEqual(u.Groups(), expected) {
		t.Fatalf("expected groups %v, got %v", expected, u.Groups())
	}

	data, err := u.Marshal()
	if err != nil {
		t.Fatalf("cannot marshal user: %v", err)
	}

	c := NewUser()
	if err := c.Unmarshal(data); err != nil {
		t.Fatalf("cannot unmarshal user: %v", err)
	}

	if !reflect.DeepEqual(c.Groups(), expected) {
		t.Fatalf("expected unmarshaled groups %v, got %v", expected, c.Groups())
	}
}
