package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client communicates with the WoL Agent running on OpenWrt via Rathole tunnel
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// PingResult represents the result of a ping check
type PingResult struct {
	Alive bool   `json:"alive"`
	IP    string `json:"ip"`
}

// WakeResult represents the result of a wake command
type WakeResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// HealthResult represents the agent health status
type HealthResult struct {
	Status   string `json:"status"`
	Hostname string `json:"hostname"`
	Uptime   string `json:"uptime"`
}

// BootStatus represents the boot monitoring progress
type BootStatus struct {
	Online  bool   `json:"online"`
	Message string `json:"message"`
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Wake sends a magic packet via the OpenWrt agent
func (c *Client) Wake(mac, iface string) (*WakeResult, error) {
	reqURL := fmt.Sprintf("%s/cgi-bin/wake?mac=%s&iface=%s&token=%s",
		c.baseURL,
		mac,
		url.QueryEscape(iface),
		url.QueryEscape(c.token),
	)
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	var result WakeResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid agent response: %w", err)
	}
	return &result, nil
}

// Ping checks if a device is online
func (c *Client) Ping(ip string) (*PingResult, error) {
	params := url.Values{
		"ip":    {ip},
		"token": {c.token},
	}
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/cgi-bin/ping?%s", c.baseURL, params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	var result PingResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid agent response: %w", err)
	}
	return &result, nil
}

// Health checks the agent health
func (c *Client) Health() (*HealthResult, error) {
	params := url.Values{
		"token": {c.token},
	}
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/cgi-bin/health?%s", c.baseURL, params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	var result HealthResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid agent response: %w", err)
	}
	return &result, nil
}
