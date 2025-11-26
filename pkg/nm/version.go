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

package nm

import "github.com/godbus/dbus/v5"

const (
	interfacePath   = "org.freedesktop.NetworkManager"
	objectPath      = "/org/freedesktop/NetworkManager"
	versionProperty = "org.freedesktop.NetworkManager.Version"
)

func Version() (string, error) {
	dbusConn, err := dbus.SystemBusPrivate()
	if err != nil {
		return "", err
	}
	defer dbusConn.Close()

	if err := dbusConn.Auth(nil); err != nil {
		return "", err
	}

	if err := dbusConn.Hello(); err != nil {
		return "", err
	}

	variant, err := dbusConn.Object(interfacePath, objectPath).GetProperty(versionProperty)
	if err != nil {
		return "", err
	}
	return variant.Value().(string), nil
}
