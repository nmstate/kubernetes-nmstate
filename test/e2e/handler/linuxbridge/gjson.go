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

package linuxbridge

import (
	"strings"
)

// buildGJsonExpression returns the correct gjson expression
// to filter `bridge -j vlan show` by interface and vlan, there
// are two json versions possible it tries to guess which one is.
func BuildGJsonExpression(bridgeVlans string) string {
	if strings.Contains(bridgeVlans, "ifname") {
		return "#(ifname==%s).vlans.#(vlan==%d)"
	}
	return "%s.#(vlan==%d)"
}
