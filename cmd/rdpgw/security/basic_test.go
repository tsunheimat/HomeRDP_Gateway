package security

import (
	"context"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/protocol"
	"testing"
)

var (
	info = protocol.Tunnel{
		RDGId:        "myid",
		TargetServer: "my.remote.server",
		RemoteAddr:   "10.0.0.1",
	}

	hosts = []string{"localhost:3389", "my-{{ preferred_username }}-host:3389"}
)

func TestCheckHost(t *testing.T) {
	t.Cleanup(func() {
		ManagedHostList = nil
	})

	info.User = identity.NewUser()
	info.User.SetUserName("MYNAME")

	ctx := context.WithValue(context.Background(), protocol.CtxTunnel, &info)

	Hosts = hosts
	ManagedHostList = func() ([]string, error) {
		return hosts, nil
	}

	// check any
	HostSelection = "any"
	host := "localhost:3389"
	if ok, err := CheckHost(ctx, host); !ok || err != nil {
		t.Fatalf("%s should be allowed with host selection %s (err: %s)", host, HostSelection, err)
	}
	host = "try.my.server:3389"
	if ok, err := CheckHost(ctx, host); ok || err == nil {
		t.Fatalf("%s should NOT be allowed with host selection %s (err: %s)", host, HostSelection, err)
	}

	HostSelection = "signed"
	if ok, err := CheckHost(ctx, host); ok || err == nil {
		t.Fatalf("signed host selection isnt supported at the moment")
	}

	HostSelection = "roundrobin"
	if ok, err := CheckHost(ctx, host); ok {
		t.Fatalf("%s should NOT be allowed with host selection %s (err: %s)", host, HostSelection, err)
	}

	host = "my-MYNAME-host:3389"
	if ok, err := CheckHost(ctx, host); !ok {
		t.Fatalf("%s should be allowed with host selection %s (err: %s)", host, HostSelection, err)
	}

}

func TestCheckHostUsesManagedHostList(t *testing.T) {
	t.Cleanup(func() {
		ManagedHostList = nil
	})

	info.User = identity.NewUser()
	info.User.SetUserName("MYNAME")

	ctx := context.WithValue(context.Background(), protocol.CtxTunnel, &info)

	Hosts = []string{"legacy.internal:3389"}
	ManagedHostList = func() ([]string, error) {
		return []string{"managed.internal:3389"}, nil
	}
	HostSelection = "roundrobin"

	if ok, err := CheckHost(ctx, "managed.internal:3389"); !ok || err != nil {
		t.Fatalf("managed host should be allowed, ok=%v err=%v", ok, err)
	}
	if ok, err := CheckHost(ctx, "legacy.internal:3389"); ok || err == nil {
		t.Fatalf("legacy host should be ignored when managed host list is configured, ok=%v err=%v", ok, err)
	}
}

func TestCheckHostFailsClosedWithoutManagedHostList(t *testing.T) {
	t.Cleanup(func() {
		ManagedHostList = nil
	})

	info.User = identity.NewUser()
	info.User.SetUserName("MYNAME")

	ctx := context.WithValue(context.Background(), protocol.CtxTunnel, &info)

	Hosts = []string{"legacy.internal:3389"}
	ManagedHostList = nil
	HostSelection = "roundrobin"

	if ok, err := CheckHost(ctx, "legacy.internal:3389"); ok || err == nil {
		t.Fatalf("host auth should fail closed without a managed host list, ok=%v err=%v", ok, err)
	}
}

func TestCheckHostIgnoresAnyModeWhenManagedHostsExist(t *testing.T) {
	t.Cleanup(func() {
		ManagedHostList = nil
	})

	info.User = identity.NewUser()
	info.User.SetUserName("MYNAME")

	ctx := context.WithValue(context.Background(), protocol.CtxTunnel, &info)

	ManagedHostList = func() ([]string, error) {
		return []string{"managed.internal:3389"}, nil
	}
	HostSelection = "any"

	if ok, err := CheckHost(ctx, "managed.internal:3389"); !ok || err != nil {
		t.Fatalf("managed host should be allowed in any mode, ok=%v err=%v", ok, err)
	}
	if ok, err := CheckHost(ctx, "unmanaged.internal:3389"); ok || err == nil {
		t.Fatalf("unmanaged host should be rejected even in any mode, ok=%v err=%v", ok, err)
	}
}
