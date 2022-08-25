package homeassistant

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/mtrossbach/waechter/internal/cfg"
	"github.com/mtrossbach/waechter/subsystem/device/homeassistant/api"
	"github.com/mtrossbach/waechter/subsystem/device/homeassistant/device"
	"github.com/mtrossbach/waechter/subsystem/device/homeassistant/model"
	"github.com/mtrossbach/waechter/system"
	"sync"
)

type homeassistant struct {
	c       *websocket.Conn
	api     *api.Api
	devices sync.Map
}

func New() *homeassistant {
	ins := &homeassistant{
		api:     api.NewApi(cfg.GetString("homeassistant.url"), cfg.GetString("homeassistant.token")),
		devices: sync.Map{},
	}
	return ins
}

func (ha *homeassistant) GetName() string {
	return model.SubsystemName
}

func (ha *homeassistant) Start(systemController system.Controller) {
	ha.api.Connect()

	st, _ := ha.api.GetStates()
	for _, s := range st.Result {
		if s.Attributes.DeviceClass == "motion" && s.Attributes.MotionValid {
			dev := device.NewMotionSensor(system.Device{
				Id:   entityId2Id(s.EntityID),
				Name: s.Attributes.FriendlyName,
				Type: "",
			}, ha.api, s.EntityID)
			dev.Setup(systemController)
			ha.devices.Store(entityId2Id(s.EntityID), dev)
			system.DInfo(dev.Device).Msg("Created.")
		}
	}
}

func (ha *homeassistant) Stop() {

}

func entityId2Id(ieeeAddress string) string {
	return fmt.Sprintf("%v-%v", "ha", ieeeAddress)
}
