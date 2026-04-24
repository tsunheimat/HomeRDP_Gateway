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

func TestParseSocketMode(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    uint32
		wantErr bool
	}{
		{name: "owner only", value: "0600", want: 0600},
		{name: "group access", value: "0660", want: 0660},
		{name: "0o prefix", value: "0o660", want: 0660},
		{name: "world readable rejected", value: "0644", wantErr: true},
		{name: "world writable rejected", value: "0666", wantErr: true},
		{name: "owner write missing rejected", value: "0400", wantErr: true},
		{name: "execute bits rejected", value: "0700", wantErr: true},
		{name: "invalid octal rejected", value: "0999", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSocketMode(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected parseSocketMode(%q) to fail", tt.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSocketMode(%q) returned error: %v", tt.value, err)
			}
			if uint32(got) != tt.want {
				t.Fatalf("expected %04o, got %04o", tt.want, got)
			}
		})
	}
}

func TestParseSocketDirMode(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    uint32
		wantErr bool
	}{
		{name: "owner only", value: "0700", want: 0700},
		{name: "group traverse", value: "0750", want: 0750},
		{name: "world access rejected", value: "0755", wantErr: true},
		{name: "owner execute missing rejected", value: "0600", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSocketDirMode(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected parseSocketDirMode(%q) to fail", tt.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSocketDirMode(%q) returned error: %v", tt.value, err)
			}
			if uint32(got) != tt.want {
				t.Fatalf("expected %04o, got %04o", tt.want, got)
			}
		})
	}
}
