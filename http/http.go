package http

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/health-monitor/conf"
	"github.com/health-monitor/logger"
	"github.com/health-monitor/metric"
	"github.com/health-monitor/timer"
	"golang.org/x/net/http2"
)

var (
	defaultTransport        = http.DefaultTransport.(*http.Transport)
	defaultProxyFuncPointer = fmt.Sprintf("%p", http.ProxyFromEnvironment)
)

type (
	// ProbeHTTP Transport
	ProbeHTTP struct {
		transport               *http.Transport
		followNonLocalRedirects bool
	}
)

const (
	httpPackage = "http"
	// HTTPTimerType must be unique across probes
	HTTPTimerType = 3001
	// dockerTimerSubtype normal processing probes
	httpTimerSubtype = 3002
	// HTTPTimerRetry Retry logic enabled
	httpTimerRetry = 3003
	// httpTimerWait Wail docker daemon recovers
	httpTimerWait = 3004
)

// EncodeURL format URL components to facilitate connection
func EncodeURL(scheme string, host string, port int, path string) *url.URL {
	return &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   path,
	}
}

// chkDefault checks to see if the transportProxyFunc is pointing to the default one
func chkDefault(transportProxier func(*http.Request) (*url.URL, error)) bool {
	transportProxierPointer := fmt.Sprintf("%p", transportProxier)
	return transportProxierPointer == defaultProxyFuncPointer
}

// TransportDefaults applies the defaults from http.DefaultTransport
func TransportDefaults(t *http.Transport) *http.Transport {
	if t.Proxy == nil || chkDefault(t.Proxy) {
		t.Proxy = NewProxyWithNoProxyCIDR(http.ProxyFromEnvironment)
	}

	if t.DialContext == nil && t.Dial == nil {
		t.DialContext = defaultTransport.DialContext
	}

	if t.TLSHandshakeTimeout == 0 {
		t.TLSHandshakeTimeout = defaultTransport.TLSHandshakeTimeout
	}

	if t.IdleConnTimeout == 0 {
		t.IdleConnTimeout = defaultTransport.IdleConnTimeout
	}
	return t
}

// useHTTP2 check whether transport support HTTP2
func useHTTP2(t *http.Transport) bool {
	if t.TLSClientConfig == nil || len(t.TLSClientConfig.NextProtos) == 0 {
		return true
	}
	for _, p := range t.TLSClientConfig.NextProtos {
		if p == http2.NextProtoTLS {
			return true
		}
	}
	return false
}

// NewProxyWithNoProxyCIDR constructs a Proxier function that respects CIDRs
func NewProxyWithNoProxyCIDR(delegate func(req *http.Request) (*url.URL, error)) func(req *http.Request) (*url.URL, error) {
	noProxyEnv := os.Getenv("NO_PROXY")
	if noProxyEnv == "" {
		noProxyEnv = os.Getenv("no_proxy")
	}
	noProxyRules := strings.Split(noProxyEnv, ",")

	cidrs := []*net.IPNet{}
	for _, noProxyRule := range noProxyRules {
		_, cidr, _ := net.ParseCIDR(noProxyRule)
		if cidr != nil {
			cidrs = append(cidrs, cidr)
		}
	}

	if len(cidrs) == 0 {
		return delegate
	}

	return func(req *http.Request) (*url.URL, error) {
		ip := net.ParseIP(req.URL.Hostname())
		if ip == nil {
			return delegate(req)
		}

		for _, cidr := range cidrs {
			if cidr.Contains(ip) {
				return nil, nil
			}
		}
		return delegate(req)
	}
}

// SetTransportDefaults applies the defaults from http.DefaultTransport
func SetTransportDefaults(t *http.Transport) *http.Transport {
	t = TransportDefaults(t)
	if useHTTP2(t) {
		if err := http2.ConfigureTransport(t); err != nil {
			logger.Warning(fmt.Sprintf("Transport failed http2 configuration: %v", err))
		}
	}
	return t
}

// New creates Prober that will skip TLS verification while probing.
func New(followNonLocalRedirects bool) ProbeHTTP {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	return NewTLSConfig(tlsConfig, followNonLocalRedirects)
}

