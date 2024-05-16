package mikrotik

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-routeros/routeros"
)

type Client interface {
	GetTransport() TransportType
	// TODO Add Cancel Context
	SendRequest(method CrudMethod, url *URL, data map[string]string) ([]MikrotikItem, error)
	WithContext(ctx context.Context) context.Context
}

type CrudMethod int

const (
	CrudRead CrudMethod = iota
	CrudPost
	CrudMonitor
)

type Config struct {
	Insecure      bool
	CaCertificate string
	HostURL       string
	Username      string
	Password      string
}

func NewClient(ctx context.Context, conf *Config) (Client, error) {

	tlsConf := tls.Config{
		InsecureSkipVerify: conf.Insecure,
	}

	if tlsConf.InsecureSkipVerify && conf.CaCertificate != "" {
		return nil, errors.New("you have selected mutually exclusive options: ca_certificate and insecure connection")
	}

	if conf.CaCertificate != "" {
		if _, err := os.Stat(conf.CaCertificate); err != nil {
			LogMessage(ctx, DEBUG, "Failed to read CA file '"+conf.CaCertificate+"'", map[string]interface{}{"error": err})
			return nil, err
		}

		certPool := x509.NewCertPool()
		file, err := os.ReadFile(conf.CaCertificate)
		if err != nil {
			LogMessage(ctx, DEBUG, "Failed to read CA file '"+conf.CaCertificate+"'", map[string]interface{}{"error": err})
			return nil, fmt.Errorf("failed to read CA file '%s', %v", conf.CaCertificate, err)
		}
		certPool.AppendCertsFromPEM(file)
		tlsConf.RootCAs = certPool
	}

	routerUrl, err := url.Parse(conf.HostURL)
	if err != nil || routerUrl.Host == "" {
		routerUrl, err = url.Parse("https://" + conf.HostURL)
	}
	if err != nil {
		return nil, fmt.Errorf("error while parsing the router URL: '%v'", conf.HostURL)
	}
	routerUrl.Path = strings.TrimSuffix(routerUrl.Path, "/")

	var useTLS = true
	var transport = TransportREST

	// Parse URL.
	switch routerUrl.Scheme {
	case "https":
	case "apis":
		routerUrl.Scheme = ""
		if routerUrl.Port() == "" {
			routerUrl.Host += ":8729"
		}
		transport = TransportAPI
	case "api":
		routerUrl.Scheme = ""
		if routerUrl.Port() == "" {
			routerUrl.Host += ":8728"
		}
		useTLS = false
		transport = TransportAPI
	default:
		panic("[NewClient] wrong transport type: " + routerUrl.Scheme)
	}

	if transport == TransportAPI {
		api := &ApiClient{
			ctx:       ctx,
			HostURL:   routerUrl.Host,
			Username:  conf.Username,
			Password:  conf.Password,
			Transport: TransportAPI,
		}

		if useTLS {
			api.Client, err = routeros.DialTLS(api.HostURL, api.Username, api.Password, &tlsConf)
		} else {
			api.Client, err = routeros.Dial(api.HostURL, api.Username, api.Password)
		}
		if err != nil {
			return nil, err
		}

		// The synchronous client has an infinite wait issue
		// when an error occurs while creating multiple resources.
		api.Async()

		return api, nil
	}

	rest := &RestClient{
		ctx:       ctx,
		HostURL:   routerUrl.String(),
		Username:  conf.Username,
		Password:  conf.Password,
		Transport: TransportREST,
	}

	rest.Client = &http.Client{
		Timeout: time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tlsConf,
		},
	}

	return rest, nil
}

type ctxKey struct{}

func Ctx(ctx context.Context) Client {
	if c, ok := ctx.Value(ctxKey{}).(*RestClient); ok {
		return c
	} else if c, ok := ctx.Value(ctxKey{}).(*ApiClient); ok {
		return c
	}
	return nil
}

type URL struct {
	Path  string   // URL path without '/rest'.
	Query []string // Query values.
}

// GetApiCmd Returns the set of commands for the API client.
func (u *URL) GetApiCmd() []string {
	res := []string{u.Path}
	//if len(u.Query) > 0 && u.Query[len(u.Query) - 1] != "?#|" {
	//	u.Query = append(u.Query, "?#|")
	//}
	return append(res, u.Query...)
}

// GetRestURL Returns the URL for the client
func (u *URL) GetRestURL() string {
	q := strings.Join(u.Query, "&")
	if len(q) > 0 && q[0] != '?' {
		q = "?" + q
	}
	return u.Path + q
}

// EscapeChars peterGo https://groups.google.com/g/golang-nuts/c/NiQiAahnl5E/m/U60Sm1of-_YJ
func EscapeChars(data []byte) []byte {
	var u = []byte(`\u0000`)
	//var u = []byte(`U+0000`)
	var res = make([]byte, 0, len(data))

	for i, ch := range data {
		if ch < 0x20 {
			res = append(res, u...)
			hex.Encode(res[len(res)-2:], data[i:i+1])
			continue
		}
		res = append(res, ch)
	}
	return res
}
