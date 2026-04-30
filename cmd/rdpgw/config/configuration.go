package config

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strings"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/security"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	TlsDisable = "disable"
	TlsAuto    = "auto"

	HostSelectionSigned     = "signed"
	HostSelectionRoundRobin = "roundrobin"

	SessionStoreCookie = "cookie"
	SessionStoreFile   = "file"

	AuthenticationOpenId   = "openid"
	AuthenticationBasic    = "local"
	AuthenticationKerberos = "kerberos"
	AuthenticationHeader   = "header"

	queryTokenSigningKeyMinBytes = 32

	uploadStorageBytesPerMegabyte = 1024 * 1024
	maxUploadStorageMegabytes     = math.MaxInt64 / uploadStorageBytesPerMegabyte
)

type Configuration struct {
	Server    ServerConfig    `koanf:"server"`
	OpenId    OpenIDConfig    `koanf:"openid"`
	Dashboard DashboardConfig `koanf:"dashboard"`
	Kerberos  KerberosConfig  `koanf:"kerberos"`
	Header    HeaderConfig    `koanf:"header"`
	Caps      RDGCapsConfig   `koanf:"caps"`
	Security  SecurityConfig  `koanf:"security"`
	Client    ClientConfig    `koanf:"client"`
}

type ServerConfig struct {
	GatewayAddress       string   `koanf:"gatewayaddress"`
	Port                 int      `koanf:"port"`
	CertFile             string   `koanf:"certfile"`
	KeyFile              string   `koanf:"keyfile"`
	Hosts                []string `koanf:"hosts"`
	HostSelection        string   `koanf:"hostselection"`
	SessionKey           string   `koanf:"sessionkey"`
	SessionEncryptionKey string   `koanf:"sessionencryptionkey"`
	SessionStore         string   `koanf:"sessionstore"`
	MaxSessionLength     int      `koanf:"maxsessionlength"`
	SendBuf              int      `koanf:"sendbuf"`
	ReceiveBuf           int      `koanf:"receivebuf"`
	Tls                  string   `koanf:"tls"`
	Authentication       []string `koanf:"authentication"`
	AuthSocket           string   `koanf:"authsocket"`
	BasicAuthTimeout     int      `koanf:"basicauthtimeout"`
	TrustedProxyCIDRs    []string `koanf:"trustedproxycidrs"`
	InternalDomains      []string `koanf:"internaldomains"`
	InternalDNSServer    string   `koanf:"internaldnsserver"`
	SecureCookies        bool     `koanf:"securecookies"`
	AllowTLSKeyLog       bool     `koanf:"allowtlskeylog"`
	EnableMetrics        bool     `koanf:"enablemetrics"`
}

type KerberosConfig struct {
	Keytab   string `koanf:"keytab"`
	Krb5Conf string `koanf:"krb5conf"`
}

type OpenIDConfig struct {
	ProviderUrl  string `koanf:"providerurl"`
	ClientId     string `koanf:"clientid"`
	ClientSecret string `koanf:"clientsecret"`
	GroupsClaim  string `koanf:"groupsclaim"`
}

type DashboardConfig struct {
	StorePath                  string   `koanf:"storepath"`
	UploadDir                  string   `koanf:"uploaddir"`
	IconDir                    string   `koanf:"icondir"`
	AuthUsersPath              string   `koanf:"authuserspath"`
	AuthHelperConfigPath       string   `koanf:"authhelperconfigpath"`
	AdminGroups                []string `koanf:"admingroups"`
	MaxUploadSizeMb            int      `koanf:"maxuploadsizemb"`
	MaxTemplateUploads         int      `koanf:"maxtemplateuploads"`
	MaxIconUploads             int      `koanf:"maxiconuploads"`
	MaxTemplateUploadStorageMb int      `koanf:"maxtemplateuploadstoragemb"`
	MaxIconUploadStorageMb     int      `koanf:"maxiconuploadstoragemb"`
}

type HeaderConfig struct {
	UserHeader        string `koanf:"userheader"`
	UserIdHeader      string `koanf:"useridheader"`
	EmailHeader       string `koanf:"emailheader"`
	DisplayNameHeader string `koanf:"displaynameheader"`
}

type RDGCapsConfig struct {
	SmartCardAuth   bool `koanf:"smartcardauth"`
	TokenAuth       bool `koanf:"tokenauth"`
	IdleTimeout     int  `koanf:"idletimeout"`
	RedirectAll     bool `koanf:"redirectall"`
	DisableRedirect bool `koanf:"disableredirect"`
	EnableClipboard bool `koanf:"enableclipboard"`
	EnablePrinter   bool `koanf:"enableprinter"`
	EnablePort      bool `koanf:"enableport"`
	EnablePnp       bool `koanf:"enablepnp"`
	EnableDrive     bool `koanf:"enabledrive"`
}

