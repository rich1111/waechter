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
	"github.com/mtrossbach/waechter/internal/config"
	"github.com/mtrossbach/waechter/system"
	"github.com/mtrossbach/waechter/system/alarm"
	"github.com/mtrossbach/waechter/system/device"
	"github.com/mtrossbach/waechter/system/zone"
	"net"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mtrossbach/waechter/notification/sparkplugb-client/sparkplug"
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

func (s *Sparkplug) NotifyDeviceAvailable(person config.Person, systemName string, device device.Spec, zone zone.Zone) bool {
	return true
}

func (s *Sparkplug) NotifyDeviceUnAvailable(person config.Person, systemName string, device device.Spec, zone zone.Zone) bool {
	return true
}

func (s *Sparkplug) NotifyBatteryLevel(person config.Person, systemName string, device device.Spec, zone zone.Zone, batteryLevel float32) bool {
	//TODO implement me
	fmt.Println("not implemented")
	return true
}

func (s *Sparkplug) NotifyLinkQuality(person config.Person, systemName string, device device.Spec, zone zone.Zone, quality float32) bool {
	//TODO implement me
	fmt.Println("not implemented")
	return true
}

func (w *Sparkplug) NotifyHumidityValue(person config.Person, systemName string, device device.Spec, zone zone.Zone, humidity float32) bool {

	return true
}

func (w *Sparkplug) NotifyTemperatureValue(person config.Person, systemName string, device device.Spec, zone zone.Zone, temperature float32) bool {

	return true
}

func (w *Sparkplug) NotifyMotionSensor(person config.Person, systemName string, device device.Spec, zone zone.Zone, motion bool) bool {

	return true
}

func (w *Sparkplug) NotifyContactSensor(person config.Person, systemName string, device device.Spec, zone zone.Zone, contact bool) bool {

	return true
}

func (w *Sparkplug) NotifySmokeSensor(person config.Person, systemName string, device device.Spec, zone zone.Zone, smoke bool) bool {

	return true
}

func (s *Sparkplug) NotifyAutoArm(person config.Person, systemName string) bool {
	return true
}

func (s *Sparkplug) NotifyAutoDisarm(person config.Person, systemName string) bool {
	return true
}

//ServerIP: 192.168.11.61
//Username: 5d9a6bce-14b5-11ee-ab7c-0242ac120006
//Password: 861peUJXKI49xlb0FqECws7Q3a52RGgo
//GroupID: 5d9a6bce-14b5-11ee-ab7c-0242ac120006
//NodeID: 70:4a:0e:d4:5f:da

