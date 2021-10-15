package networkmanager

const (
	VersionProperty = "org.freedesktop.NetworkManager.Version"
)

func (client *Client) GetVersion() (string, error) {
	variant, err := client.conn.Object(InterfacePath, ObjectPath).GetProperty(VersionProperty)
	if err != nil {
		return "", err
	}
	return variant.Value().(string), nil
}
