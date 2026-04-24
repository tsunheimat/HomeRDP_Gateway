package kdcproxy

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	krbconfig "github.com/bolkedebruin/gokrb5/v8/config"
)

func testProxy(kdcs ...string) KerberosProxy {
	return KerberosProxy{krb5Config: &krbconfig.Config{
		LibDefaults: krbconfig.LibDefaults{DefaultRealm: "EXAMPLE.COM"},
		Realms: []krbconfig.Realm{{
			Realm: "EXAMPLE.COM",
			KDC:   kdcs,
		}},
	}}
}

func TestForwardReturnsWhenNoKDCConnectionStarts(t *testing.T) {
	proxy := testProxy("127.0.0.1:bad-port")

	done := make(chan error, 1)
	go func() {
		_, err := proxy.forward("EXAMPLE.COM", []byte{0, 0, 0, 1, 0x6a})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error when all KDC dials fail")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("forward blocked when no KDC connection started")
	}
}

func TestForwardDoesNotPanicOnShortUDPPayload(t *testing.T) {
	proxy := testProxy("127.0.0.1:9")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("forward panicked on short UDP payload: %v", r)
		}
	}()

	if _, err := proxy.forward("EXAMPLE.COM", []byte{1, 2, 3}); err == nil {
		t.Fatal("expected an error for a short UDP payload")
	}
}

func TestForwardWaitsOnlyForStartedReplyGoroutines(t *testing.T) {
	response := []byte{0, 0, 0, 2, 0xaa, 0xbb}
	addr, closeServer := startTCPKDC(t, response)
	defer closeServer()

	proxy := testProxy("127.0.0.1:bad-port", addr)

	done := make(chan struct {
		resp []byte
		err  error
	}, 1)
	go func() {
		resp, err := proxy.forward("EXAMPLE.COM", []byte{0, 0, 0, 1, 0x6a})
		done <- struct {
			resp []byte
			err  error
		}{resp: resp, err: err}
	}()

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("forward returned unexpected error: %v", result.err)
		}
		if !bytes.Equal(result.resp, response) {
			t.Fatalf("forward response = %v, want %v", result.resp, response)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("forward blocked waiting for replies from KDCs that were never started")
	}
}

func TestForwardClosesStartedConnections(t *testing.T) {
	response := []byte{0, 0, 0, 1, 0xcc}
	clientClosed := make(chan struct{})

	addr, closeServer := startTCPKDCWithHandler(t, func(conn net.Conn) {
		defer conn.Close()
		if _, err := io.CopyN(io.Discard, conn, 5); err != nil {
			return
		}
		_, _ = conn.Write(response)
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
		close(clientClosed)
	})
	defer closeServer()

	proxy := testProxy(addr)
	resp, err := proxy.forward("EXAMPLE.COM", []byte{0, 0, 0, 1, 0x6a})
	if err != nil {
		t.Fatalf("forward returned unexpected error: %v", err)
	}
	if !bytes.Equal(resp, response) {
		t.Fatalf("forward response = %v, want %v", resp, response)
	}

	select {
	case <-clientClosed:
	case <-time.After(2 * time.Second):
		t.Fatal("started KDC connection was not closed")
	}
}

func startTCPKDC(t *testing.T, response []byte) (string, func()) {
	t.Helper()
	return startTCPKDCWithHandler(t, func(conn net.Conn) {
		defer conn.Close()
		_, _ = io.CopyN(io.Discard, conn, 5)
		_, _ = conn.Write(response)
	})
}

func startTCPKDCWithHandler(t *testing.T, handler func(net.Conn)) (string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handler(conn)
		}
	}()

	return ln.Addr().String(), func() {
		_ = ln.Close()
		<-done
	}
}