func NewSparkplug() *Sparkplug {
	addrMac, _, err := getNetInterfaceMacIPAddr("wlan0")

	sp := Sparkplug{
		node: sparkplug.ClientNode{
			Config: sparkplug.Config{
				ServerUrl: "192.168.11.61",
				Username:  "5d9a6bce-14b5-11ee-ab7c-0242ac120006",
				Password:  "861peUJXKI49xlb0FqECws7Q3a52RGgo",
				GroupID:   "5d9a6bce-14b5-11ee-ab7c-0242ac120006",
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
	ms := getNodeBirthMetrics()
	err = sp.node.PublishNodeBirth(ms)
	if err != nil {
		fmt.Println(err)
	}

	return &sp
}

func getBDSeq() int {
	s := system.LoadState() // load state from file
	bdSeq := s.BdSeq
	if bdSeq == 255 {
		s.BdSeq = 0 // recursive to 0
	} else {
		s.BdSeq = bdSeq + 1 // increase 1
	}
	system.PersistState(s) // write state to file
	return bdSeq
}

// ******************************************************************************
// *************************** Application Handlers *****************************
// ******************************************************************************
var messagePubHandlerA mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Application: Received message by, topic=", msg.Topic())
	topic := strings.Split(msg.Topic(), "/")
	if len(topic) >= 2 && topic[0] == "spBv1.0" {
		if topic[1] == "STATE" {
			fmt.Println("Application: Payload=", string(msg.Payload()))
		} else {
			p := sparkplug.Payload{}
			err := p.DecodePayload(msg.Payload())
			if err != nil {
				fmt.Println(err)
			}
			ms := p.Metrics
			for i := range ms {
				fmt.Println("Application: Metric: Name=", ms[i].Name, ", DataType=", ms[i].DataType.String(), ", Value=", ms[i].Value)
			}
		}
	}
}

var connectHandlerA mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Application: Connected")
}

var connectLostHandlerA mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Application: Connect lost: %v", err)
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
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

var reconnectingHandler mqtt.ReconnectHandler = func(client mqtt.Client, options *mqtt.ClientOptions) {
	fmt.Printf("Reconnect handler lost")
	// Note: Need to increment the bdSeq here for reconnecting
	wp, err := sparkplug.GetWillPayload(getBDSeq())
	if err != nil {
		fmt.Println("Error encoding will payload: ", err)
	}
	options.WillPayload = wp
}

// ******************************************************************************
// ******************************************************************************

func main_test() {
	// App Node
	// app := sparkplug.ClientApp{
	// 	Auth: sparkplug.Auth{
	// 		ServerUrl: "192.168.11.61",
	// 		Username:  "DMS",
	// 		Password:  "12345678901234567890123456789012",
	// 		GroupID:   "DMS",
	// 	},
	// 	MessagePubHandler:  &messagePubHandlerA,
	// 	ConnectHandler:     &connectHandlerA,
	// 	ConnectLostHandler: &connectLostHandlerA,
	// 	// ReconnectingHandler: &reconnectingHandlerA,
	// }
	// app.Connect()
	// app.SetOnline()

	// Client Node
	node := sparkplug.ClientNode{
		Config: sparkplug.Config{
			ServerUrl: "192.168.11.61",
			Username:  "f2179884-ff7b-11ed-ac90-0242ac150002",
			Password:  "Xm36xFJ4WCTid98op5jZ27hI10SvlVuy",
			GroupID:   "f2179884-ff7b-11ed-ac90-0242ac150002",
			NodeID:    "ac:de:48:00:11:22",
			ClientID:  "ac:de:48:00:11:22", // Device MAC
		},
		MessagePubHandler:   &messagePubHandler,
		ConnectHandler:      &connectHandler,
		ConnectLostHandler:  &connectLostHandler,
		ReconnectingHandler: &reconnectingHandler,
	}

	err := node.Connect(0)
	if err != nil {
		fmt.Println(err)
	}
	// Publish Node Birth
	ms := getNodeBirthMetrics()
	err = node.PublishNodeBirth(ms)
	if err != nil {
		fmt.Println(err)
	}
	// Publish Device Birth
	// Note: First Node Birth must be published
	ms1 := getDeviceBirthMetrics()
	deviceID := "ZB-4995887" // Get this ID from Zigbee device
	err = node.PublishDeviceBirth(deviceID, ms1)
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(time.Second * 60)

	// Publish Device Data
	// When there is change in device metrics
	ms2 := getDeviceDataMetrics_1()
	err = node.PublishDeviceData(deviceID, ms2)
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(time.Second * 60)

	// Publish Device Data
	// When there is change in device metrics
	ms3 := getDeviceDataMetrics_2()
	err = node.PublishDeviceData(deviceID, ms3)
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(time.Second * 60)

	// Publish Device Death
	// If Device cannot be contacted
	err = node.PublishDeviceDeath(deviceID)
	if err != nil {
		fmt.Println(err)
	}

	// Sleep for 3 minutes then simulate a Node "death"
	// by letting the code run its course
	time.Sleep(time.Second * 60 * 3)

}

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

func getDeviceBirthMetrics() []sparkplug.Metric {
	m1 := sparkplug.Metric{
		Name:     "Device Name",
		DataType: sparkplug.TypeString,
		Value:    "大廳鐵捲門",
	}
	m2 := sparkplug.Metric{
		Name:     "Model Type",
		DataType: sparkplug.TypeString,
		Value:    "DoorSensor",
	}
	m3 := sparkplug.Metric{
		Name:     "Model Name",
		DataType: sparkplug.TypeString,
		Value:    "ZB-001",
	}
	m4 := sparkplug.Metric{
		Name:     "Firmware Version",
		DataType: sparkplug.TypeString,
		Value:    "1.0.1",
	}
	m5 := sparkplug.Metric{
		Name:     "Battery Level",
		DataType: sparkplug.TypeInt,
		Value:    "100",
	}
	m6 := sparkplug.Metric{
		Name:     "zb/onoff",
		DataType: sparkplug.TypeInt,
		Value:    "0",
	}
	ms := []sparkplug.Metric{}
	ms = append(ms, m1)
	ms = append(ms, m2)
	ms = append(ms, m3)
	ms = append(ms, m4)
	ms = append(ms, m5)
	ms = append(ms, m6)

	return ms
}

func getDeviceDataMetrics_1() []sparkplug.Metric {
	m5 := sparkplug.Metric{
		Name:     "Battery Level",
		DataType: sparkplug.TypeInt,
		Value:    "90",
	}
	m6 := sparkplug.Metric{
		Name:     "zb/onoff",
		DataType: sparkplug.TypeInt,
		Value:    "1",
	}
	ms := []sparkplug.Metric{}
	ms = append(ms, m5)
	ms = append(ms, m6)

	return ms
}

func getDeviceDataMetrics_2() []sparkplug.Metric {
	m6 := sparkplug.Metric{
		Name:     "zb/onoff",
		DataType: sparkplug.TypeInt,
		Value:    "0",
	}
	ms := []sparkplug.Metric{}
	ms = append(ms, m6)

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
