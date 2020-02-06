package networkmanager

import (
	"github.com/godbus/dbus"
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

func (client *Client) CheckpointCreate(devices []*Device, timeout uint, flags uint) *Checkpoint {
	devicesPaths := make([]dbus.ObjectPath, len(devices))
	for _, device := range devices {
		devicesPaths = append(devicesPaths, device.deviceObject.Path())
	}

	call := client.conn.Object(InterfacePath, ObjectPath).Call(CheckpointCreateCall, 0, devicesPaths, timeout, flags)
	check(call.Err)
	var checkpointPath dbus.ObjectPath
	call.Store(&checkpointPath)

	checkpoint := &Checkpoint{
		checkpointObject: client.conn.Object(InterfacePath, checkpointPath),
	}
	return checkpoint
}

// TODO: wrap return value
func (client *Client) CheckpointRollback(checkpoint *Checkpoint) map[string]uint {
	call := client.conn.Object(InterfacePath, ObjectPath).Call(CheckpointRollbackCall, 0, checkpoint.checkpointObject.Path())
	check(call.Err)
	var rollbackResult map[string]uint
	call.Store(&rollbackResult)
	return rollbackResult
}

func (client *Client) CheckpointDestroy(checkpoint *Checkpoint) {
	err := client.conn.Object(InterfacePath, ObjectPath).Call(CheckpointDestroyCall, 0, checkpoint.checkpointObject.Path()).Err
	check(err)
}
