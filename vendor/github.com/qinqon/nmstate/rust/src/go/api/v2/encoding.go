package nmstate

import (
	"encoding/json"
	"fmt"
)

func (i Interface) MarshalJSON() ([]byte, error) {
	if i.LinuxBridge != nil {
		i.BaseInterface.IfaceType = InterfaceTypeLinuxBridge
		return json.Marshal(&struct {
			*BaseInterface
			*LinuxBridgeInterface
		}{
			&i.BaseInterface,
			i.LinuxBridge,
		})
	} else if i.Ethernet != nil {
		if i.BaseInterface.IfaceType == "" ||
			i.BaseInterface.IfaceType == InterfaceTypeUnknown {
			if i.Ethernet.Veth != nil {
				i.BaseInterface.IfaceType = InterfaceTypeVeth
			} else {
				i.BaseInterface.IfaceType = InterfaceTypeEthernet
			}
		}
		return json.Marshal(&struct {
			*BaseInterface
			*EthernetInterface
		}{
			&i.BaseInterface,
			i.Ethernet,
		})
	} else if i.Bond != nil {
		i.BaseInterface.IfaceType = InterfaceTypeBond
		return json.Marshal(&struct {
			*BaseInterface
			*BondInterface
		}{
			&i.BaseInterface,
			i.Bond,
		})
	} else if i.Dummy != nil {
		i.BaseInterface.IfaceType = InterfaceTypeDummy
		return json.Marshal(&struct {
			*BaseInterface
			*DummyInterface
		}{
			&i.BaseInterface,
			i.Dummy,
		})
	} else if i.Vlan != nil {
		i.BaseInterface.IfaceType = InterfaceTypeVlan
		return json.Marshal(&struct {
			*BaseInterface
			*VlanInterface
		}{
			&i.BaseInterface,
			i.Vlan,
		})
	} else if i.OvsBridge != nil {
		i.BaseInterface.IfaceType = InterfaceTypeOvsBridge
		return json.Marshal(&struct {
			*BaseInterface
			*OvsBridgeInterface
		}{
			&i.BaseInterface,
			i.OvsBridge,
		})
	} else if i.OvsInterface != nil {
		i.BaseInterface.IfaceType = InterfaceTypeOvsInterface
		return json.Marshal(&struct {
			*BaseInterface
			*OvsInterface
		}{
			&i.BaseInterface,
			i.OvsInterface,
		})
	} else if i.MacVlan != nil {
		i.BaseInterface.IfaceType = InterfaceTypeMacVlan
		return json.Marshal(&struct {
			*BaseInterface
			*MacVlanInterface
		}{
			&i.BaseInterface,
			i.MacVlan,
		})
	} else if i.MacVtap != nil {
		i.BaseInterface.IfaceType = InterfaceTypeMacVtap
		return json.Marshal(&struct {
			*BaseInterface
			*MacVtapInterface
		}{
			&i.BaseInterface,
			i.MacVtap,
		})
	} else if i.Vrf != nil {
		i.BaseInterface.IfaceType = InterfaceTypeVrf
		return json.Marshal(&struct {
			*BaseInterface
			*VrfInterface
		}{
			&i.BaseInterface,
			i.Vrf,
		})
	} else if i.InfiniBand != nil {
		i.BaseInterface.IfaceType = InterfaceTypeInfiniBand
		return json.Marshal(&struct {
			*BaseInterface
			*InfiniBandInterface
		}{
			&i.BaseInterface,
			i.InfiniBand,
		})
	} else if i.Unknown != nil {
		return json.Marshal(&struct {
			*BaseInterface
			*UnknownInterface
		}{
			&i.BaseInterface,
			i.Unknown,
		})
	}
	return nil, fmt.Errorf("unknown interface")
}

func (i *Interface) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &i.BaseInterface); err != nil {
		return err
	}
	if i.BaseInterface.IfaceType == InterfaceTypeLinuxBridge {
		i.LinuxBridge = &LinuxBridgeInterface{}
		if err := json.Unmarshal(data, i.LinuxBridge); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeEthernet {
		i.Ethernet = &EthernetInterface{}
		if err := json.Unmarshal(data, i.Ethernet); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeBond {
		i.Bond = &BondInterface{}
		if err := json.Unmarshal(data, i.Bond); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeVlan {
		i.Vlan = &VlanInterface{}
		if err := json.Unmarshal(data, i.Vlan); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeOvsBridge {
		i.OvsBridge = &OvsBridgeInterface{}
		if err := json.Unmarshal(data, i.OvsBridge); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeOvsInterface {
		i.OvsInterface = &OvsInterface{}
		if err := json.Unmarshal(data, i.OvsInterface); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeMacVlan {
		i.MacVlan = &MacVlanInterface{}
		if err := json.Unmarshal(data, i.MacVlan); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeMacVtap {
		i.MacVtap = &MacVtapInterface{}
		if err := json.Unmarshal(data, i.MacVtap); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeVrf {
		i.Vrf = &VrfInterface{}
		if err := json.Unmarshal(data, i.Vrf); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeVeth {
		i.Ethernet = &EthernetInterface{}
		if err := json.Unmarshal(data, i.Ethernet); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeInfiniBand {
		i.InfiniBand = &InfiniBandInterface{}
		if err := json.Unmarshal(data, i.InfiniBand); err != nil {
			return err
		}
		return nil
	} else if i.BaseInterface.IfaceType == InterfaceTypeDummy {
		i.Dummy = &DummyInterface{}
		if err := json.Unmarshal(data, i.Dummy); err != nil {
			return err
		}
		return nil

	} else {
		i.Unknown = &UnknownInterface{}
		if err := json.Unmarshal(data, i.Unknown); err != nil {
			return err
		}
		return nil
	}
}