type SecurityConfig struct {
	PAATokenEncryptionKey  string `koanf:"paatokenencryptionkey"`
	PAATokenSigningKey     string `koanf:"paatokensigningkey"`
	UserTokenEncryptionKey string `koanf:"usertokenencryptionkey"`
	UserTokenSigningKey    string `koanf:"usertokensigningkey"`
	QueryTokenSigningKey   string `koanf:"querytokensigningkey"`
	QueryTokenIssuer       string `koanf:"querytokenissuer"`
	VerifyClientIp         bool   `koanf:"verifyclientip"`
	EnableUserToken        bool   `koanf:"enableusertoken"`
}

type ClientConfig struct {
	Defaults string `koanf:"defaults"`
	// kept for backwards compatibility
	UsernameTemplate string `koanf:"usernametemplate"`
	SplitUserDomain  bool   `koanf:"splituserdomain"`
	NoUsername       bool   `koanf:"nousername"`
	SigningCert      string `koanf:"signingcert"`
	SigningKey       string `koanf:"signingkey"`
}

func ToCamel(s string) string {
	s = strings.TrimSpace(s)
	n := strings.Builder{}
	n.Grow(len(s))
	var capNext bool = true
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if capNext {
			if vIsLow {
				v += 'A'
				v -= 'a'
			}
		} else if i == 0 {
			if vIsCap {
				v += 'a'
				v -= 'A'
			}
		}
		if vIsCap || vIsLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
			if v == '.' {
				n.WriteByte(v)
			}
		}
	}
	return n.String()
}

var envKeyOverrides = map[string]string{
	"Openid.Groupsclaim":                   "OpenId.GroupsClaim",
	"Dashboard.Storepath":                  "Dashboard.StorePath",
	"Dashboard.Uploaddir":                  "Dashboard.UploadDir",
	"Dashboard.Icondir":                    "Dashboard.IconDir",
	"Dashboard.Authuserspath":              "Dashboard.AuthUsersPath",
	"Dashboard.Authhelperconfigpath":       "Dashboard.AuthHelperConfigPath",
	"Dashboard.Admingroups":                "Dashboard.AdminGroups",
	"Dashboard.Maxuploadsizemb":            "Dashboard.MaxUploadSizeMb",
	"Dashboard.Maxtemplateuploads":         "Dashboard.MaxTemplateUploads",
	"Dashboard.Maxiconuploads":             "Dashboard.MaxIconUploads",
	"Dashboard.Maxtemplateuploadstoragemb": "Dashboard.MaxTemplateUploadStorageMb",
	"Dashboard.Maxiconuploadstoragemb":     "Dashboard.MaxIconUploadStorageMb",
	"Server.Trustedproxycidrs":             "Server.TrustedProxyCIDRs",
	"Server.Internaldomains":               "Server.InternalDomains",
	"Server.Internaldnsserver":             "Server.InternalDNSServer",
	"Server.Securecookies":                 "Server.SecureCookies",
	"Server.Allowtlskeylog":                "Server.AllowTLSKeyLog",
	"Server.Enablemetrics":                 "Server.EnableMetrics",
}

var Conf Configuration

func deriveUploadDir(storePath string) string {
	if storePath == "" {
		return "uploads"
	}
	if strings.HasSuffix(storePath, "/") {
		return storePath + "uploads"
	}
	return storePath + "/uploads"
}

func deriveIconDir(storePath string) string {
	if storePath == "" {
		return "icons"
	}
	if strings.HasSuffix(storePath, "/") {
		return storePath + "icons"
	}
	return storePath + "/icons"
}

func deriveAuthUsersPath(storePath string) string {
	if storePath == "" {
		return "auth-users.json"
	}
	if strings.HasSuffix(storePath, "/") {
		return storePath + "auth-users.json"
	}
	return storePath + "/auth-users.json"
}

func deriveAuthHelperConfigPath(storePath string) string {
	if storePath == "" {
		return "rdpgw-auth.yaml"
	}
	if strings.HasSuffix(storePath, "/") {
		return storePath + "rdpgw-auth.yaml"
	}
	return storePath + "/rdpgw-auth.yaml"
}

