package fath

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to a free@home SysAP's local REST API.
type Client struct {
	baseURL string
	user    string
	pass    string
	http    *http.Client
}

// Options configures a Client.
type Options struct {
	// Host is the SysAP's address, e.g. "192.168.1.50" or "192.168.1.50:443".
	Host string
	// Scheme is "https" (default, and what real SysAPs expect) or "http".
	Scheme string
	// APIBase is the REST API path prefix, defaults to "/fhapi/v1/api/rest".
	APIBase string
	// User/Password are the local API credentials configured on the SysAP
	// (System -> free@home Settings -> Local API on the SysAP itself).
	User     string
	Password string
	// InsecureSkipVerify disables TLS certificate verification, which is
	// normally required because the SysAP presents a self-signed certificate.
	InsecureSkipVerify bool
	// Timeout bounds every HTTP request. Defaults to 10s.
	Timeout time.Duration
}

func NewClient(opts Options) *Client {
	scheme := opts.Scheme
	if scheme == "" {
		scheme = "https"
	}
	apiBase := opts.APIBase
	if apiBase == "" {
		apiBase = "/fhapi/v1/api/rest"
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.InsecureSkipVerify},
	}

	return &Client{
		baseURL: fmt.Sprintf("%s://%s%s", scheme, opts.Host, apiBase),
		user:    opts.User,
		pass:    opts.Password,
		http:    &http.Client{Transport: transport, Timeout: timeout},
	}
}

// GetConfiguration fetches the full data model (all devices, channels and
// their current datapoint values) from the SysAP.
func (c *Client) GetConfiguration(ctx context.Context) (Configuration, error) {
	body, err := c.do(ctx, http.MethodGet, "/configuration", nil, "")
	if err != nil {
		return nil, err
	}
	var cfg Configuration
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("decoding configuration response: %w", err)
	}
	return cfg, nil
}

// GetRawConfiguration fetches the same endpoint as GetConfiguration but
// returns the unparsed JSON body, which is useful when the SysAP's response
// doesn't match the shape assumed by the Configuration struct.
func (c *Client) GetRawConfiguration(ctx context.Context) ([]byte, error) {
	return c.do(ctx, http.MethodGet, "/configuration", nil, "")
}

// SetDatapoint writes a value to an input (command) datapoint, e.g. to switch
// an actuator on/off, move a blind, or set a dimming level. sysapID,
// deviceID, channelID and datapointID must match the keys used in the
// configuration response (e.g. the SysAP's UUID, "ABB700000001", "ch0000",
// "idp0000"). Per the free@home local API, the request body is the raw value
// itself (e.g. "1" or "70"), not a JSON object.
func (c *Client) SetDatapoint(ctx context.Context, sysapID, deviceID, channelID, datapointID, value string) error {
	path := fmt.Sprintf("/datapoint/%s/%s.%s.%s", sysapID, deviceID, channelID, datapointID)
	_, err := c.do(ctx, http.MethodPut, path, []byte(value), "application/x-www-form-urlencoded")
	return err
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, contentType string) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("Accept", "application/json")
	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", req.URL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed (HTTP 401): check the local API user/password and that 'Activate Local API' is enabled on the SysAP")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %s from %s: %s", resp.Status, req.URL, truncate(respBody, 500))
	}
	return respBody, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
