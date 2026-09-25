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

package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

func TestHealthServerWhileWaitingForNodeLock(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "health.sock")
	lockPath := filepath.Join(t.TempDir(), "handler.lock")
	t.Setenv("NMSTATE_INSTANCE_NODE_LOCK_FILE", lockPath)
	lock := flock.New(lockPath)
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiting := make(chan struct{})
	done := make(chan int, 1)
	go func() {
		done <- runWithHealthServer(ctx, socketPath, func(ctx context.Context) int {
			close(waiting)
			// A second handler must remain live while waiting for the first.
			if _, err := lockHandler(ctx); err == nil {
				return generalExitStatus
			}
			return 0
		})
	}()
	select {
	case <-waiting:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not reach the lock wait")
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200", resp.StatusCode)
	}
	select {
	case code := <-done:
		t.Fatalf("handler exited during lock wait: %d", code)
	default:
	}

	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("handler exit code = %d", code)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("handler did not shut down")
	}
	if _, err := os.Lstat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("health socket was not removed: %v", err)
	}
	if resp, err := client.Do(req); err == nil {
		resp.Body.Close()
		t.Fatal("health endpoint still responds after shutdown")
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

	server, err := newHealthServer(context.Background(), socketPath)
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
	if server, err := newHealthServer(context.Background(), socketPath); err == nil {
		server.Listener.Close()
		t.Fatal("expected an error for a non-socket file")
	}
	content, err := os.ReadFile(socketPath)
	if err != nil || string(content) != "keep" {
		t.Fatalf("existing file was changed: %q, %v", content, err)
	}
}

func TestHealthServerPropagatesHandlerExit(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "health.sock")
	code := runWithHealthServer(context.Background(), socketPath, func(context.Context) int {
		return generalExitStatus
	})
	if code != generalExitStatus {
		t.Fatalf("handler exit code = %d", code)
	}
	if _, err := os.Lstat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("health socket was not removed: %v", err)
	}
}