func Load(configFile string) Configuration {
	Conf.Dashboard = DashboardConfig{}

	var k = koanf.New(".")

	k.Load(confmap.Provider(map[string]interface{}{
		"Server.Tls":                           "auto",
		"Server.Port":                          443,
		"Server.SessionStore":                  "cookie",
		"Server.HostSelection":                 "roundrobin",
		"Server.Authentication":                "openid",
		"Server.AuthSocket":                    "/run/rdpgw/rdpgw-auth.sock",
		"Server.BasicAuthTimeout":              5,
		"Server.AllowTLSKeyLog":                false,
		"OpenId.GroupsClaim":                   "groups",
		"Dashboard.StorePath":                  "./data/dashboard",
		"Dashboard.MaxUploadSizeMb":            5,
		"Dashboard.MaxTemplateUploads":         100,
		"Dashboard.MaxIconUploads":             100,
		"Dashboard.MaxTemplateUploadStorageMb": 100,
		"Dashboard.MaxIconUploadStorageMb":     100,
		"Client.NetworkAutoDetect":             1,
		"Client.BandwidthAutoDetect":           1,
		"Security.VerifyClientIp":              true,
		"Caps.TokenAuth":                       true,
	}, "."), nil)

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Printf("Config file %s not found, using defaults and environment", configFile)
	} else {
		if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
			log.Fatalf("Error loading config from file: %v", err)
		}
	}

	if err := k.Load(env.ProviderWithValue("RDPGW_", ".", func(s string, v string) (string, interface{}) {
		key := strings.Replace(strings.ToLower(strings.TrimPrefix(s, "RDPGW_")), "__", ".", -1)
		key = ToCamel(key)
		if override, ok := envKeyOverrides[key]; ok {
			key = override
		}

		v = strings.Trim(v, " ")

		// handle lists
		if strings.Contains(v, " ") {
			return key, strings.Split(v, " ")
		}
		return key, v

	}), nil); err != nil {
		log.Fatalf("Error loading config from environment: %v", err)
	}

	koanfTag := koanf.UnmarshalConf{Tag: "koanf"}
	k.UnmarshalWithConf("Server", &Conf.Server, koanfTag)
	k.UnmarshalWithConf("OpenId", &Conf.OpenId, koanfTag)
	k.UnmarshalWithConf("Dashboard", &Conf.Dashboard, koanfTag)
	k.UnmarshalWithConf("Header", &Conf.Header, koanfTag)
	k.UnmarshalWithConf("Caps", &Conf.Caps, koanfTag)
	k.UnmarshalWithConf("Security", &Conf.Security, koanfTag)
	k.UnmarshalWithConf("Client", &Conf.Client, koanfTag)
	k.UnmarshalWithConf("Kerberos", &Conf.Kerberos, koanfTag)

	if Conf.Dashboard.UploadDir == "" {
		Conf.Dashboard.UploadDir = deriveUploadDir(Conf.Dashboard.StorePath)
	}
	if Conf.Dashboard.IconDir == "" {
		Conf.Dashboard.IconDir = deriveIconDir(Conf.Dashboard.StorePath)
	}
	if Conf.Dashboard.AuthUsersPath == "" {
		Conf.Dashboard.AuthUsersPath = deriveAuthUsersPath(Conf.Dashboard.StorePath)
	}
	if Conf.Dashboard.AuthHelperConfigPath == "" {
		Conf.Dashboard.AuthHelperConfigPath = deriveAuthHelperConfigPath(Conf.Dashboard.StorePath)
	}

	if !Conf.Server.OpenIDEnabled() {
		Conf.Caps.TokenAuth = false
		Conf.Security.EnableUserToken = false
	}

	if len(Conf.Security.PAATokenEncryptionKey) != 32 {
		Conf.Security.PAATokenEncryptionKey, _ = security.GenerateRandomString(32)
		log.Printf("No valid `security.paatokenencryptionkey` specified (empty or not 32 characters). Setting to random")
	}

	if len(Conf.Security.PAATokenSigningKey) != 32 {
		Conf.Security.PAATokenSigningKey, _ = security.GenerateRandomString(32)
		log.Printf("No valid `security.paatokensigningkey` specified (empty or not 32 characters). Setting to random")
	}

	if Conf.Security.EnableUserToken {
		if len(Conf.Security.UserTokenEncryptionKey) != 32 {
			Conf.Security.UserTokenEncryptionKey, _ = security.GenerateRandomString(32)
			log.Printf("No valid `security.usertokenencryptionkey` specified (empty or not 32 characters). Setting to random")
		}
	}

	if len(Conf.Server.SessionKey) != 32 {
		Conf.Server.SessionKey, _ = security.GenerateRandomString(32)
		log.Printf("No valid `server.sessionkey` specified (empty or not 32 characters). Setting to random")
	}

	if len(Conf.Server.SessionEncryptionKey) != 32 {
		Conf.Server.SessionEncryptionKey, _ = security.GenerateRandomString(32)
		log.Printf("No valid `server.sessionencryptionkey` specified (empty or not 32 characters). Setting to random")
	}

	if Conf.Server.BasicAuthEnabled() && Conf.Server.Tls == "disable" {
		log.Fatalf("basicauth=local and tls=disable are mutually exclusive")
	}

	if Conf.Server.NtlmEnabled() && Conf.Server.KerberosEnabled() {
		log.Fatalf("ntlm and kerberos authentication are not stackable")
	}

	if !Conf.Caps.TokenAuth && Conf.Server.OpenIDEnabled() {
		log.Fatalf("openid is configured but tokenauth disabled")
	}

	if Conf.Server.KerberosEnabled() && Conf.Kerberos.Keytab == "" {
		log.Fatalf("kerberos is configured but no keytab was specified")
	}

	if Conf.Server.HeaderEnabled() && Conf.Header.UserHeader == "" {
		log.Fatalf("header authentication is configured but no user header was specified")
	}

	if err := Conf.Validate(); err != nil {
		log.Fatalf("%s", err)
	}

	// prepend '//' if required for URL parsing
	if !strings.Contains(Conf.Server.GatewayAddress, "//") {
		Conf.Server.GatewayAddress = "//" + Conf.Server.GatewayAddress
	}

	return Conf
}

