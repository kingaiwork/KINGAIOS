package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	Socket  string
	Timeout time.Duration
}

func (c Client) Execute(ctx context.Context, req Request) (Result, error) {
	if c.Socket == "" { c.Socket = "/run/kingai-execd/execd.sock" }
	if !filepath.IsAbs(c.Socket) || strings.ContainsAny(c.Socket, "\x00\n\r") {
		return Result{}, errors.New("execution daemon socket must be an absolute local path")
	}
	if c.Timeout <= 0 { c.Timeout = 35 * time.Second }
	body, err := json.Marshal(req)
	if err != nil { return Result{}, err }
	reqCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", c.Socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: c.Timeout}
	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, "http://kingai-execd/v1/execute", bytes.NewReader(body))
	if err != nil { return Result{}, err }
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil { return Result{}, err }
	defer resp.Body.Close()
	if resp.StatusCode >= 300 { return Result{}, fmt.Errorf("execution daemon returned %s", resp.Status) }
	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return Result{}, err }
	if !result.OK { return result, errors.New(result.Message) }
	return result, nil
}

func (c Client) Health(ctx context.Context) error {
	if c.Socket == "" { c.Socket = "/run/kingai-execd/execd.sock" }
	if !filepath.IsAbs(c.Socket) || strings.ContainsAny(c.Socket, "\x00\n\r") { return errors.New("execution daemon socket must be an absolute local path") }
	if c.Timeout <= 0 { c.Timeout = 5 * time.Second }
	reqCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return (&net.Dialer{}).DialContext(ctx, "unix", c.Socket) }}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: c.Timeout}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "http://kingai-execd/healthz", nil)
	if err != nil { return err }
	resp, err := client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode >= 300 { return fmt.Errorf("execution daemon returned %s", resp.Status) }
	return nil
}
