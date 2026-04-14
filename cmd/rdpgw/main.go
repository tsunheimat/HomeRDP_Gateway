package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bolkedebruin/gokrb5/v8/keytab"
	"github.com/bolkedebruin/gokrb5/v8/service"
	"github.com/bolkedebruin/gokrb5/v8/spnego"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/config"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/kdcproxy"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/protocol"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/security"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/web"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thought-machine/go-flags"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/oauth2"
)

const (
	gatewayEndPoint  = "/remoteDesktopGateway/"
	kdcProxyEndPoint = "/KdcProxy"
)

var opts struct {
	ConfigFile string `short:"c" long:"conf" default:"rdpgw.yaml" description:"config file (yaml)"`
}

var conf config.Configuration

func authHelperConfigPath(conf config.Configuration) string {
	if runtimePath := strings.TrimSpace(os.Getenv("RDPGW_AUTH_HELPER_CONFIG")); runtimePath != "" {
		return runtimePath
	}
	return conf.Dashboard.AuthHelperConfigPath
}

func validateManagedDirectAuthConfig(conf config.Configuration) error {
	if !conf.Server.BasicAuthEnabled() && !conf.Server.NtlmEnabled() {
		return nil
	}
	if !conf.Server.OpenIDEnabled() {
		return errors.New("local and ntlm direct auth require openid-managed dashboard state")
	}
	return nil
}