func (c Configuration) Validate() error {
	for _, cidr := range c.Server.TrustedProxyCIDRs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid server.trustedproxycidrs entry %q: %w", cidr, err)
		}
	}

	if c.Server.HeaderEnabled() {
		if c.Header.UserHeader == "" {
			return fmt.Errorf("header authentication is configured but no user header was specified")
		}
		if !hasTrustedProxyCIDR(c.Server.TrustedProxyCIDRs) {
			return fmt.Errorf("header authentication requires server.trustedproxycidrs to trust the authenticating reverse proxy")
		}
	}

	if c.Server.HostSelection == HostSelectionSigned && len(c.Security.QueryTokenSigningKey) < queryTokenSigningKeyMinBytes {
		return fmt.Errorf("host selection is set to %q but security.querytokensigningkey is shorter than %d bytes", HostSelectionSigned, queryTokenSigningKeyMinBytes)
	}

	if c.Dashboard.MaxUploadSizeMb < 0 {
		return fmt.Errorf("dashboard.maxuploadsizemb must be greater than or equal to zero")
	}
	if c.Dashboard.MaxUploadSizeMb > maxUploadStorageMegabytes {
		return fmt.Errorf("dashboard.maxuploadsizemb must be less than or equal to %d", maxUploadStorageMegabytes)
	}
	if c.Dashboard.MaxTemplateUploads < 0 {
		return fmt.Errorf("dashboard.maxtemplateuploads must be greater than or equal to zero")
	}
	if c.Dashboard.MaxIconUploads < 0 {
		return fmt.Errorf("dashboard.maxiconuploads must be greater than or equal to zero")
	}
	if c.Dashboard.MaxTemplateUploadStorageMb < 0 {
		return fmt.Errorf("dashboard.maxtemplateuploadstoragemb must be greater than or equal to zero")
	}
	if c.Dashboard.MaxTemplateUploadStorageMb > maxUploadStorageMegabytes {
		return fmt.Errorf("dashboard.maxtemplateuploadstoragemb must be less than or equal to %d", maxUploadStorageMegabytes)
	}
	if c.Dashboard.MaxIconUploadStorageMb < 0 {
		return fmt.Errorf("dashboard.maxiconuploadstoragemb must be greater than or equal to zero")
	}
	if c.Dashboard.MaxIconUploadStorageMb > maxUploadStorageMegabytes {
		return fmt.Errorf("dashboard.maxiconuploadstoragemb must be less than or equal to %d", maxUploadStorageMegabytes)
	}

	return nil
}

func hasTrustedProxyCIDR(cidrs []string) bool {
	for _, cidr := range cidrs {
		if strings.TrimSpace(cidr) != "" {
			return true
		}
	}
	return false
}

func (s *ServerConfig) OpenIDEnabled() bool {
	return s.matchAuth("openid")
}

func (s *ServerConfig) KerberosEnabled() bool {
	return s.matchAuth("kerberos")
}

func (s *ServerConfig) BasicAuthEnabled() bool {
	return s.matchAuth("local") || s.matchAuth("basic")
}

func (s *ServerConfig) NtlmEnabled() bool {
	return s.matchAuth("ntlm")
}

func (s *ServerConfig) HeaderEnabled() bool {
	return s.matchAuth("header")
}

func (s *ServerConfig) matchAuth(needle string) bool {
	for _, q := range s.Authentication {
		if q == needle {
			return true
		}
	}
	return false
}
