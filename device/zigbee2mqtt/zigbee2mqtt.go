package zigbee2mqtt

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mtrossbach/waechter/device"
	"github.com/mtrossbach/waechter/device/zigbee2mqtt/connector"
	"github.com/mtrossbach/waechter/device/zigbee2mqtt/driver"
	"github.com/mtrossbach/waechter/internal/cfg"
	"github.com/mtrossbach/waechter/internal/log"
	"github.com/mtrossbach/waechter/internal/wslice"
	"github.com/mtrossbach/waechter/system"
	"sync"
	"time"
)

const namespace string = "zm"

type zigbee2mqtt struct {
	systemController device.SystemController
	connector        *connector.Connector
	devices          sync.Map
}

func New() *zigbee2mqtt {
	return &zigbee2mqtt{
		connector: connector.New(),
	}
}

func (zm *zigbee2mqtt) Start(systemController device.SystemController) {
	zm.systemController = systemController
	systemController.SubscribeStateUpdate(zm, zm.updateState)
	log.Info().Str("uri", cfg.GetString(cConnection)).Msg("Connecting to Zigbee2Mqtt broker...")
	err := zm.connector.Connect(cOptions(), func() {
		log.Info().Msg("Connected to Zigbee2Mqtt broker")
	}, func(err error) {
		if err != nil {
			log.Error().Err(err).Msg("Connection to Zigbee2Mqtt broker lost. Retrying in a few seconds...")
			zm.reconnect(systemController)
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("Could not connect to Zigbee2Mqtt broker. Retrying in a few seconds...")
		zm.reconnect(systemController)
	} else {
		zm.connector.Subscribe("bridge/devices", zm.handleNewDeviceList)
		zm.connector.Subscribe("bridge/event", zm.handleDeviceEvent)
	}
}

func (zm *zigbee2mqtt) reconnect(systemController device.SystemController) {
	zm.Stop()
	<-time.After(10 * time.Second)
	zm.Start(systemController)
}

func (zm *zigbee2mqtt) updateState(state system.State, armingMode system.ArmingMode, alarmType system.AlarmType) {
	zm.devices.Range(func(_, d any) bool {
		dev := d.(system.Device)
		zm.updateStateForDevice(&dev)
		return true
	})
}

func (zm *zigbee2mqtt) updateStateForDevice(dev *system.Device) {
	if dev != nil {
		switch dev.Type {
		case system.Siren:
			driver.SirenStateUpdater(zm.systemController, zm.sender(dev))
		case system.Keypad:
			driver.KeypadStateUpdater(zm.systemController, zm.sender(dev))
		case system.SmokeSensor:
			driver.SmokeStateUpdater(zm.systemController, zm.sender(dev))
		}
	}
}

func (zm *zigbee2mqtt) sender(dev *system.Device) driver.Sender {
	return func(payload any) {
		if dev != nil {
			zm.connector.Publish(fmt.Sprintf("%v/set", dev.Name), payload)
		}
	}
}

func (zm *zigbee2mqtt) tearDownAllDevices(connectionLost bool) {
	zm.devices.Range(func(id, d any) bool {
		dev := d.(system.Device)
		if !connectionLost {
			zm.connector.Unsubscribe(dev.Name)
		}
		system.DInfo(dev).Msg("Remove Zigbee device")
		return true
	})
	zm.devices = sync.Map{}
}

func (zm *zigbee2mqtt) setupDevice(dev system.Device) {
	zm.devices.Store(dev.Id, dev)

	switch dev.Type {
	case system.MotionSensor:
		zm.connector.Subscribe(dev.Name, driver.MotionSensorHandler(&dev, zm.systemController))
	case system.ContactSensor:
		zm.connector.Subscribe(dev.Name, driver.ContactSensorHandler(&dev, zm.systemController))
	case system.SmokeSensor:
		zm.connector.Subscribe(dev.Name, driver.SmokeSensorHandler(&dev, zm.systemController))
	case system.Keypad:
		zm.connector.Subscribe(dev.Name, driver.KeypadHandler(&dev, zm.systemController, zm.sender(&dev)))
	case system.Siren:
		zm.connector.Subscribe(dev.Name, driver.SirenHandler(&dev, zm.systemController))
	}
	zm.updateStateForDevice(&dev)
	system.DInfo(dev).Msg("Setup Zigbee device")
}

func (zm *zigbee2mqtt) Stop() {
	zm.tearDownAllDevices(true)
	zm.connector.Disconnect()
	zm.systemController = nil
}

func (zm *zigbee2mqtt) handleDeviceEvent(msg mqtt.Message) {
	var deviceEvent DeviceEvent
	if err := json.Unmarshal(msg.Payload(), &deviceEvent); err != nil {
		log.Error().Str("payload", string(msg.Payload())).Msg("Could not parse Zigbee device event!")
		return
	}

	if deviceEvent.Type == "device_announce" && len(deviceEvent.Data.IeeeAddress) > 0 {
		d, ok := zm.devices.Load(deviceEvent.Data.IeeeAddress)
		if ok {
			dev := d.(system.Device)
			zm.updateStateForDevice(&dev)
		}
	}
}

func (zm *zigbee2mqtt) handleNewDeviceList(msg mqtt.Message) {
	var newDevices []Z2MDeviceInfo
	if err := json.Unmarshal(msg.Payload(), &newDevices); err != nil {
		log.Error().Str("payload", string(msg.Payload())).Msg("Could not parse Zigbee device payload!")
		return
	}

	relevantDevices := make(map[string]Z2MDeviceInfo)
	for _, device := range newDevices {
		if device.Type == "EndDevice" && device.Supported {
			relevantDevices[device.IeeeAddress] = device
		}
	}

	zm.tearDownAllDevices(false)

	for _, d := range relevantDevices {
		dev := zm.deviceFromMessage(d)
		if dev != nil {
			zm.setupDevice(*dev)
		}
	}
}

func (zm *zigbee2mqtt) deviceFromMessage(message Z2MDeviceInfo) *system.Device {
	dev := system.Device{
		Namespace: namespace,
		Id:        message.IeeeAddress,
		Name:      message.FriendlyName,
	}

	var exposes []string
	for _, e := range message.Definition.Exposes {
		exposes = append(exposes, e.Property)
	}

	if wslice.ContainsAll(exposes, []string{"action_code", "action"}) {
		dev.Type = system.Keypad
	} else if wslice.ContainsAll(exposes, []string{"warning"}) {
		dev.Type = system.Siren
	} else if wslice.ContainsAll(exposes, []string{"contact"}) {
		dev.Type = system.ContactSensor
	} else if wslice.ContainsAll(exposes, []string{"smoke"}) {
		dev.Type = system.SmokeSensor
	} else if wslice.ContainsAll(exposes, []string{"occupancy"}) {
		dev.Type = system.MotionSensor
	} else {
		return nil
	}

	return &dev
}