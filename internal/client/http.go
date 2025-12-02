package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"golang.org/x/net/proxy"
)

// NewHTTPClient returns an http.Client that can optionally use a proxy based on the setting.
// When useProxy is false, the client bypasses any proxy configuration.
func NewHTTPClient(useProxy bool) (*http.Client, error) {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default transport is not *http.Transport")
	}
	cloned := transport.Clone()

	if !useProxy {
		cloned.Proxy = nil
		return &http.Client{Transport: cloned}, nil
	}

	proxyURLStr, err := op.SettingGetString(model.SettingKeyProxyURL)
	if err != nil {
		return nil, err
	}
	if proxyURLStr == "" {
		return nil, fmt.Errorf("proxy url is empty")
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy url: %w", err)
	}

	switch proxyURL.Scheme {
	case "http", "https":
		cloned.Proxy = http.ProxyURL(proxyURL)
	case "socks", "socks5":
		socksDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("invalid socks proxy: %w", err)
		}
		cloned.Proxy = nil
		cloned.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return socksDialer.Dial(network, addr)
		}
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", proxyURL.Scheme)
	}

	return &http.Client{Transport: cloned}, nil
}
