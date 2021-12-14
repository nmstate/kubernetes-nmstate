package networkmanager

import (
	"github.com/godbus/dbus/v5"
)

const (
	CheckpointCreateCall   = "org.freedesktop.NetworkManager.CheckpointCreate"
	CheckpointDestroyCall  = "org.freedesktop.NetworkManager.CheckpointDestroy"
	CheckpointRollbackCall = "org.freedesktop.NetworkManager.CheckpointRollback"

	NmCheckpointCreateFlagNone                 = 0x00
	NmCheckpointCreateFlagDestroyAll           = 0x01
	NmCheckpointCreateFlagDeleteNewConnections = 0x02
	NmCheckpointCreateFlagDisconnectNewDevices = 0x03

	NmCheckpointNoTimeout = 0
)

type Checkpoint struct {
	checkpointObject dbus.BusObject
}

func (client *Client) CheckpointCreate(devices []*Device, timeout uint, flags uint) (*Checkpoint, error) {
	devicesPaths := make([]dbus.ObjectPath, len(devices))
	for _, device := range devices {
		devicesPaths = append(devicesPaths, device.deviceObject.Path())
	}

	var checkpointPath dbus.ObjectPath

	call := client.conn.Object(InterfacePath, ObjectPath).Call(CheckpointCreateCall, 0, devicesPaths, timeout, flags)
	if call.Err != nil {
		return nil, call.Err
	}

	call.Store(&checkpointPath)

	checkpoint := &Checkpoint{
		checkpointObject: client.conn.Object(InterfacePath, checkpointPath),
	}
	return checkpoint, nil
}

// TODO: wrap return value
func (client *Client) CheckpointRollback(checkpoint *Checkpoint) (map[string]uint, error) {
	var rollbackResult map[string]uint

	call := client.conn.Object(InterfacePath, ObjectPath).Call(CheckpointRollbackCall, 0, checkpoint.checkpointObject.Path())
	if call.Err != nil {
		return rollbackResult, call.Err
	}

	call.Store(&rollbackResult)

	return rollbackResult, nil
}

func (client *Client) CheckpointDestroy(checkpoint *Checkpoint) error {
	err := client.conn.Object(InterfacePath, ObjectPath).Call(CheckpointDestroyCall, 0, checkpoint.checkpointObject.Path()).Err
	return err
}
