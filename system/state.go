package system

import (
	"encoding/json"
	"github.com/mtrossbach/waechter/internal/config"
	"github.com/mtrossbach/waechter/internal/log"
	"github.com/mtrossbach/waechter/system/alarm"
	"github.com/mtrossbach/waechter/system/arm"
	"github.com/mtrossbach/waechter/system/device"
	"os"
	"path"
	"time"
)

type State struct {
	ArmMode        arm.Mode   `json:"armMode"`
	Alarm          alarm.Type `json:"alarm"`
	ArmModeUpdated time.Time  `json:"ArmModeUpdated"`
	BdSeq          int        `json:"bdSeq"`
}

func LoadState() State {
	var state State

	// initial value if state file not exist
	state.ArmMode = arm.Perimeter
	state.Alarm = alarm.Burglar
	state.BdSeq = 0

	filename := path.Join(config.Dir(), "state")
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Error().Str("filename", filename).Err(err).Msg("Could not read state file")
		return state
	}

	err = json.Unmarshal(data, &state)
	if err != nil {
		log.Error().Err(err).Msg("Could not unmarshal state file")
		return state
	}

	log.Info().Str("filename", filename).Msg("State loaded")
	return state
}

func PersistState(state State) {
	data, err := json.Marshal(state)
	if err != nil {
		log.Error().Err(err).Msg("Could not marshal state")
		return
	}

	filename := path.Join(config.Dir(), "state")
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Error().Err(err).Msg("Could not write state file")
		return
	}
}

func (s State) Armed() bool {
	return s.ArmMode != arm.Disarmed
}

func (s State) stateActorPayload() device.StateActorPayload {
	return device.StateActorPayload{
		ArmMode: s.ArmMode,
		Alarm:   s.Alarm,
	}
}

func (s State) alarmActorPayload() device.AlarmActorPayload {
	return device.AlarmActorPayload{Alarm: s.Alarm}
}
