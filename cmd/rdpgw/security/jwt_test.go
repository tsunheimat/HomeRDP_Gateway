package security

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/protocol"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestGenerateUserToken(t *testing.T) {
	cases := []struct {
		SigningKey    []byte
		EncryptionKey []byte
		name          string
		username      string
	}{
		{
			SigningKey:    []byte("5aa3a1568fe8421cd7e127d5ace28d2d"),
			EncryptionKey: []byte("d3ecd7e565e56e37e2f2e95b584d8c0c"),
			name:          "sign_and_encrypt",
			username:      "test_sign_and_encrypt",
		},
		{
			SigningKey:    nil,
			EncryptionKey: []byte("d3ecd7e565e56e37e2f2e95b584d8c0c"),
			name:          "encrypt_only",
			username:      "test_encrypt_only",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			SigningKey = tc.SigningKey
			UserEncryptionKey = tc.EncryptionKey
			token, err := GenerateUserToken(context.Background(), tc.username)
			if err != nil {
				t.Fatalf("GenerateUserToken failed: %s", err)
			}
			claims, err := UserInfo(context.Background(), token)
			if err != nil {
				t.Fatalf("UserInfo failed: %s", err)
			}
			if claims.Subject != tc.username {
				t.Fatalf("Expected %s, got %s", tc.username, claims.Subject)
			}
		})
	}

}

func TestPAACookie(t *testing.T) {
	SigningKey = []byte("5aa3a1568fe8421cd7e127d5ace28d2d")
	EncryptionKey = []byte("d3ecd7e565e56e37e2f2e95b584d8c0c")

	username := "test_paa_cookie"
	attr_client_ip := "127.0.0.1"
	attr_access_token := "aabbcc"

	id := identity.NewUser()
	id.SetUserName(username)
	id.SetAttribute(identity.AttrClientIp, attr_client_ip)
	id.SetAttribute(identity.AttrAccessToken, attr_access_token)

	ctx := context.Background()
	ctx = context.WithValue(ctx, identity.CTXKey, id)

	token, err := GeneratePAAToken(ctx, "test_paa_cookie", "host.does.not.exist")
	if err != nil {
		t.Fatalf("GeneratePAAToken failed: %s", err)
	}
	if strings.Contains(token, attr_access_token) {
		t.Fatalf("serialized PAA token contains access token in plaintext")
	}
	for _, segment := range strings.Split(token, ".") {
		decoded, err := base64.RawURLEncoding.DecodeString(segment)
		if err != nil {
			continue
		}
		if strings.Contains(string(decoded), attr_access_token) {
			t.Fatalf("serialized PAA token has readable segment containing access token: %q", string(decoded))
		}
	}

	enc, err := jwt.ParseSignedAndEncrypted(
		token,
		[]jose.KeyAlgorithm{jose.DIRECT},
		[]jose.ContentEncryption{jose.A128CBC_HS256},
		[]jose.SignatureAlgorithm{jose.HS256},
	)
	if err != nil {
		t.Fatalf("ParseSignedAndEncrypted failed: %s", err)
	}
	signedToken, err := enc.Decrypt(EncryptionKey)
	if err != nil {
		t.Fatalf("decrypting encrypted PAA token failed: %s", err)
	}
	standard := jwt.Claims{}
	custom := customClaims{}
	if err := signedToken.Claims(SigningKey, &standard, &custom); err != nil {
		t.Fatalf("validating encrypted PAA token signature failed: %s", err)
	}
	if standard.Subject != username {
		t.Fatalf("expected subject %q, got %q", username, standard.Subject)
	}
	if custom.AccessToken != attr_access_token {
		t.Fatalf("expected encrypted access token claim to round-trip")
	}
	/*ok, err := CheckPAACookie(ctx, token)
	if err != nil {
		t.Fatalf("CheckPAACookie failed: %s", err)
	}
	if !ok {
		t.Fatalf("CheckPAACookie failed")
	}*/
}

func TestCheckSessionWithoutLegacyHostCheck(t *testing.T) {
	previousVerifyClientIP := VerifyClientIP
	VerifyClientIP = true
	defer func() {
		VerifyClientIP = previousVerifyClientIP
	}()

	id := identity.NewUser()
	id.SetAttribute(identity.AttrClientIp, "10.0.0.10")

	ctx := context.WithValue(context.Background(), protocol.CtxTunnel, &protocol.Tunnel{
		TargetServer: "lab-win11.internal:3389",
		RemoteAddr:   "10.0.0.10",
	})
	ctx = context.WithValue(ctx, identity.CTXKey, id)

	checkHost := CheckSession(nil)
	ok, err := checkHost(ctx, "lab-win11.internal:3389")
	if err != nil {
		t.Fatalf("CheckSession returned error: %v", err)
	}
	if !ok {
		t.Fatalf("CheckSession returned false, want true")
	}
}
