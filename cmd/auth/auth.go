package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/bolkedebruin/rdpgw/cmd/auth/database"
	"github.com/bolkedebruin/rdpgw/cmd/auth/ntlm"
	"github.com/bolkedebruin/rdpgw/shared/auth"
	"github.com/thought-machine/go-flags"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

const (
	protocol = "unix"
)

var opts struct {
	SocketAddr string `short:"s" long:"socket" default:"/tmp/rdpgw-auth.sock" description:"the location of the socket"`
	ConfigFile string `short:"c" long:"conf" default:"rdpgw-auth.yaml" description:"users config file (yaml)"`
}

type AuthServiceImpl struct {
	auth.UnimplementedAuthenticateServer

	database database.Database
	ntlm     *ntlm.NTLMAuth
}

var _ auth.AuthenticateServer = (*AuthServiceImpl)(nil)

func NewAuthService(database database.Database) auth.AuthenticateServer {
	s := &AuthServiceImpl{
		database: database,
		ntlm:     ntlm.NewNTLMAuth(database),
	}
	return s
}

func (s *AuthServiceImpl) Authenticate(ctx context.Context, message *auth.UserPass) (*auth.AuthResponse, error) {
	r := &auth.AuthResponse{}
	storedPassword := s.database.GetPassword(message.Username)
	if storedPassword == "" || storedPassword != message.Password {
		log.Printf("Authentication for user: %s failed", message.Username)
		r.Error = "Authentication failure"
		return r, nil
	}

	log.Printf("User: %s authenticated", message.Username)
	r.Authenticated = true
	return r, nil
}

func (s *AuthServiceImpl) NTLM(ctx context.Context, message *auth.NtlmRequest) (*auth.NtlmResponse, error) {
	r, err := s.ntlm.Authenticate(message)

	if err != nil {
		log.Printf("[%s] NTLM failed: %s", message.Session, err)
	} else if r.Authenticated {
		log.Printf("[%s] User: %s authenticated using NTLM", message.Session, r.Username)
	} else if r.NtlmMessage != "" {
		log.Printf("[%s] Sending NTLM challenge", message.Session)
	}

	return r, err
}

func listenUnixSocket(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create auth socket directory: %w", err)
	}
	if err := removeStaleUnixSocket(path); err != nil {
		return nil, err
	}

	oldUmask := syscall.Umask(0077)
	listener, err := func() (net.Listener, error) {
		defer syscall.Umask(oldUmask)
		return net.Listen(protocol, path)
	}()
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("restrict auth socket permissions: %w", err)
	}
	return listener, nil
}

func removeStaleUnixSocket(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat auth socket: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to remove non-socket auth socket path %q", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale auth socket: %w", err)
	}
	return nil
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		var fErr *flags.Error
		if errors.As(err, &fErr) {
			if fErr.Type == flags.ErrHelp {
				fmt.Printf("Acknowledgements:\n")
				fmt.Printf(" - This product includes software developed by the Thomson Reuters Global Resources. (go-ntlm - https://github.com/m7913d/go-ntlm - BSD-4 License)\n")
			}
		}
		return
	}

	log.Printf("Starting auth server on %s", opts.SocketAddr)
	listener, err := listenUnixSocket(opts.SocketAddr)
	if err != nil {
		log.Fatal(err)
	}
	server := grpc.NewServer()
	db, err := database.NewConfigFile(opts.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}
	service := NewAuthService(db)
	auth.RegisterAuthenticateServer(server, service)
	server.Serve(listener)
}
