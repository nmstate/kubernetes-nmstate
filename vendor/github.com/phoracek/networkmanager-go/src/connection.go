package networkmanager

import (
	"github.com/godbus/dbus"
)

const (
	ActiveConnectionsProperty          = "org.freedesktop.NetworkManager.ActiveConnections"
	ActiveConnectionConnectionProperty = "org.freedesktop.NetworkManager.Connection.Active.Connection"
	SettingsObjectPath                 = "/org/freedesktop/NetworkManager/Settings"
	SettingsListConnectionsCall        = "org.freedesktop.NetworkManager.Settings.ListConnections"
	SettingsAddConnectionCall          = "org.freedesktop.NetworkManager.Settings.AddConnection"
	SettingsConnectionGetSettingsCall  = "org.freedesktop.NetworkManager.Settings.Connection.GetSettings"
	SettingsConnectionUpdateCall       = "org.freedesktop.NetworkManager.Settings.Connection.Update"
	SettingsConnectionDeleteCall       = "org.freedesktop.NetworkManager.Settings.Connection.Delete"
	ActivateConnectionCall             = "org.freedesktop.NetworkManager.ActivateConnection"
	DeactivateConnectionCall           = "org.freedesktop.NetworkManager.DeactivateConnection"
)

type Connection struct {
	connectionObject dbus.BusObject
}

func (client *Client) newConnectionFromObject(connectionObject dbus.BusObject) *Connection {
	connection := new(Connection)
	connection.connectionObject = connectionObject
	return connection
}
func (client *Client) newConnectionFromPath(connectionPath dbus.ObjectPath) *Connection {
	connectionObject := client.conn.Object(InterfacePath, connectionPath)
	connection := client.newConnectionFromObject(connectionObject)
	return connection
}

type ActiveConnection struct {
	activeConnectionObject dbus.BusObject
	Connection             *Connection
}

func (client *Client) newActiveConnectionFromPath(activeConnectionPath dbus.ObjectPath) *ActiveConnection {
	activeConnection := new(ActiveConnection)
	activeConnection.activeConnectionObject = client.conn.Object(InterfacePath, activeConnectionPath)
	connectionProperty, _ := activeConnection.activeConnectionObject.GetProperty(ActiveConnectionConnectionProperty)
	connection := client.newConnectionFromPath(connectionProperty.Value().(dbus.ObjectPath))
	activeConnection.Connection = connection
	return activeConnection
}

func (client *Client) ListConnections() []*Connection {
	call := client.conn.Object(InterfacePath, SettingsObjectPath).Call(SettingsListConnectionsCall, 0)
	check(call.Err)
	var connectionsPaths []dbus.ObjectPath
	call.Store(&connectionsPaths)

	connections := make([]*Connection, 0, len(connectionsPaths))
	for _, connectionPath := range connectionsPaths {
		connection := client.newConnectionFromPath(connectionPath)
		connections = append(connections, connection)
	}
	return connections
}

func (client *Client) ListActiveConnections() []*ActiveConnection {
	// XXX possible bug, we may need to refresh object property
	activeConnectionsPathsVariant, _ := client.conn.Object(InterfacePath, ObjectPath).GetProperty(ActiveConnectionsProperty)
	activeConnectionsPaths := activeConnectionsPathsVariant.Value().([]dbus.ObjectPath)

	activeConnections := make([]*ActiveConnection, 0, len(activeConnectionsPaths))
	for _, activeConnectionPath := range activeConnectionsPaths {
		activeConnection := client.newActiveConnectionFromPath(activeConnectionPath)
		activeConnections = append(activeConnections, activeConnection)
	}
	return activeConnections
}

func (client *Client) AddConnection(settings map[string]map[string]interface{}) error {
	dbusSettings := NativeSettingsToDbus(settings)
	return client.conn.Object(InterfacePath, SettingsObjectPath).Call(SettingsAddConnectionCall, 0, dbusSettings).Err
}

func (client *Client) ActivateConnection(connection *Connection) error {
	return client.conn.Object(InterfacePath, ObjectPath).Call(ActivateConnectionCall, 0, connection.connectionObject.Path(), dbus.ObjectPath("/"), dbus.ObjectPath("/")).Err
}

func (client *Client) DeactivateConnection(activeConnection *ActiveConnection) error {
	return client.conn.Object(InterfacePath, ObjectPath).Call(DeactivateConnectionCall, 0, activeConnection.activeConnectionObject.Path()).Err
}

func (connection *Connection) GetSettings() (map[string]map[string]interface{}, error) {
	call := connection.connectionObject.Call(SettingsConnectionGetSettingsCall, 0)
	if call.Err != nil {
		return nil, call.Err
	}
	var dbusSettings map[string]map[string]dbus.Variant
	call.Store(&dbusSettings)
	return DbusSettingsToNative(dbusSettings), nil
}

func (connection *Connection) Update(newSettings map[string]map[string]interface{}) error {
	newDbusSettings := NativeSettingsToDbus(newSettings)
	return connection.connectionObject.Call(SettingsConnectionUpdateCall, 0, newDbusSettings).Err
}

func (connection *Connection) Delete() error {
	return connection.connectionObject.Call(SettingsConnectionDeleteCall, 0).Err
}
