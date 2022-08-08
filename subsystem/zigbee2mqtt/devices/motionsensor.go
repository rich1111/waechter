package devices

import (
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mtrossbach/waechter/misc"
	"github.com/mtrossbach/waechter/subsystem/zigbee2mqtt/model"
	"github.com/mtrossbach/waechter/subsystem/zigbee2mqtt/zigbee"
	"github.com/mtrossbach/waechter/system"
)

type motionsensorStatusPayload struct {
	Battery     int  `json:"battery"`
	Linkquality int  `json:"linkquality"`
	Occupancy   bool `json:"occupancy"`
	Tamper      bool `json:"tamper"`
}

type genericMotionSensor struct {
	deviceInfo    model.Z2MDeviceInfo
	z2mManager    *zigbee.Z2MManager
	systemControl system.SystemControl
}

func newGenericMotionSensor(deviceInfo model.Z2MDeviceInfo, z2mManager *zigbee.Z2MManager) *genericMotionSensor {
	return &genericMotionSensor{
		deviceInfo: deviceInfo,
		z2mManager: z2mManager,
	}
}

func (s *genericMotionSensor) GetId() string {
	return s.deviceInfo.IeeeAddress
}

func (s *genericMotionSensor) GetDisplayName() string {
	return s.deviceInfo.FriendlyName
}

func (s *genericMotionSensor) GetSubsystem() string {
	return model.SubsystemName
}

func (s *genericMotionSensor) GetType() system.DeviceType {
	return system.MotionSensor
}

func (s *genericMotionSensor) OnSystemStateChanged(state system.State) {

}

func (s *genericMotionSensor) OnDeviceAnnounced() {

}

func (s *genericMotionSensor) Setup(systemControl system.SystemControl) {
	misc.Log.Debugf("Setup device %v:%v:%v", s.GetType(), s.GetId(), s.GetDisplayName())
	s.systemControl = systemControl
	s.z2mManager.Subscribe(s.deviceInfo.FriendlyName, s.handleMessage)
}

func (s *genericMotionSensor) Teardown() {
	misc.Log.Debugf("Teardown device %v:%v:%v", s.GetType(), s.GetId(), s.GetDisplayName())
	s.systemControl = nil
	s.z2mManager.Unsubscribe(s.deviceInfo.FriendlyName)
}

func (s *genericMotionSensor) handleMessage(msg mqtt.Message) {
	var payload motionsensorStatusPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		misc.Log.Warnf("Could not parse payload: %v", string(msg.Payload()))
		return
	}

	misc.Log.Debugf("Got data: %v", string(msg.Payload()))

	if payload.Battery > 0 {
		s.systemControl.ReportBattery(s, float32(payload.Battery)/float32(100))
	}

	if payload.Linkquality > 0 {
		s.systemControl.ReportBattery(s, float32(payload.Linkquality)/float32(255))
	}

	if payload.Tamper {
		s.systemControl.ReportTampered(s)
	}

	if payload.Occupancy {
		s.systemControl.ReportTriggered(s)
	}
}