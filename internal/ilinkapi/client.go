package ilinkapi

import (
	"net/http"

	"github.com/wonli/apic/v2"
)

// Client 是对 iLink HTTP / CDN 能力的最底层封装。
// 它本身不持有状态，主要负责承载公共能力。
// 具体业务逻辑由各 api_*.go 中的接口对象实现，再按需复用 Client 的基础方法。
type Client struct {
	apic.Apic
	apiClient           *apic.ApiClients
	maxRemoteMediaBytes int64
	allowPrivateHosts   bool
	allowHosts          map[string]struct{}
}

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		if httpClient != nil {
			c.apiClient = c.apiClient.WithHTTPClient(httpClient)
		}
	}
}

func WithRemoteMediaMaxBytes(limit int64) ClientOption {
	return func(c *Client) {
		if limit > 0 {
			c.maxRemoteMediaBytes = limit
		}
	}
}

func WithRemoteMediaAllowedHosts(hosts ...string) ClientOption {
	return func(c *Client) {
		if c.allowHosts == nil {
			c.allowHosts = map[string]struct{}{}
		}
		for _, host := range hosts {
			host = normalizeHost(host)
			if host == "" {
				continue
			}
			c.allowHosts[host] = struct{}{}
		}
	}
}

func WithRemoteMediaAllowPrivateHosts() ClientOption {
	return func(c *Client) {
		c.allowPrivateHosts = true
	}
}

// NewClient 创建一个可复用的 iLink 底层客户端。
func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		apiClient:           apic.NewApiClient(),
		maxRemoteMediaBytes: 20 << 20,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}
