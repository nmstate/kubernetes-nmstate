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

package nmstatectl

import (
	"errors"
	"fmt"
)

// NmstatectlApplyError represents an error that occurs during the nmstatectl apply operation.
// It wraps the error.
type NmstatectlApplyError struct {
	err error // The underlying error
}

// NewNmstatectlApplyError creates a new NmstatectlApplyError.
// It accepts an underlying error to wrap and returns an error that implements the NmstatectlApplyError type.
func NewNmstatectlApplyError(err error) error {
	return &NmstatectlApplyError{
		err: err, // The underlying error being wrapped.
	}
}

// Error implements the error interface for NmstatectlApplyError.
// It returns a formatted string representation of the error.
func (e *NmstatectlApplyError) Error() string {
	return fmt.Sprintf("nmstatectl apply error: %v", e.err)
}

// IsNmstatectlApplyError checks if the given error is of type NmstatectlApplyError.
// It uses errors.As to determine if the error or any wrapped error matches.
func IsNmstatectlApplyError(err error) bool {
	var target *NmstatectlApplyError
	return errors.As(err, &target)
}
