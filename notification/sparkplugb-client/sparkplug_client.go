/*
Sparkplug 3.0.0
Note: Complies to v3.0.0 of the Sparkplug specification

	to the extent needed for Winsonic DataIO and other industrial 4.0 products.

Copyright (c) 2023 Winsonic Electronics, Taiwan
@author David Lee

* This program and the accompanying materials are made available under the
* terms of the Eclipse Public License 2.0 which is available at
* http://www.eclipse.org/legal/epl-2.0.
*/
package sparkplugb_client

import (
	"bytes"
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mtrossbach/waechter/internal/config"
	"github.com/mtrossbach/waechter/notification/sparkplugb-client/sparkplug"
	"github.com/mtrossbach/waechter/system"
	"github.com/mtrossbach/waechter/system/alarm"
	"github.com/mtrossbach/waechter/system/device"
	"github.com/mtrossbach/waechter/system/zone"
	"net"
	"strconv"
)

type Sparkplug struct {
	node      sparkplug.ClientNode
	connected bool
}

func (s *Sparkplug) Name() string {
	return "Sparkplug"
}

func (s *Sparkplug) NotifyAlarm(person config.Person, systemName string, a alarm.Type, device device.Spec, zone zone.Zone) bool {
	return true
}

func (s *Sparkplug) NotifyRecovery(person config.Person, systemName string, device device.Spec, zone zone.Zone) bool {
	return true
}

