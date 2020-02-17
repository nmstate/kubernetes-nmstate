package networkmanager

import (
	"github.com/godbus/dbus"
)

var variantDecoders = map[string]func(string, dbus.Variant) interface{}{
	"connection":     decodeConnectionVariant,
	"802-3-ethernet": decodeEthernetVariant,
	"ipv4":           decodeIpv4Variant,
}

func DbusSettingsToNative(dbusSettings map[string]map[string]dbus.Variant) map[string]map[string]interface{} {
	settings := make(map[string]map[string]interface{})
	for groupKey, groupValue := range dbusSettings {
		groupSettings := make(map[string]interface{})
		for attributeKey, attributeValue := range groupValue {
			if variantDecoder, ok := variantDecoders[groupKey]; ok {
				decodedVariant := variantDecoder(attributeKey, attributeValue)
				if decodedVariant != nil {
					groupSettings[attributeKey] = decodedVariant
				}
			}
		}
		if len(groupSettings) > 0 {
			settings[groupKey] = groupSettings
		}
	}
	return settings
}

func decodeConnectionVariant(name string, variant dbus.Variant) interface{} {
	switch name {
	case "autoconnect", "autoconnect-priority", "autoconnect-slaves", "lldp", "id", "interface-name", "type", "uuid", "slave-type", "master":
		return variant.Value()
	default:
		return nil
	}
}

func decodeEthernetVariant(name string, variant dbus.Variant) interface{} {
	switch name {
	case "mtu", "mac-address":
		return variant.Value()
	default:
		return nil
	}
}

func decodeIpv4Variant(name string, variant dbus.Variant) interface{} {
	switch name {
	case "ignore-auto-routes", "ignore-auto-dns", "method", "gateway", "dns", "addresses", "routes":
		return variant.Value()
	case "address-data", "route-data":
		dbusData := variant.Value().([]map[string]dbus.Variant)
		nativeData := make([]map[string]interface{}, len(dbusData))
		for i, dbusItem := range dbusData {
			nativeData[i] = make(map[string]interface{}, len(dbusItem))
			for k, dbusValue := range dbusItem {
				nativeData[i][k] = dbusValue.Value()
			}
		}
		return nativeData
	default:
		return nil
	}
}

var variantEncoders = map[string]func(string, interface{}) (dbus.Variant, bool){
	"connection":     encodeConnectionVariant,
	"802-3-ethernet": encodeEthernetVariant,
	"ipv4":           encodeIpv4Variant,
}

func NativeSettingsToDbus(settings map[string]map[string]interface{}) map[string]map[string]dbus.Variant {
	dbusSettings := make(map[string]map[string]dbus.Variant)
	for groupKey, groupValue := range settings {
		groupSettings := make(map[string]dbus.Variant)
		for attributeKey, attributeValue := range groupValue {
			if variantEncoder, ok := variantEncoders[groupKey]; ok {
				encodedVariant, encoded := variantEncoder(attributeKey, attributeValue)
				if encoded {
					groupSettings[attributeKey] = encodedVariant
				}
			}
		}
		if len(groupSettings) > 0 {
			dbusSettings[groupKey] = groupSettings
		}
	}
	return dbusSettings
}

func encodeConnectionVariant(name string, value interface{}) (dbus.Variant, bool) {
	switch name {
	case "autoconnect":
		return makeVariant(value, "b"), true
	case "autoconnect-priority", "autoconnect-slaves", "lldp":
		return makeVariant(value, "i"), true
	case "id", "interface-name", "type", "uuid", "slave-type", "master":
		return makeVariant(value, "s"), true
	default:
		return makeVariant(false, "b"), false
	}
}

func encodeEthernetVariant(name string, value interface{}) (dbus.Variant, bool) {
	switch name {
	case "mtu":
		return makeVariant(value, "i"), true
	case "mac-address":
		return makeVariant(value, "ay"), true
	default:
		return makeVariant(false, "b"), false
	}
}

func encodeIpv4Variant(name string, value interface{}) (dbus.Variant, bool) {
	switch name {
	case "ignore-auto-routes", "ignore-auto-dns":
		return makeVariant(value, "b"), true
	case "method", "gateway":
		return makeVariant(value, "s"), true
	case "dns":
		return makeVariant(value, "au"), true
	case "addresses", "routes":
		return makeVariant(value, "aau"), true
	case "address-data", "route-data":
		nativeData := value.([]map[string]interface{})
		dbusData := make([]map[string]dbus.Variant, len(nativeData))
		for i, nativeItem := range nativeData {
			dbusData[i] = make(map[string]dbus.Variant, len(nativeItem))
			for k, nativeValue := range nativeItem {
				switch k {
				case "address", "dest":
					dbusData[i][k] = makeVariant(nativeValue, "s")
				case "prefix", "metric":
					dbusData[i][k] = makeVariant(nativeValue, "u")
				}
			}
		}
		return makeVariant(dbusData, "aa{sv}"), true
	default:
		return makeVariant(false, "b"), false
	}
}

func makeVariant(value interface{}, signature string) dbus.Variant {
	return dbus.MakeVariantWithSignature(value, dbus.ParseSignatureMust(signature))
}