// NewTLSConfig takes tls config as parameter.
func NewTLSConfig(config *tls.Config, followNonLocalRedirects bool) ProbeHTTP {
	transport := SetTransportDefaults(
		&http.Transport{
			TLSClientConfig:   config,
			DisableKeepAlives: true,
			Proxy:             http.ProxyURL(nil),
		})
	return ProbeHTTP{transport, followNonLocalRedirects}
}

// Method interface for making HTTP requests
type Method interface {
	Do(req *http.Request) (*http.Response, error)
}

// bounceEnabled Should we bounce this serivice daemon
func bounceEnabled(conf *conf.Conf) bool {
	if conf.Env.ActionFatal {
		return true
	}
	armTimer(conf)
	return false
}

// bounceService running docker daemon
func bounceService(conf *conf.Conf) {
	if bounceEnabled(conf) {
		cmd := exec.Command("systemctl", "kill", "kubelet")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			logger.Err(err.Error())
			panic(err)
		} else {
			logger.Info(out.String())
			metric.IncrementRestartCount()
			timer.Launch(
				timer.Name(conf.Env.Name),
				timer.Timeout(conf.Env.Interval),
				timer.Type(HTTPTimerType),
				timer.SubType(httpTimerWait),
				timer.User(conf),
			)
		}
	}
}

// serviceRestart Check whether service needs restarting
func restartService(conf *conf.Conf) {
	conf.RetryCounter++
	if conf.RetryCounter > conf.Env.Retries {
		conf.RetryCounter = 0
		logger.Warning(fmt.Sprintf(
			"Name: %s Package: %s Retries Exceeded Max: %d Curr: %d",
			conf.Env.Name,
			conf.Env.Package,
			conf.Env.Retries,
			conf.RetryCounter,
		))
		bounceService(conf)
	} else {
		timer.Launch(
			timer.Name(conf.Env.Name),
			timer.Timeout(conf.Env.Interval),
			timer.Type(HTTPTimerType),
			timer.SubType(httpTimerRetry),
			timer.User(conf),
		)
	}
}

func armTimer(conf *conf.Conf) {
	timer.Launch(
		timer.Name(conf.Env.Name),
		timer.Timeout(conf.Env.Interval),
		timer.Type(HTTPTimerType),
		timer.SubType(httpTimerSubtype),
		timer.User(conf),
	)
}

// Client create and initiate HTTP check.
func (p ProbeHTTP) Client(conf *conf.Conf) (err error) {
	err = ProbeURL(
		EncodeURL(
			strings.ToUpper(conf.Env.Package),
			conf.Env.IP,
			conf.Env.Port,
			conf.Env.Path),
		strings.ToLower(conf.Env.Package),
		&http.Client{
			Timeout:       time.Duration(conf.Env.Interval) * time.Second,
			Transport:     p.transport,
			CheckRedirect: redirects(p.followNonLocalRedirects),
		},
		conf.Env.Response,
	)
	return
}

// ProbeURL checks whether http method to the url succeeds.
func ProbeURL(url *url.URL, method string, client Method, response string) error {
	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if !strings.Contains(res.Status, response) {
		return fmt.Errorf(res.Status)
	}
	metric.SetKubeletMetric(1)
	return err
}

// redirects Follows non-local redirects
func redirects(followRedirects bool) func(*http.Request, []*http.Request) error {
	if followRedirects {
		return nil
	}

	return func(req *http.Request, via []*http.Request) error {
		if req.URL.Hostname() != via[0].URL.Hostname() {
			logger.Info(http.ErrUseLastResponse.Error())
			return http.ErrUseLastResponse
		}
		if len(via) >= 10 {
			err := errors.New("stopped after 10 redirects")
			logger.Warning(err.Error())
			return err
		}
		return nil
	}
}

// Run launch specified client timer(s)
func Run() {
	conf.Run(
		httpPackage,
		HTTPTimerType,
		httpTimerSubtype,
	)
}

// Probe specified HTTP endpoint
func Probe(tle *timer.TLE) {
	if conf, ok := tle.User.(*conf.Conf); ok {
		p := New(false)
		if err := p.Client(conf); err != nil {
			metric.SetKubeletMetric(0)
			logger.Warning(err.Error())
			restartService(conf)
		} else {
			metric.SetKubeletMetric(1)
			conf.RetryCounter = 0
			armTimer(conf)
		}
	}
}
