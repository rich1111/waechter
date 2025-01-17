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
package sparkplug

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type ClientApp struct {
	client              mqtt.Client
	Auth                Auth
	MessagePubHandler   *mqtt.MessageHandler
	ConnectHandler      *mqtt.OnConnectHandler
	ConnectLostHandler  *mqtt.ConnectionLostHandler
	ReconnectingHandler *mqtt.ReconnectHandler
}

type Auth struct {
	ServerUrl string
	Username  string
	Password  string
	GroupID   string
}

// Connect will connect to the MQTT broker

func (c *ClientApp) Connect() error {

	opts := mqtt.NewClientOptions()
	// Set the connection parameters
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", c.Auth.ServerUrl, 1883))
	opts.SetClientID(c.Auth.Username)
	opts.SetUsername(c.Auth.Username)
	opts.SetPassword(c.Auth.Password)
	// Set the handlers
	opts.SetDefaultPublishHandler(*c.MessagePubHandler)
	opts.OnConnect = *c.ConnectHandler
	opts.OnConnectionLost = *c.ConnectLostHandler
	//opts.OnReconnecting = *c.ReconnectingHandler

	// Set to Auto re-connect
	opts.SetAutoReconnect(true)
	// Set Clean Session on broker to false
	// Broker will remember the previous connection
	opts.CleanSession = false

	// Set the Will topic and message
	opts.WillEnabled = true
	opts.WillQos = 1
	opts.WillRetained = true
	opts.WillTopic = namespace + "/" + state + "/" + c.Auth.Username
	// Encode and set Will payload
	wp := getOnlinePayload(false)
	opts.WillPayload = wp

	c.client = mqtt.NewClient(opts)
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		err := token.Error()
		fmt.Println("Error connecting to MQTT broker: ", err)
		return err
	}

	// If client app is "SCADA", subscribe to all messages
	if c.Auth.Username == "SCADA" {
		topic := namespace + "/#"
		c.subscribe(topic)
	} else {
		// Subscribe to only "state" and messages related to this "host id"
		topic1 := namespace + "/" + state + "/" + c.Auth.Username
		topic2 := namespace + "/" + c.Auth.Username + "/#"
		c.subscribe(topic1)
		c.subscribe(topic2)
	}

	return nil
}

// getOnlinePayload gets the Payload whether online or offline
func getOnlinePayload(isOnline bool) []byte {
	p := fmt.Sprintf(`{
		"online": %t,
		"timestamp": "%s"
	}`, isOnline, time.Now().Format(time.RFC3339))
	return []byte(p)
}

func (c *ClientApp) subscribe(topic string) {
	token := c.client.Subscribe(topic, byte(1), nil)
	token.Wait()
	fmt.Println("Subscribed to topic:", topic)
}

// SetOnline publishes and set state of application to "online"
func (c *ClientApp) SetOnline() error {
	topic := namespace + "/" + state + "/" + c.Auth.Username
	p := getOnlinePayload(true)
	token := c.client.Publish(topic, byte(1), false, p)
	token.Wait()
	return nil
}

func RequestNodeRebirth(client mqtt.Client, groupID string, nodeID string) error {
	topic := namespace + "/" + groupID + "/" + MESSAGETYPE_NCMD + "/" + nodeID
	ms := []Metric{}
	m1 := Metric{
		Name:     "Node Control/Rebirth",
		DataType: TypeBool,
		Value:    "true",
	}
	ms = append(ms, m1)
	p := Payload{
		Metrics: ms,
	}
	// Encode payload
	b, err := p.EncodePayload(false)
	if err != nil {
		fmt.Println("Error encoding payload: ", err)
		return err
	}
	token := client.Publish(topic, 0, false, b)
	token.Wait()
	return nil
}

// Disconnect disconnects the client from the MQTT server
func (c *ClientApp) Disconnect() {
	c.client.Disconnect(0)
}
