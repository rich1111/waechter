package system

import (
	"fmt"
	"github.com/mtrossbach/waechter/internal/config"
	"github.com/mtrossbach/waechter/internal/log"
	"github.com/mtrossbach/waechter/internal/wslice"
	"github.com/mtrossbach/waechter/system/alarm"
	"github.com/mtrossbach/waechter/system/arm"
	"github.com/mtrossbach/waechter/system/device"
	"github.com/mtrossbach/waechter/system/zone"
	"golang.org/x/exp/maps"
	"sync"
	"time"
)

type Waechter struct {
	name             string
	state            State
	zones            map[zone.Id]*zone.Zone
	devices          map[device.Id]*device.Device
	deviceConnectors []DeviceConnector
	wrongPinCount    int
	noteMgr          *notificationManager

	entryTimers          sync.Map
	unavailabilityTimers sync.Map
}

func NewWaechter() *Waechter {
	w := Waechter{
		state:                State{},
		zones:                nil,
		devices:              nil,
		wrongPinCount:        0,
		deviceConnectors:     []DeviceConnector{},
		noteMgr:              newNotificationManager(),
		entryTimers:          sync.Map{},
		unavailabilityTimers: sync.Map{},
	}

	w.loadZones()
	w.loadDevices()
	w.loadState()

	/*
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				if scanner.Err() != nil {
					log.Error().Err(scanner.Err()).Msg("Could not read from stdin")
					break
				}

				cmdParts := strings.Split(strings.TrimSpace(scanner.Text()), " ")
				cmd := strings.ToLower(cmdParts[0])
				ok := false
				switch cmd {
				case "arm":
					w.arm(systemDeviceId, arm.All)
					ok = true
				case "disarm":
					if len(cmdParts) > 1 {
						w.disarm(systemDeviceId, cmdParts[1])
						ok = true
					}
				case "entry-delay":
					if w.state.Armed() {
						w.alarm(systemDeviceId, alarm.Burglar, true)
						ok = true
					}
				}
				if !ok {
					log.Error().Str("cmd", cmd).Msg("Could not execute command.")
				}
			}

		}()
	*/
	return &w
}

func (w *Waechter) AddDeviceConnector(connector DeviceConnector) {
	w.deviceConnectors = append(w.deviceConnectors, connector)
	connector.Setup(w)
}

func (w *Waechter) RemoveDeviceConnector(id string) {
	connector, i := wslice.FilterOne[DeviceConnector](w.deviceConnectors,
		func(i DeviceConnector) bool { return i.Id() == id })
	if connector != nil {
		(*connector).Teardown()
		w.deviceConnectors = wslice.Remove[DeviceConnector](w.deviceConnectors, i)
	}
}

func (w *Waechter) AddNotificationAdapter(adapter NotificationAdapter) {
	w.noteMgr.AddAdapter(adapter)
}

func (w *Waechter) loadState() {
	s := LoadState()
	w.setAlarm(s.Alarm)
	w.setArmMode(s.ArmMode)
	w.state.ArmModeUpdated = s.ArmModeUpdated
	w.state.BdSeq = s.BdSeq
}

func (w *Waechter) loadZones() {
	w.zones = make(map[zone.Id]*zone.Zone)
	for _, zc := range config.Zones() {
		z := zone.ZoneFromConfig(zc)
		w.zones[z.Id] = &z
	}
}

func (w *Waechter) loadDevices() {
	w.devices = make(map[device.Id]*device.Device)
	/*
		// device list not from config file, by Jack Chen

		for _, dc := range config.Devices() {
			d := device.DeviceFromConfig(dc)
			w.devices[d.Id] = &d
		}

	*/
	w.devices[systemDeviceId] = systemDevice()
}

func (w *Waechter) zoneForDeviceId(id device.Id) zone.Zone {
	z, ok := w.zones[w.devices[id].Zone]
	if !ok {
		return zone.SubstitutionZone(w.name, w.state.Armed())
	}
	return *z
}