func (s *Sparkplug) NotifyDeviceAvailable(person config.Person, systemName string, dev device.Spec, zone zone.Zone) bool {
	// Publish Device Birth
	// Note: First Node Birth must be published
	ms := getDeviceBirthMetrics(dev)
	deviceID := dev.IeeeAddress // Get this ID from Zigbee device
	err := s.node.PublishDeviceBirth(deviceID, ms)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (s *Sparkplug) NotifyDeviceUnAvailable(person config.Person, systemName string, dev device.Spec, zone zone.Zone) bool {
	// Publish Device Death
	// If Device cannot be contacted
	deviceID := dev.IeeeAddress // Get this ID from Zigbee device
	err := s.node.PublishDeviceDeath(deviceID)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (s *Sparkplug) NotifyBatteryLevel(person config.Person, systemName string, dev device.Spec, zone zone.Zone, batteryLevel float32) bool {
	// Publish Device Data
	// When there is change in dev metrics
	deviceID := dev.IeeeAddress // Get this ID from Zigbee dev
	ms := getDeviceDataMetrics_2(device.BatteryLevelSensor, device.BatteryLevelSensorValue{BatteryLevel: batteryLevel})
	err := s.node.PublishDeviceData(deviceID, ms)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (s *Sparkplug) NotifyLinkQuality(person config.Person, systemName string, dev device.Spec, zone zone.Zone, quality float32) bool {
	// Publish Device Data
	// When there is change in dev metrics
	deviceID := dev.IeeeAddress // Get this ID from Zigbee dev
	ms := getDeviceDataMetrics_2(device.LinkQualitySensor, device.LinkQualitySensorValue{LinkQuality: quality})
	err := s.node.PublishDeviceData(deviceID, ms)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (s *Sparkplug) NotifyHumidityValue(person config.Person, systemName string, dev device.Spec, zone zone.Zone, humidity float32) bool {
	// Publish Device Data
	// When there is change in dev metrics
	deviceID := dev.IeeeAddress // Get this ID from Zigbee dev
	ms := getDeviceDataMetrics_2(device.Humidity, device.HumiditySensorValue{Humidity: humidity})
	err := s.node.PublishDeviceData(deviceID, ms)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (s *Sparkplug) NotifyTemperatureValue(person config.Person, systemName string, dev device.Spec, zone zone.Zone, temperature float32) bool {
	// Publish Device Data
	// When there is change in dev metrics
	deviceID := dev.IeeeAddress // Get this ID from Zigbee dev
	ms := getDeviceDataMetrics_2(device.Temperature, device.TemperatureSensorValue{Temperature: temperature})
	err := s.node.PublishDeviceData(deviceID, ms)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (s *Sparkplug) NotifyMotionSensor(person config.Person, systemName string, dev device.Spec, zone zone.Zone, motion bool) bool {
	// Publish Device Data
	// When there is change in dev metrics
	deviceID := dev.IeeeAddress // Get this ID from Zigbee dev
	ms := getDeviceDataMetrics_2(device.MotionSensor, device.MotionSensorValue{Motion: motion})
	err := s.node.PublishDeviceData(deviceID, ms)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (s *Sparkplug) NotifyContactSensor(person config.Person, systemName string, dev device.Spec, zone zone.Zone, contact bool) bool {
	// Publish Device Data
	// When there is change in dev metrics
	deviceID := dev.IeeeAddress // Get this ID from Zigbee dev
	ms := getDeviceDataMetrics_2(device.ContactSensor, device.ContactSensorValue{Contact: contact})
	err := s.node.PublishDeviceData(deviceID, ms)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (s *Sparkplug) NotifySmokeSensor(person config.Person, systemName string, dev device.Spec, zone zone.Zone, smoke bool) bool {
	// Publish Device Data
	// When there is change in dev metrics
	deviceID := dev.IeeeAddress // Get this ID from Zigbee dev
	ms := getDeviceDataMetrics_2(device.SmokeSensor, device.SmokeSensorValue{Smoke: smoke})
	err := s.node.PublishDeviceData(deviceID, ms)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (s *Sparkplug) NotifyAutoArm(person config.Person, systemName string) bool {
	return true
}

func (s *Sparkplug) NotifyAutoDisarm(person config.Person, systemName string) bool {
	return true
}

//ServerIP: 192.168.11.61
//Username: 3a061058-30f5-11ee-94c7-0242ac120006
//Password: J21Pg3w7R4nAdCKELHs8VMFjvS5DY069
//GroupID: 3a061058-30f5-11ee-94c7-0242ac120006
//NodeID: 70:4a:0e:d4:5f:da

var sysController system.Controller
var sp Sparkplug

func NewSparkplug(w system.Controller) *Sparkplug {
	sysController = w

	addrMac, _, err := getNetInterfaceMacIPAddr("wlan0")

	sp = Sparkplug{
		node: sparkplug.ClientNode{
			Config: sparkplug.Config{
				ServerUrl: "192.168.11.61",
				Username:  "3a061058-30f5-11ee-94c7-0242ac120006",
				Password:  "J21Pg3w7R4nAdCKELHs8VMFjvS5DY069",
				GroupID:   "3a061058-30f5-11ee-94c7-0242ac120006",
				NodeID:    addrMac,
				ClientID:  addrMac, // Gateway MAC
			},
			MessagePubHandler:   &messagePubHandler,
			ConnectHandler:      &connectHandler,
			ConnectLostHandler:  &connectLostHandler,
			ReconnectingHandler: &reconnectingHandler,
		},
	}

	err = sp.node.Connect(getBDSeq())
	if err != nil {
		fmt.Println(err)
		sp.connected = false
	} else {
		sp.connected = true
	}

	// Publish Node Birth
	sendingNodeBirth()

	return &sp
}

func sendingNodeBirth() {
	// Publish Node Birth
	ms := getNodeBirthMetrics()
	err := sp.node.PublishNodeBirth(ms)
	if err != nil {
		fmt.Println(err)
	}
}

func reconnectZigbee2Mqtt() {
	sysController.DeviceConnectorForId("z2m").DisconnectForReconnect()
}

func getBDSeq() int {
	s := sysController.SystemState()
	bdSeq := s.BdSeq
	//fmt.Printf("got bdSeq %d\n", bdSeq)
	if bdSeq == 255 {
		s.BdSeq = 0 // recursive to 0
	} else {
		s.BdSeq = bdSeq + 1 // increase 1
	}
	system.PersistState(s) // write state to file
	return bdSeq
}

// ******************************************************************************
// ******************************* Node Handlers ********************************
// ******************************************************************************
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Received message, topic=", msg.Topic())
	p := sparkplug.Payload{}
	err := p.DecodePayload(msg.Payload())
	if err != nil {
		fmt.Println(err)
	}
	ms := p.Metrics
	for i := range ms {
		fmt.Println("Metric: Name=", ms[i].Name, ", DataType=", ms[i].DataType.String(), ", Value=", ms[i].Value)
		if ms[i].Name == "Node Control/Rebirth" && ms[i].DataType == sparkplug.TypeBool && ms[i].Value == "true" {
			// here want to send Node Rebirth
			sendingNodeBirth()
			// reconnect Zigbee2Mqtt for sending devices list again
			reconnectZigbee2Mqtt()
		}
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

var reconnectingHandler mqtt.ReconnectHandler = func(client mqtt.Client, options *mqtt.ClientOptions) {
	fmt.Printf("Reconnect handler")
	// Note: Need to increment the bdSeq here for reconnecting
	wp, err := sparkplug.GetWillPayload(getBDSeq())
	if err != nil {
		fmt.Println("Error encoding will payload: ", err)
	}
	options.WillPayload = wp

	// reconnect Zigbee2Mqtt for sending devices list again
	reconnectZigbee2Mqtt()
}

// ******************************************************************************
// ******************************************************************************

func getNodeBirthMetrics() []sparkplug.Metric {
	m1 := sparkplug.Metric{
		Name:     "Node Control/Rebirth",
		DataType: sparkplug.TypeBool,
		Value:    "false",
	}
	m2 := sparkplug.Metric{
		Name:     "Model Type",
		DataType: sparkplug.TypeString,
		Value:    "ZIGBEE",
	}
	m3 := sparkplug.Metric{
		Name:     "Model Name",
		DataType: sparkplug.TypeString,
		Value:    "ZIGBEE2MQTT",
	}
	m4 := sparkplug.Metric{
		Name:     "Firmware Version",
		DataType: sparkplug.TypeString,
		Value:    "1.0.0",
	}
	ms := []sparkplug.Metric{}
	ms = append(ms, m1)
	ms = append(ms, m2)
	ms = append(ms, m3)
	ms = append(ms, m4)

	return ms
}

func getDeviceBirthMetrics(dev_spec device.Spec) []sparkplug.Metric {
	m1 := sparkplug.Metric{
		Name:     "Device Name",
		DataType: sparkplug.TypeString,
		Value:    dev_spec.DisplayName,
	}
	m2 := sparkplug.Metric{
		Name:     "Model Type",
		DataType: sparkplug.TypeString,
		Value:    dev_spec.Description,
	}
	m3 := sparkplug.Metric{
		Name:     "Model Name",
		DataType: sparkplug.TypeString,
		Value:    dev_spec.Model,
	}
	m4 := sparkplug.Metric{
		Name:     "Vendor",
		DataType: sparkplug.TypeString,
		Value:    dev_spec.Vendor,
	}
	m5 := sparkplug.Metric{
		Name:     "Firmware Version",
		DataType: sparkplug.TypeString,
		Value:    "1.0.0",
	}

	ms := []sparkplug.Metric{}
	ms = append(ms, m1)
	ms = append(ms, m2)
	ms = append(ms, m3)
	ms = append(ms, m4)
	ms = append(ms, m5)

	for _, s := range dev_spec.Sensors {
		switch s {
		case device.Humidity:
			m := sparkplug.Metric{
				Name:     "metric/humidity",
				DataType: sparkplug.TypeFloat,
				Value:    "0",
			}
			ms = append(ms, m)
		case device.Temperature:
			m := sparkplug.Metric{
				Name:     "metric/temperature",
				DataType: sparkplug.TypeFloat,
				Value:    "0",
			}
			ms = append(ms, m)
		case device.MotionSensor:
			m := sparkplug.Metric{
				Name:     "metric/motion",
				DataType: sparkplug.TypeBool,
				Value:    "false",
			}
			ms = append(ms, m)
		case device.ContactSensor:
			m := sparkplug.Metric{
				Name:     "metric/contact",
				DataType: sparkplug.TypeBool,
				Value:    "true",
			}
			ms = append(ms, m)
		case device.SmokeSensor:
			m := sparkplug.Metric{
				Name:     "metric/smoke",
				DataType: sparkplug.TypeBool,
				Value:    "false",
			}
			ms = append(ms, m)
		case device.BatteryLevelSensor:
			m := sparkplug.Metric{
				Name:     "Battery Level",
				DataType: sparkplug.TypeFloat,
				Value:    "100",
			}
			ms = append(ms, m)
		case device.LinkQualitySensor:
			m := sparkplug.Metric{
				Name:     "Link Quality",
				DataType: sparkplug.TypeFloat,
				Value:    "255",
			}
			ms = append(ms, m)
		}
	}

	return ms
}

func getDeviceDataMetrics_2(sensor device.Sensor, value any) []sparkplug.Metric {
	ms := []sparkplug.Metric{}
	switch sensor {
	case device.Humidity:
		if v, ok := value.(device.HumiditySensorValue); ok {
			m := sparkplug.Metric{
				Name:     "metric/humidity",
				DataType: sparkplug.TypeFloat,
				Value:    fmt.Sprintf("%f", v.Humidity),
			}
			ms = append(ms, m)
		}
	case device.Temperature:
		if v, ok := value.(device.TemperatureSensorValue); ok {
			m := sparkplug.Metric{
				Name:     "metric/temperature",
				DataType: sparkplug.TypeFloat,
				Value:    fmt.Sprintf("%f", v.Temperature),
			}
			ms = append(ms, m)
		}
	case device.MotionSensor:
		if v, ok := value.(device.MotionSensorValue); ok {
			m := sparkplug.Metric{
				Name:     "metric/motion",
				DataType: sparkplug.TypeBool,
				Value:    strconv.FormatBool(v.Motion),
			}
			ms = append(ms, m)
		}
	case device.ContactSensor:
		if v, ok := value.(device.ContactSensorValue); ok {
			m := sparkplug.Metric{
				Name:     "metric/contact",
				DataType: sparkplug.TypeBool,
				Value:    strconv.FormatBool(v.Contact),
			}
			ms = append(ms, m)
		}
	case device.SmokeSensor:
		if v, ok := value.(device.SmokeSensorValue); ok {
			m := sparkplug.Metric{
				Name:     "metric/smoke",
				DataType: sparkplug.TypeBool,
				Value:    strconv.FormatBool(v.Smoke),
			}
			ms = append(ms, m)
		}
	case device.BatteryLevelSensor:
		if v, ok := value.(device.BatteryLevelSensorValue); ok {
			m := sparkplug.Metric{
				Name:     "Battery Level",
				DataType: sparkplug.TypeFloat,
				Value:    fmt.Sprintf("%f", v.BatteryLevel),
			}
			ms = append(ms, m)
		}
	case device.LinkQualitySensor:
		if v, ok := value.(device.LinkQualitySensorValue); ok {
			m := sparkplug.Metric{
				Name:     "Link Quality",
				DataType: sparkplug.TypeFloat,
				Value:    fmt.Sprintf("%f", v.LinkQuality),
			}
			ms = append(ms, m)
		}
	}

	return ms
}

// getNetInterfaceMacIPAddr gets network interface MAC hardware
// address, IP address of the host machine
func getNetInterfaceMacIPAddr(interfaceName string) (addrMAC string, addrIP string, err error) {
	var (
		interfaces []net.Interface
		addrs      []net.Addr
		ipv4Addr   net.IP
		macAddr    net.HardwareAddr
	)

	interfaces, err = net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			if i.Name == interfaceName {
				if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
					// Don't use random as we have a real address
					macAddr = i.HardwareAddr
					addrMAC = macAddr.String()
					if addrs, err = i.Addrs(); err != nil { // get addresses
						return addrMAC, "0.0.0.0", err
					}
					for _, addr := range addrs { // get ipv4 address
						if ipv4Addr = addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
							break
						}
					}
					if ipv4Addr == nil {
						err = errors.New(fmt.Sprintf("interface %s don't have an ipv4 address\n", interfaceName))
						return addrMAC, "0.0.0.0", err
					}
					addrIP = ipv4Addr.String()
					//fmt.Println("Resolved Host MAC: " + addrMAC + " ,IP address " + addrIP)
					break
				} else {
					err = errors.New(fmt.Sprintf("interface %s not used\n", interfaceName))
					return "0:0:0:0:0:0", "0.0.0.0", err
				}
			}
		}
		if ipv4Addr == nil || macAddr == nil {
			err = errors.New(fmt.Sprintf("interface %s not found\n", interfaceName))
			return "0:0:0:0:0:0", "0.0.0.0", err
		}
	}
	return
}
