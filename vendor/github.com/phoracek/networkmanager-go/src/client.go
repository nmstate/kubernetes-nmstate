package networkmanager

import (
	"github.com/godbus/dbus"
)

const (
	InterfacePath = "org.freedesktop.NetworkManager"
	ObjectPath    = "/org/freedesktop/NetworkManager"
)

type Client struct {
	conn *dbus.Conn
}

func NewClient() *Client {
	client := new(Client)
	dbusConn, err := dbus.SystemBus()
	check(err)
	client.conn = dbusConn
	return client
}

func (client *Client) Close() {
	client.conn.Close()
	client.conn = nil
}