func (w *Waechter) DeviceConnectorForId(id string) DeviceConnector {
	c, _ := wslice.FilterOne[DeviceConnector](w.deviceConnectors, func(i DeviceConnector) bool { return i.Id() == id })
	return *c
}

func (w *Waechter) DeviceSensorValue(id device.Id, sensor device.Sensor) interface{} {
	value := w.devices[id].State[sensor]

	if v, ok := value.(device.MotionSensorValue); ok {
		return v.Motion
	} else if v, ok := value.(device.ContactSensorValue); ok {
		return v.Contact
	} else if v, ok := value.(device.SmokeSensorValue); ok {
		return v.Smoke
	} else if v, ok := value.(device.BatteryLevelSensorValue); ok {
		return v.BatteryLevel
	} else if v, ok := value.(device.LinkQualitySensorValue); ok {
		return v.LinkQuality
	} else if v, ok := value.(device.HumiditySensorValue); ok {
		return v.Humidity
	} else if v, ok := value.(device.TemperatureSensorValue); ok {
		return v.Temperature
	}

	return nil
}

func (w *Waechter) DeliverSensorValue(id device.Id, sensor device.Sensor, value any) bool {
	log.Debug().Str("id", string(id)).Str("sensor", string(sensor)).Msg("DeliverSensorValue")

	oldValue := w.devices[id].State[sensor]
	w.devices[id].State[sensor] = value

	if oldValue != nil && oldValue == value {
		return false
	}

	z := w.zoneForDeviceId(id)

	if v, ok := value.(device.MotionSensorValue); ok {
		fmt.Printf("Motion Sensor %v\n", v.Motion)
		w.noteMgr.NotifyMotionSensor(w.specForDeviceId(id), w.zoneForDeviceId(id), v.Motion)
		if z.Armed && v.Motion {
			if !(w.isDuringExitDelay()) {
				w.alarm(id, alarm.Burglar, z.Delayed)
			}
		}

	} else if v, ok := value.(device.ContactSensorValue); ok {
		fmt.Printf("Contact Sensor %v\n", v.Contact)
		w.noteMgr.NotifyContactSensor(w.specForDeviceId(id), w.zoneForDeviceId(id), v.Contact)
		if z.Armed && !v.Contact {
			if !(w.isDuringExitDelay()) {
				w.alarm(id, alarm.Burglar, z.Delayed)
			}
		}

	} else if v, ok := value.(device.SmokeSensorValue); ok {
		fmt.Printf("Smoke Sensor %v\n", v.Smoke)
		w.noteMgr.NotifySmokeSensor(w.specForDeviceId(id), w.zoneForDeviceId(id), v.Smoke)
		if v.Smoke {
			w.alarm(id, alarm.Fire, false)
		}

	} else if v, ok := value.(device.PanicSensorValue); ok {
		fmt.Printf("Panic Sensor %v\n", v.Panic)
		if v.Panic {
			w.alarm(id, alarm.Panic, false)
		}

	} else if v, ok := value.(device.BatteryWarningSensorValue); ok {
		fmt.Printf("Battery Low Warning %v\n", v.BatteryWarning)
		if v.BatteryWarning {
			w.noteMgr.NotifyBatteryLevel(w.specForDeviceId(id), w.zoneForDeviceId(id), 0)
		}

	} else if v, ok := value.(device.TamperSensorValues); ok {
		fmt.Printf("Tamper Sensor %v\n", v.Tamper)
		if v.Tamper {
			if (z.Armed && config.General().TamperAlarmWhileArmed) || (!z.Armed && config.General().TamperAlarmWhileDisarmed) {
				w.alarm(id, alarm.Tamper, false)
			}
		}

	} else if v, ok := value.(device.BatteryLevelSensorValue); ok {
		fmt.Printf("Battery Value %f\n", v.BatteryLevel)
		//if v.BatteryLevel/100 < config.General().BatteryThreshold {
		w.noteMgr.NotifyBatteryLevel(w.specForDeviceId(id), w.zoneForDeviceId(id), v.BatteryLevel)
		//}

	} else if v, ok := value.(device.LinkQualitySensorValue); ok {
		fmt.Printf("Link Quality Value %f\n", v.LinkQuality)
		//if v.LinkQuality/255 < config.General().LinkQualityThreshold {
		w.noteMgr.NotifyLinkQuality(w.specForDeviceId(id), w.zoneForDeviceId(id), v.LinkQuality)
		//}

	} else if v, ok := value.(device.HumiditySensorValue); ok {
		fmt.Printf("Humidity Value %f\n", v.Humidity)
		w.noteMgr.NotifyHumidityValue(w.specForDeviceId(id), w.zoneForDeviceId(id), v.Humidity)

	} else if v, ok := value.(device.TemperatureSensorValue); ok {
		fmt.Printf("Temperature Value %f\n", v.Temperature)
		w.noteMgr.NotifyTemperatureValue(w.specForDeviceId(id), w.zoneForDeviceId(id), v.Temperature)

	} else if v, ok := value.(device.ArmingSensorValue); ok {
		if v.ArmMode == arm.Disarmed {
			return false
		}
		return w.arm(id, v.ArmMode)

	} else if v, ok := value.(device.DisarmingSensorValue); ok {
		return w.disarm(id, v.Pin)

	} else {
		log.Error().Str("device", string(id)).Interface("value", value).Msg("Unknown sensor value received")
		return false
	}

	return true
}

