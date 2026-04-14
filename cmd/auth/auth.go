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
	cleanup := func() {
		if _, err := os.Stat(opts.SocketAddr); err == nil {
			if err := os.RemoveAll(opts.SocketAddr); err != nil {
				log.Fatal(err)
			}
		}
	}
	cleanup()

	oldUmask := syscall.Umask(0)
	listener, err := net.Listen(protocol, opts.SocketAddr)
	syscall.Umask(oldUmask)
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
