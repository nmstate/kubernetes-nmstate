/*
Copyright The Kubernetes NMState Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package health

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHealthServerLifecycle(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "health.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, err := newUnixSocketServer(ctx, socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Listener.Close()

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}

	t.Run("not ready before startup", func(t *testing.T) {
		probeCtx, cancelProbe := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancelProbe()
		request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, "http://localhost/readyz", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if response != nil {
			response.Body.Close()
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected probe to time out before startup, got %v", err)
		}
	})

	done := make(chan error, 1)
	go func() { done <- server.Start(ctx) }()
	defer func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("server shutdown failed: %v", err)
			}
		case <-time.After(6 * time.Second):
			server.Server.Close()
			t.Error("server did not shut down within its shutdown timeout")
		}
		if _, err := os.Lstat(socketPath); !os.IsNotExist(err) {
			t.Errorf("expected socket to be removed after shutdown, got %v", err)
		}
	}()

	for _, endpoint := range []string{"/readyz", "/healthz"} {
		t.Run(endpoint, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+endpoint, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusOK || string(body) != "ok" {
				t.Fatalf("expected 200 OK with body ok, got %d with body %q", response.StatusCode, body)
			}
		})
	}
}

func TestHealthServerRecoversStaleSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "health.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	listener.SetUnlinkOnClose(false)
	listener.Close()

	server, err := newUnixSocketServer(context.Background(), socketPath)
	if err != nil {
		t.Fatal(err)
	}
	server.Listener.Close()
}

func TestHealthServerPreservesNonSocketFile(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "health.sock")
	if err := os.WriteFile(socketPath, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if server, err := newUnixSocketServer(context.Background(), socketPath); err == nil {
		server.Listener.Close()
		t.Fatal("expected an error for a non-socket file")
	}
	content, err := os.ReadFile(socketPath)
	if err != nil || string(content) != "keep" {
		t.Fatalf("existing file was changed: %q, %v", content, err)
	}
}