func (w *Waechter) isDuringExitDelay() bool {
	exitDelay := time.Duration(config.General().ExitDelay) * time.Second
	return w.state.Armed() && time.Now().Sub(w.state.ArmModeUpdated) < exitDelay
}

func (w *Waechter) alarm(id device.Id, alarmType alarm.Type, delayedZone bool) {
	if alarmType == alarm.Burglar && delayedZone && (w.state.Alarm == alarm.None || w.state.Alarm == alarm.EntryDelay) {
		w._alarm(id, alarm.EntryDelay)
		t, ok := w.entryTimers.Load(id)
		if !ok {
			t = time.AfterFunc(time.Duration(config.General().EntryDelay)*time.Second, func() {
				w.entryTimers.Delete(id)
				if w.zoneForDeviceId(id).Armed {
					w._alarm(id, alarmType)
				}
			})
			w.entryTimers.Store(id, t)
		}

	} else {
		w._alarm(id, alarmType)
	}
}

func (w *Waechter) specForDeviceId(id device.Id) device.Spec {
	d, ok := w.devices[id]
	if !ok {
		d = systemDevice()
	}
	return (*d).Spec
}

func (w *Waechter) _alarm(id device.Id, a alarm.Type) {
	w.setAlarm(a)
	if a != alarm.EntryDelay {
		w.noteMgr.NotifyAlarm(a, w.specForDeviceId(id), w.zoneForDeviceId(id))
	}
}

func (w *Waechter) arm(id device.Id, mode arm.Mode) bool {
	if w.state.Armed() || mode == arm.Disarmed {
		return false
	}
	if mode == arm.Disarmed {
		mode = arm.All
	}
	w.wrongPinCount = 0
	w.setArmMode(mode)

	for _, d := range w.DevicesWithTamper() {
		log.Warn().Str("_id", string(d.Id)).Msg("! Device is tampered!")
	}

	for _, d := range w.OpenContactSensors() {
		log.Warn().Str("_id", string(d.Id)).Msg("! Door/Window is still open!")
	}
	return true
}

func (w *Waechter) DevicesWithTamper() []*device.Device {
	result := make(map[device.Id]*device.Device)

	w.iterateDeviceStates(func(d *device.Device, sensor device.Sensor, value any) {
		if v, ok := value.(device.TamperSensorValues); ok {
			if v.Tamper {
				result[d.Id] = d
			}
		}
	})

	return maps.Values(result)
}

