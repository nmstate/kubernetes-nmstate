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

package log

import (
	"strings"

	"github.com/go-logr/logr"
)

type Writer struct {
	logger logr.Logger
	level  int
}

func NewWriter(logger logr.Logger, level int) *Writer {
	return &Writer{
		logger: logger,
		level:  level,
	}
}

func (lw *Writer) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	lw.logger.V(lw.level).Info(message)
	return len(p), nil
}
