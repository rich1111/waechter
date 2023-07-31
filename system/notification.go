package system

import (
	"github.com/mtrossbach/waechter/internal/config"
	"github.com/mtrossbach/waechter/system/alarm"
	"github.com/mtrossbach/waechter/system/device"
	"github.com/mtrossbach/waechter/system/zone"
)

type NotificationAdapter interface {
	Name() string
	NotifyAlarm(person config.Person, systemName string, a alarm.Type, device device.Spec, zone zone.Zone) bool
	NotifyRecovery(person config.Person, systemName string, device device.Spec, zone zone.Zone) bool
	NotifyDeviceAvailable(person config.Person, systemName string, device device.Spec, zone zone.Zone) bool
	NotifyDeviceUnAvailable(person config.Person, systemName string, device device.Spec, zone zone.Zone) bool
	NotifyMotionSensor(person config.Person, systemName string, device device.Spec, zone zone.Zone, motion bool) bool
	NotifyContactSensor(person config.Person, systemName string, device device.Spec, zone zone.Zone, contact bool) bool
	NotifySmokeSensor(person config.Person, systemName string, device device.Spec, zone zone.Zone, smoke bool) bool
	NotifyBatteryLevel(person config.Person, systemName string, device device.Spec, zone zone.Zone, batteryLevel float32) bool
	NotifyLinkQuality(person config.Person, systemName string, device device.Spec, zone zone.Zone, quality float32) bool
	NotifyHumidityValue(person config.Person, systemName string, device device.Spec, zone zone.Zone, humidity float32) bool
	NotifyTemperatureValue(person config.Person, systemName string, device device.Spec, zone zone.Zone, temperature float32) bool
	NotifyAutoArm(person config.Person, systemName string) bool
	NotifyAutoDisarm(person config.Person, systemName string) bool
}