func (w *Waechter) OpenContactSensors() []*device.Device {
	result := make(map[device.Id]*device.Device)

	w.iterateDeviceStates(func(d *device.Device, sensor device.Sensor, value any) {
		if v, ok := value.(device.ContactSensorValue); ok {
			if !v.Contact {
				result[d.Id] = d
			}
		}
	})

	return maps.Values(result)
}

func (w *Waechter) iterateDeviceStates(handler func(d *device.Device, sensor device.Sensor, value any)) {
	for _, d := range w.devices {
		for sensor, value := range d.State {
			handler(d, sensor, value)
		}
	}
}

func (w *Waechter) disarm(id device.Id, enteredPin string) bool {
	persons := config.Persons()
	person, _ := wslice.FilterOne(persons, func(p config.Person) bool { return p.Pin == enteredPin })

	if person != nil {
		w.wrongPinCount = 0
		if w.state.Alarm != alarm.None && w.state.Alarm != alarm.EntryDelay {
			w.noteMgr.NotifyRecovery(w.specForDeviceId(id), w.zoneForDeviceId(id))
		}
		log.Info().Str("name", person.Name).Msg("Disarmed by pin")
		w.setAlarm(alarm.None)
		w.setArmMode(arm.Disarmed)
		w.entryTimers.Range(func(key, value any) bool {
			t := value.(*time.Timer)
			t.Stop()
			w.entryTimers.Delete(key)
			return true
		})
		return true
	} else {
		w.wrongPinCount += 1
		log.Info().Str("device", string(id)).Int("wrongPinCount", w.wrongPinCount).Msg("Wrong PIN entered.")
		if w.wrongPinCount > config.General().MaxWrongPinCount {
			log.Info().Str("device", string(id)).Int("wrongPinCount", w.wrongPinCount).Msg("Maximum number of wrong PINs exceed.")
			w.alarm(id, alarm.TamperPin, false)
		}
		return false
	}
}

func (w *Waechter) SystemState() State {
	return w.state
}

func (w *Waechter) DeviceListUpdated(connector DeviceConnector) {
	if connector == nil {
		return
	}
	deviceSpecs := connector.EnumerateDevices()
	log.Info().Str("connector", connector.DisplayName()).Str("id", connector.Id()).Msg("Received new device list:")
	for _, s := range deviceSpecs {
		if ad, ok := w.devices[s.Id]; ok {
			ad.Spec = s
		} else {
			// not existing device, new it, by Jack Chen
			d := device.NewDevice(s.Id)
			d.Spec = s
			w.devices[d.Id] = &d
		}
		var sensors []string
		var actors []string
		for _, ss := range s.Sensors {
			sensors = append(sensors, string(ss))
		}
		for _, sa := range s.Actors {
			actors = append(actors, string(sa))
		}
		log.Info().Str("id", string(s.Id)).Str("displayName", s.DisplayName).Str("vendor", s.Vendor).Str("model", s.Model).Strs("sensors", sensors).Strs("actors", actors).Msg("\t- Device detected")
	}

	log.Info().Str("connector", connector.DisplayName()).Msg("Trying to activate devices")
	for _, d := range w.devices {
		if /*!d.Active &&*/ d.Id.Prefix() == connector.Id() {
			err := connector.ActivateDevice(d.Id)
			if err != nil {
				device.DError(d).Err(err).Msg("✗ Could not activate device")
			} else {
				device.DInfo(d).Msg("✓ Device active")
			}
		}
	}
	log.Info().Str("connector", connector.DisplayName()).Msg("Done with activating devices")
}

func (w *Waechter) OperationalStateChanged(connector DeviceConnector) {
	if !connector.Operational() && config.General().DeviceSystemFaultAlarm && w.state.Armed() {
		time.AfterFunc(time.Duration(config.General().DeviceSystemFaultAlarmDelay)*time.Second, func() {
			if !connector.Operational() && w.state.Armed() {
				w.alarm(systemDeviceId, alarm.Tamper, false)
			}
		})
	}
}

