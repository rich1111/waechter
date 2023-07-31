package system

import (
	"github.com/mtrossbach/waechter/internal/config"
	"github.com/mtrossbach/waechter/internal/log"
	"github.com/mtrossbach/waechter/system/alarm"
	"github.com/mtrossbach/waechter/system/device"
	"github.com/mtrossbach/waechter/system/zone"
)

type notificationManager struct {
	adapters []NotificationAdapter
}

func newNotificationManager() *notificationManager {
	return &notificationManager{adapters: []NotificationAdapter{}}
}

func (n *notificationManager) AddAdapter(adapter NotificationAdapter) {
	n.adapters = append(n.adapters, adapter)
}

func (n *notificationManager) allPersons() []config.Person {
	return config.Persons()
}

func (n *notificationManager) notify(persons []config.Person, handler func(person config.Person, adapter NotificationAdapter) bool) {
	for _, p := range persons {
		var successAdapter NotificationAdapter
		for _, a := range n.adapters {
			if handler(p, a) {
				successAdapter = a
				break
			}
		}
		if successAdapter != nil {
			log.Info().Str("name", p.Name).Str("adapter", successAdapter.Name()).Msg("Sent notification")
		}
	}
}

func (n *notificationManager) NotifyAlarm(a alarm.Type, device device.Spec, zone zone.Zone) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyAlarm(person, config.General().Name, a, device, zone)
	})
}

func (n *notificationManager) NotifyRecovery(device device.Spec, zone zone.Zone) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyRecovery(person, config.General().Name, device, zone)
	})
}

func (n *notificationManager) NotifyDeviceAvailable(device device.Spec, zone zone.Zone) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyRecovery(person, config.General().Name, device, zone)
	})
}

func (n *notificationManager) NotifyDeviceUnAvailable(device device.Spec, zone zone.Zone) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyRecovery(person, config.General().Name, device, zone)
	})
}

func (n *notificationManager) NotifyBatteryLevel(device device.Spec, zone zone.Zone, batteryLevel float32) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyBatteryLevel(person, config.General().Name, device, zone, batteryLevel)
	})
}

func (n *notificationManager) NotifyLinkQuality(device device.Spec, zone zone.Zone, quality float32) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyLinkQuality(person, config.General().Name, device, zone, quality)
	})
}

func (n *notificationManager) NotifyHumidityValue(device device.Spec, zone zone.Zone, humidity float32) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyHumidityValue(person, config.General().Name, device, zone, humidity)
	})
}

func (n *notificationManager) NotifyTemperatureValue(device device.Spec, zone zone.Zone, temperature float32) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyTemperatureValue(person, config.General().Name, device, zone, temperature)
	})
}

func (n *notificationManager) NotifyMotionSensor(device device.Spec, zone zone.Zone, motion bool) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyMotionSensor(person, config.General().Name, device, zone, motion)
	})
}

func (n *notificationManager) NotifyContactSensor(device device.Spec, zone zone.Zone, contact bool) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyContactSensor(person, config.General().Name, device, zone, contact)
	})
}

func (n *notificationManager) NotifySmokeSensor(device device.Spec, zone zone.Zone, smoke bool) {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifySmokeSensor(person, config.General().Name, device, zone, smoke)
	})
}

func (n *notificationManager) NotifyAutoArm() {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyAutoArm(person, config.General().Name)
	})
}

func (n *notificationManager) NotifyAutoDisarm() {
	n.notify(n.allPersons(), func(person config.Person, adapter NotificationAdapter) bool {
		return adapter.NotifyAutoDisarm(person, config.General().Name)
	})
}
