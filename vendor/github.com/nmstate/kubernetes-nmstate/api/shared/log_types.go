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

package shared

// LogLevel defines the log level for nmstate operations
type LogLevel string

const (
	// LogLevelInfo indicates info log level (default, minimal output)
	LogLevelInfo LogLevel = "info"
	// LogLevelDebug indicates debug log level (verbose output for debugging)
	LogLevelDebug LogLevel = "debug"
)
