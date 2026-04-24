package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bolkedebruin/rdpgw/cmd/auth/database"
	"github.com/bolkedebruin/rdpgw/cmd/auth/ntlm"
	"github.com/bolkedebruin/rdpgw/shared/auth"
	"github.com/thought-machine/go-flags"
	"google.golang.org/grpc"
)

const (
	protocol             = "unix"
	defaultSocketMode    = 0600
	defaultSocketDirMode = 0700
)

var opts struct {
	SocketAddr    string `short:"s" long:"socket" default:"/tmp/rdpgw-auth.sock" description:"the location of the socket"`
	SocketMode    string `long:"socket-mode" default:"0600" description:"octal permissions for the auth socket; owner read/write required, world access rejected (use 0660 with --socket-group for cross-user access)"`
	SocketDirMode string `long:"socket-dir-mode" default:"0700" description:"octal permissions for a newly-created auth socket directory; owner rwx required, world access rejected"`
	SocketOwner   string `long:"socket-owner" description:"user name or numeric UID to own the auth socket (optional)"`
	SocketGroup   string `long:"socket-group" description:"group name or numeric GID to own the auth socket (optional; pair with --socket-mode 0660 for group access)"`
	ConfigFile    string `short:"c" long:"conf" default:"rdpgw-auth.yaml" description:"users config file (yaml)"`
}

type unixSocketOptions struct {
	Mode    fs.FileMode
	DirMode fs.FileMode
	Owner   string
	Group   string
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

func defaultUnixSocketOptions() unixSocketOptions {
	return unixSocketOptions{
		Mode:    defaultSocketMode,
		DirMode: defaultSocketDirMode,
	}
}

func unixSocketOptionsFromFlags() (unixSocketOptions, error) {
	socketMode, err := parseSocketMode(opts.SocketMode)
	if err != nil {
		return unixSocketOptions{}, err
	}
	dirMode, err := parseSocketDirMode(opts.SocketDirMode)
	if err != nil {
		return unixSocketOptions{}, err
	}
	return unixSocketOptions{
		Mode:    socketMode,
		DirMode: dirMode,
		Owner:   opts.SocketOwner,
		Group:   opts.SocketGroup,
	}, nil
}

func parseSocketMode(value string) (fs.FileMode, error) {
	mode, err := parseOctalMode(value, "socket mode")
	if err != nil {
		return 0, err
	}
	if mode&0600 != 0600 {
		return 0, fmt.Errorf("socket mode %04o must allow owner read/write", mode)
	}
	if mode&0007 != 0 {
		return 0, fmt.Errorf("socket mode %04o must not grant world access", mode)
	}
	if mode&0111 != 0 {
		return 0, fmt.Errorf("socket mode %04o must not include execute bits", mode)
	}
	return mode, nil
}

func parseSocketDirMode(value string) (fs.FileMode, error) {
	mode, err := parseOctalMode(value, "socket directory mode")
	if err != nil {
		return 0, err
	}
	if mode&0700 != 0700 {
		return 0, fmt.Errorf("socket directory mode %04o must allow owner read/write/execute", mode)
	}
	if mode&0007 != 0 {
		return 0, fmt.Errorf("socket directory mode %04o must not grant world access", mode)
	}
	return mode, nil
}

func parseOctalMode(value, name string) (fs.FileMode, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "0o")
	value = strings.TrimPrefix(value, "0O")
	if value == "" {
		return 0, fmt.Errorf("%s must not be empty", name)
	}
	parsed, err := strconv.ParseUint(value, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %s %q as octal: %w", name, value, err)
	}
	if parsed > 0777 {
		return 0, fmt.Errorf("%s %04o exceeds permission bits", name, parsed)
	}
	return fs.FileMode(parsed).Perm(), nil
}

func listenUnixSocket(path string) (net.Listener, error) {
	return listenUnixSocketWithOptions(path, defaultUnixSocketOptions())
}

func listenUnixSocketWithOptions(path string, socketOpts unixSocketOptions) (net.Listener, error) {
	if err := ensureSocketDir(filepath.Dir(path), socketOpts.DirMode); err != nil {
		return nil, fmt.Errorf("create auth socket directory: %w", err)
	}
	if err := removeStaleUnixSocket(path); err != nil {
		return nil, err
	}

	mask := int(0777 &^ socketOpts.Mode)
	listener, err := listenWithUmask(mask, func() (net.Listener, error) {
		return net.Listen(protocol, path)
	})
	if err != nil {
		return nil, err
	}
	if err := applyUnixSocketOwnership(path, socketOpts.Owner, socketOpts.Group); err != nil {
		listener.Close()
		return nil, err
	}
	if err := ensureUnixSocketPath(path, "restrict auth socket permissions"); err != nil {
		listener.Close()
		return nil, err
	}
	if err := os.Chmod(path, socketOpts.Mode); err != nil {
		listener.Close()
		return nil, fmt.Errorf("restrict auth socket permissions: %w", err)
	}
	if err := ensureUnixSocketPath(path, "verify auth socket permissions target"); err != nil {
		listener.Close()
		return nil, err
	}
	return listener, nil
}

func ensureSocketDir(dir string, mode fs.FileMode) error {
	if dir == "" {
		dir = "."
	}
	_, statErr := os.Stat(dir)
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	if err := os.MkdirAll(dir, mode); err != nil {
		return err
	}
	if os.IsNotExist(statErr) {
		return os.Chmod(dir, mode)
	}
	return nil
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
	socketOpts, err := unixSocketOptionsFromFlags()
	if err != nil {
		log.Fatal(err)
	}
	listener, err := listenUnixSocketWithOptions(opts.SocketAddr, socketOpts)
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