func initOIDC(callbackUrl *url.URL) *web.OIDC {
	// set oidc config
	provider, err := oidc.NewProvider(context.Background(), conf.OpenId.ProviderUrl)
	if err != nil {
		log.Fatalf("Cannot get oidc provider: %s", err)
	}
	oidcConfig := &oidc.Config{
		ClientID: conf.OpenId.ClientId,
	}
	verifier := provider.Verifier(oidcConfig)

	oauthConfig := oauth2.Config{
		ClientID:     conf.OpenId.ClientId,
		ClientSecret: conf.OpenId.ClientSecret,
		RedirectURL:  callbackUrl.String(),
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	security.OIDCProvider = provider
	security.Oauth2Config = oauthConfig

	o := web.OIDCConfig{
		OAuth2Config:      &oauthConfig,
		OIDCTokenVerifier: verifier,
		GroupsClaim:       conf.OpenId.GroupsClaim,
	}

	return o.New()
}

func main() {
	// load config
	_, err := flags.Parse(&opts)
	if err != nil {
		panic(err)
	}
	conf = config.Load(opts.ConfigFile)
	if err := validateManagedDirectAuthConfig(conf); err != nil {
		log.Fatal(err)
	}
	helperConfigPath := authHelperConfigPath(conf)

	// set callback url and external advertised gateway address
	url, err := url.Parse(conf.Server.GatewayAddress)
	if err != nil {
		log.Printf("Cannot parse server gateway address %s due to %s", url, err)
	}
	if url.Scheme == "" {
		url.Scheme = "https"
	}
	url.Path = "callback"

	// set security options
	security.VerifyClientIP = conf.Security.VerifyClientIp
	security.SigningKey = []byte(conf.Security.PAATokenSigningKey)
	security.EncryptionKey = []byte(conf.Security.PAATokenEncryptionKey)
	security.UserEncryptionKey = []byte(conf.Security.UserTokenEncryptionKey)
	security.UserSigningKey = []byte(conf.Security.UserTokenSigningKey)
	security.QuerySigningKey = []byte(conf.Security.QueryTokenSigningKey)
	security.HostSelection = conf.Server.HostSelection
	security.Hosts = nil
	security.ManagedHostList = nil

	// init session store
	web.InitStore([]byte(conf.Server.SessionKey),
		[]byte(conf.Server.SessionEncryptionKey),
		conf.Server.SessionStore,
		conf.Server.MaxSessionLength,
	)

	var dashboardStore dashboard.Store
	var authUserStore dashboard.AuthUserStore
	if conf.Server.OpenIDEnabled() {
		dashboardStore, err = dashboard.NewFileStore(conf.Dashboard.StorePath, conf.Dashboard.UploadDir)
		if err != nil {
			log.Fatalf("Cannot initialize dashboard store: %s", err)
		}
		authUserStore, err = dashboard.NewFileAuthUserStore(conf.Dashboard.AuthUsersPath)
		if err != nil {
			log.Fatalf("Cannot initialize auth user store: %s", err)
		}
		authUsers, err := authUserStore.List()
		if err != nil {
			log.Fatalf("Cannot list auth users: %s", err)
		}
		if err := dashboard.WriteAuthHelperConfig(helperConfigPath, authUsers); err != nil {
			log.Fatalf("Cannot write auth helper config: %s", err)
		}
		security.ManagedHostList = func() ([]string, error) {
			entries, err := dashboardStore.List()
			if err != nil {
				return nil, err
			}
			return dashboard.EnabledHostAddresses(entries), nil
		}
	}

	// configure web backend
	w := &web.Config{
		QueryInfo:        security.QueryInfo,
		QueryTokenIssuer: conf.Security.QueryTokenIssuer,
		EnableUserToken:  conf.Security.EnableUserToken,
		AdminGroups:      conf.Dashboard.AdminGroups,
		Hosts:            conf.Server.Hosts,
		HostSelection:    conf.Server.HostSelection,
		RdpOpts: web.RdpOpts{
			UsernameTemplate: conf.Client.UsernameTemplate,
			SplitUserDomain:  conf.Client.SplitUserDomain,
			NoUsername:       conf.Client.NoUsername,
		},
		GatewayAddress:         url,
		TemplateFile:           conf.Client.Defaults,
		RdpSigningCert:         conf.Client.SigningCert,
		RdpSigningKey:          conf.Client.SigningKey,
		DashboardStore:         dashboardStore,
		DashboardAuthUserStore: authUserStore,
		AuthHelperConfigPath:   helperConfigPath,
	}

	if conf.Caps.TokenAuth {
		w.PAATokenGenerator = security.GeneratePAAToken
	}
	if conf.Security.EnableUserToken {
		w.UserTokenGenerator = security.GenerateUserToken
	}
	if conf.Server.OpenIDEnabled() {
		w.DashboardMaxUploadBytes = int64(conf.Dashboard.MaxUploadSizeMb) * 1024 * 1024
	}
	h := w.NewHandler()

	log.Printf("Starting remote desktop gateway server")
	cfg := &tls.Config{}

	// configure tls security
	if conf.Server.Tls == config.TlsDisable {
		log.Printf("TLS disabled - rdp gw connections require tls, make sure to have a terminator")
	} else {
		// auto config
		tlsConfigured := false

		tlsDebug := os.Getenv("SSLKEYLOGFILE")
		if tlsDebug != "" {
			w, err := os.OpenFile(tlsDebug, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				log.Fatalf("Cannot open key log file %s for writing %s", tlsDebug, err)
			}
			log.Printf("Key log file set to: %s", tlsDebug)
			cfg.KeyLogWriter = w
		}

		if conf.Server.KeyFile != "" && conf.Server.CertFile != "" {
			cert, err := tls.LoadX509KeyPair(conf.Server.CertFile, conf.Server.KeyFile)
			if err != nil {
				log.Printf("Cannot load certfile or keyfile (%s) falling back to acme", err)
			}
			cfg.Certificates = append(cfg.Certificates, cert)
			tlsConfigured = true
		}

		if !tlsConfigured {
			log.Printf("Using acme / letsencrypt for tls configuration. Enabling http (port 80) for verification")
			// setup a simple handler which sends a HTHS header for six months (!)
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Strict-Transport-Security", "max-age=15768000 ; includeSubDomains")
				fmt.Fprintf(w, "Hello from RDPGW")
			})

			certMgr := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(url.Host),
				Cache:      autocert.DirCache("/tmp/rdpgw"),
			}
			cfg.GetCertificate = certMgr.GetCertificate

			go func() {
				http.ListenAndServe(":80", certMgr.HTTPHandler(nil))
			}()
		}
	}

	// gateway confg
	gw := protocol.Gateway{
		RedirectFlags: protocol.RedirectFlags{
			Clipboard:  conf.Caps.EnableClipboard,
			Drive:      conf.Caps.EnableDrive,
			Printer:    conf.Caps.EnablePrinter,
			Port:       conf.Caps.EnablePort,
			Pnp:        conf.Caps.EnablePnp,
			DisableAll: conf.Caps.DisableRedirect,
			EnableAll:  conf.Caps.RedirectAll,
		},
		IdleTimeout:   conf.Caps.IdleTimeout,
		SmartCardAuth: conf.Caps.SmartCardAuth,
		TokenAuth:     conf.Caps.TokenAuth,
		ReceiveBuf:    conf.Server.ReceiveBuf,
		SendBuf:       conf.Server.SendBuf,
	}

	if conf.Caps.TokenAuth {
		gw.CheckPAACookie = security.CheckPAACookie
		gw.CheckHost = security.CheckSession(nil)
	} else {
		gw.CheckHost = security.CheckHost
	}

	r := mux.NewRouter()

	// ensure identity is set in context and get some extra info
	r.Use(web.EnrichContext)

	// prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	// for sso callbacks
	r.HandleFunc("/tokeninfo", web.TokenInfo)

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// gateway endpoint
	rdp := r.PathPrefix(gatewayEndPoint).Subrouter()

	// openid
	if conf.Server.OpenIDEnabled() {
		log.Printf("enabling openid extended authentication")
		o := initOIDC(url)
		r.Handle("/connect/entries/{id}.rdp", o.Authenticated(http.HandlerFunc(h.HandleEntryDownload)))
		r.HandleFunc("/callback", o.HandleCallback)

		// Web interface and API routes (authenticated)
		r.Handle("/", o.Authenticated(http.HandlerFunc(h.HandleDashboard)))
		r.Handle("/admin", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminPage))))
		api.Handle("/user", o.Authenticated(http.HandlerFunc(h.HandleDashboardUserInfo)))
		api.Handle("/entries", o.Authenticated(http.HandlerFunc(h.HandleEntryList)))
		api.Handle("/admin/entries", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminListEntries)))).Methods(http.MethodGet)
		api.Handle("/admin/entries/host", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminCreateHostEntry)))).Methods(http.MethodPost)
		api.Handle("/admin/entries/template", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminCreateTemplateEntry)))).Methods(http.MethodPost)
		api.Handle("/admin/entries/{id}", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminUpdateEntry)))).Methods(http.MethodPut)
		api.Handle("/admin/entries/{id}", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminDeleteEntry)))).Methods(http.MethodDelete)
		api.Handle("/admin/auth-users", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminListAuthUsers)))).Methods(http.MethodGet)
		api.Handle("/admin/auth-users", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminCreateAuthUser)))).Methods(http.MethodPost)
		api.Handle("/admin/auth-users/{username}", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminUpdateAuthUser)))).Methods(http.MethodPut)
		api.Handle("/admin/auth-users/{username}", o.Authenticated(h.AdminOnly(http.HandlerFunc(h.HandleAdminDeleteAuthUser)))).Methods(http.MethodDelete)

		// Static files (no authentication required)
		r.HandleFunc("/static/style.css", h.ServeStaticFile("style.css"))
		r.HandleFunc("/static/dashboard.js", h.ServeStaticFile("dashboard.js"))
		r.HandleFunc("/static/admin.js", h.ServeStaticFile("admin.js"))
		// Asset files (no authentication required)
		r.HandleFunc("/assets/connect.svg", h.ServeAssetFile("connect.svg"))
		r.HandleFunc("/assets/icon.svg", h.ServeAssetFile("icon.svg"))

		// only enable un-auth endpoint for openid only config
		if !conf.Server.KerberosEnabled() && !conf.Server.BasicAuthEnabled() && !conf.Server.NtlmEnabled() && !conf.Server.HeaderEnabled() {
			rdp.Name("gw").HandlerFunc(gw.HandleGatewayProtocol)
		}
	}

	// header auth (configurable proxy)
	if conf.Server.HeaderEnabled() {
		log.Printf("enabling header authentication with user header: %s", conf.Header.UserHeader)
		headerConfig := &web.HeaderConfig{
			UserHeader:        conf.Header.UserHeader,
			UserIdHeader:      conf.Header.UserIdHeader,
			EmailHeader:       conf.Header.EmailHeader,
			DisplayNameHeader: conf.Header.DisplayNameHeader,
		}
		headerAuth := headerConfig.New()
		r.Handle("/connect", headerAuth.Authenticated(http.HandlerFunc(h.HandleDownload)))

		// Web interface and API routes (authenticated)
		r.Handle("/", headerAuth.Authenticated(http.HandlerFunc(h.HandleWebInterface)))
		api.Handle("/hosts", headerAuth.Authenticated(http.HandlerFunc(h.HandleHostList)))
		api.Handle("/user", headerAuth.Authenticated(http.HandlerFunc(h.HandleUserInfo)))

		// Static files (no authentication required)
		r.HandleFunc("/static/style.css", h.ServeStaticFile("style.css"))
		r.HandleFunc("/static/app.js", h.ServeStaticFile("app.js"))
		// Asset files (no authentication required)
		r.HandleFunc("/assets/connect.svg", h.ServeAssetFile("connect.svg"))
		r.HandleFunc("/assets/icon.svg", h.ServeAssetFile("icon.svg"))

		// only enable un-auth endpoint for header only config
		if !conf.Server.KerberosEnabled() && !conf.Server.BasicAuthEnabled() && !conf.Server.NtlmEnabled() && !conf.Server.OpenIDEnabled() {
			rdp.Name("gw").HandlerFunc(gw.HandleGatewayProtocol)
		}
	}

	// for stacking of authentication
	auth := web.NewAuthMux()
	rdp.MatcherFunc(web.NoAuthz).HandlerFunc(auth.SetAuthenticate)

	// ntlm
	if conf.Server.NtlmEnabled() {
		log.Printf("enabling NTLM authentication")
		ntlm := web.NTLMAuthHandler{SocketAddress: conf.Server.AuthSocket, Timeout: conf.Server.BasicAuthTimeout}
		rdp.NewRoute().HeadersRegexp("Authorization", "NTLM").HandlerFunc(ntlm.NTLMAuth(gw.HandleGatewayProtocol))
		rdp.NewRoute().HeadersRegexp("Authorization", "Negotiate").HandlerFunc(ntlm.NTLMAuth(gw.HandleGatewayProtocol))
		auth.Register([]string{`NTLM`, `Negotiate`}, func(r *http.Request) bool {
			return r.Header.Get("Sec-WebSocket-Protocol") != "binary" // rdp client for ios is incompatible with this NTLM method.
		})
	}

	// basic auth
	if conf.Server.BasicAuthEnabled() {
		log.Printf("enabling basic authentication")
		q := web.BasicAuthHandler{SocketAddress: conf.Server.AuthSocket, Timeout: conf.Server.BasicAuthTimeout}
		rdp.NewRoute().HeadersRegexp("Authorization", "Basic").HandlerFunc(q.BasicAuth(gw.HandleGatewayProtocol))
		auth.Register([]string{`Basic realm="restricted", charset="UTF-8"`}, nil)
	}

	// spnego / kerberos
	if conf.Server.KerberosEnabled() {
		log.Printf("enabling kerberos authentication")
		keytab, err := keytab.Load(conf.Kerberos.Keytab)
		if err != nil {
			log.Fatalf("Cannot load keytab: %s", err)
		}
		rdp.NewRoute().HeadersRegexp("Authorization", "Negotiate").Handler(
			spnego.SPNEGOKRB5Authenticate(web.TransposeSPNEGOContext(http.HandlerFunc(gw.HandleGatewayProtocol)),
				keytab,
				service.Logger(log.Default())))

		// kdcproxy
		k := kdcproxy.InitKdcProxy(conf.Kerberos.Krb5Conf)
		r.HandleFunc(kdcProxyEndPoint, k.Handler).Methods("POST")
		auth.Register([]string{"Negotiate"}, nil)
	}

	// setup server
	server := http.Server{
		Addr:         ":" + strconv.Itoa(conf.Server.Port),
		Handler:      r,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)), // disable http2
	}

	if conf.Server.Tls == config.TlsDisable {
		err = server.ListenAndServe()
	} else {
		err = server.ListenAndServeTLS("", "")
	}
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
