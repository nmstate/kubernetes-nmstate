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
