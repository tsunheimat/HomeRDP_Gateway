package main

import (
	"context"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/auth/config"
	"github.com/bolkedebruin/rdpgw/cmd/auth/database"
	"github.com/bolkedebruin/rdpgw/shared/auth"
)

func TestAuthenticateUsesConfiguredUsers(t *testing.T) {
	service := NewAuthService(database.NewConfig([]config.UserConfig{
		{Username: "alice", Password: "secret"},
	}))

	res, err := service.Authenticate(context.Background(), &auth.UserPass{
		Username: "alice",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if !res.Authenticated {
		t.Fatalf("expected configured user to authenticate, got %+v", res)
	}
}

func TestAuthenticateRejectsWrongPassword(t *testing.T) {
	service := NewAuthService(database.NewConfig([]config.UserConfig{
		{Username: "alice", Password: "secret"},
	}))

	res, err := service.Authenticate(context.Background(), &auth.UserPass{
		Username: "alice",
		Password: "wrong",
	})
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if res.Authenticated {
		t.Fatalf("expected wrong password to be rejected, got %+v", res)
	}
}
