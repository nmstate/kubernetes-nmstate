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
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	healthSocket        = "/run/nmstate-health/health.sock"
	healthDirectoryMode = 0700
)

func NewUnixSocketServer(ctx context.Context) (*manager.Server, error) {
	return newUnixSocketServer(ctx, healthSocket)
}

func newUnixSocketServer(ctx context.Context, socketPath string) (*manager.Server, error) {
	if err := os.MkdirAll(filepath.Dir(socketPath), healthDirectoryMode); err != nil {
		return nil, err
	}
	// This directory belongs to the single handler process in this container.
	// Recover a socket left behind by an unclean exit without deleting other files.
	if info, err := os.Lstat(socketPath); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("health socket path %s exists and is not a socket", socketPath)
		}
		if err := os.Remove(socketPath); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "unix", socketPath)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.StripPrefix("/healthz", &healthz.Handler{
		Checks: map[string]healthz.Checker{"ping": healthz.Ping},
	}))
	mux.Handle("/readyz", http.StripPrefix("/readyz", &healthz.Handler{
		Checks: map[string]healthz.Checker{"ping": healthz.Ping},
	}))
	shutdownTimeout := 5 * time.Second
	return &manager.Server{
		Name: "handler-health",
		Server: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
		Listener:        listener,
		ShutdownTimeout: &shutdownTimeout,
	}, nil
}