func (w *Waechter) DeviceUnavailable(id device.Id) {
	d, ok := w.devices[id]
	if ok {
		d.Active = false
		w.noteMgr.NotifyDeviceUnAvailable(w.specForDeviceId(id), w.zoneForDeviceId(id))
	}

	z := w.zoneForDeviceId(id)
	if z.Armed {
		w.alarm(id, alarm.Tamper, false)
	}

	// remove this device when it is deactivated and not available, by Jack Chen
	delete(w.devices, id)
}

func (w *Waechter) DeviceAvailable(id device.Id) {
	d, ok := w.devices[id]
	if ok {
		d.Active = true
		w.noteMgr.NotifyDeviceAvailable(w.specForDeviceId(id), w.zoneForDeviceId(id))
		w.updateActor(id, device.StateActor, w.state.stateActorPayload())
		w.updateActor(id, device.AlarmActor, w.state.alarmActorPayload())
	}
}

func (w *Waechter) setArmMode(mode arm.Mode) {
	if w.state.ArmMode != mode {
		w.state.ArmMode = mode
		w.state.ArmModeUpdated = time.Now()

		w.syncZones()

		w.updateActors(device.StateActor, w.state.stateActorPayload())

		l := log.Info().Str("mode", string(mode))
		if w.state.Armed() {
			l = l.Int("exitDelay", config.General().ExitDelay)
		}
		l.Msg("➔ System mode changed")
		if w.state.Armed() {
			go func() {
				w.notificationBeep(false)
				for true {
					if w.state.Armed() && w.isDuringExitDelay() {
						r := config.General().ExitDelay - int(time.Now().Sub(w.state.ArmModeUpdated).Seconds())
						if r > 0 {
							log.Info().Int("remaining", r).Msg("Exit delay.")
						}
						time.Sleep(5 * time.Second)
					} else {
						if w.state.Armed() {
							log.Info().Msg("Exit delay ended.")
							w.notificationBeep(true)
						}
						return
					}
				}
			}()
		}
		PersistState(w.state)
	}
}

func (w *Waechter) setAlarm(a alarm.Type) {
	if w.state.Alarm != a {
		w.state.Alarm = a

		w.updateActors(device.StateActor, w.state.stateActorPayload())
		w.updateActors(device.AlarmActor, w.state.alarmActorPayload())

		l := log.Info().Str("alarm", string(a))
		if a == alarm.EntryDelay {
			l = l.Int("entryDelay", config.General().EntryDelay)
		}
		l.Msg("➔ Alarm changed")
		PersistState(w.state)
	}
}

func (w *Waechter) notificationBeep(long bool) {
	a := device.NotificationShortActor
	if long {
		a = device.NotificationLongActor
	}
	for _, d := range w.devices {
		if wslice.Contains(d.Spec.Actors, a) {
			if c := w.DeviceConnectorForId(d.Id.Prefix()); c != nil {
				c.ControlActor(d.Id, a, nil)
			}
		}
	}
}

func (w *Waechter) updateActor(id device.Id, actor device.Actor, payload any) {
	if d, ok := w.devices[id]; ok && d != nil && wslice.Contains(d.Spec.Actors, actor) {
		if c := w.DeviceConnectorForId(d.Id.Prefix()); c != nil {
			c.ControlActor(d.Id, actor, payload)
		}
	}
}

func (w *Waechter) updateActors(actor device.Actor, payload any) {
	for i := range w.devices {
		w.updateActor(i, actor, payload)
	}
}

func (w *Waechter) syncZones() {
	for _, z := range w.zones {
		if z.Perimeter {
			z.Armed = w.state.Armed()
		} else {
			if w.state.Armed() && w.state.ArmMode != arm.Perimeter {
				z.Armed = true
			} else {
				z.Armed = false
			}
		}
	}
}
