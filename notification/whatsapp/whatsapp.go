package whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mtrossbach/waechter/internal/config"
	"github.com/mtrossbach/waechter/internal/i18n"
	"github.com/mtrossbach/waechter/internal/log"
	"github.com/mtrossbach/waechter/system/alarm"
	"github.com/mtrossbach/waechter/system/device"
	"github.com/mtrossbach/waechter/system/zone"
	"io"
	"net/http"
	"time"
)

type WhatsApp struct {
	client *http.Client
	config config.WhatsAppConfiguration
}

func NewWhatsApp(configuration config.WhatsAppConfiguration) *WhatsApp {
	return &WhatsApp{client: &http.Client{
		Timeout: 60 * time.Second,
	}, config: configuration}
}

func (w *WhatsApp) send(phone string, template string, lang string, parameters []string) error {
	if len(phone) < 5 {
		log.Error().Str("phone", phone).Msg("Could not send WhatsApp message. Invalid phone number")
		return fmt.Errorf("invalid phone number: %v", phone)
	}
	if len(template) < 1 {
		log.Error().Str("template", template).Msg("Could not send WhatsApp message. Invalid template name")
		return fmt.Errorf("invalid template name: %v", template)
	}
	var ps []Parameter
	for _, s := range parameters {
		ps = append(ps, Parameter{
			Type: "text",
			Text: s,
		})
	}
	payload := MessagePayload{
		MessagingProduct: "whatsapp",
		To:               phone,
		Type:             "template",
		Template: Template{
			Name:     template,
			Language: Language{Code: lang},
			Components: []Component{{
				Type:       "body",
				Parameters: ps,
			}},
		},
	}

	var response interface{}

	r, err := w.post(w.config.PhoneId, payload, &response)
	if err != nil {
		log.Error().Err(err).Str("phone", phone).Interface("response", response).Msg("Could not send WhatsApp message")
		return err
	}
	if r.StatusCode >= 300 {
		log.Error().Str("phone", phone).Interface("response", response).Int("status-code", r.StatusCode).Msg("Could not send WhatsApp message")
		return fmt.Errorf("could not send message to whatsapp, statuscode is %v", r.StatusCode)
	}
	log.Info().Str("phone", phone).Msg("Successfully sent message via WhatsApp")
	return nil
}

func (w *WhatsApp) post(phoneId string, payload MessagePayload, response interface{}) (*http.Response, error) {
	url := fmt.Sprintf("https://graph.facebook.com/v13.0/%v/messages", phoneId)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", w.config.Token))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if response == nil {
		return resp, nil
	}

	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return nil, err
	}

	_ = resp.Body.Close()
	return resp, nil
}

func (w *WhatsApp) Name() string {
	return "WhatsApp"
}

func (w *WhatsApp) NotifyAlarm(person config.Person, systemName string, a alarm.Type, device device.Spec, zone zone.Zone) bool {
	err := w.send(person.WhatsApp, w.config.TemplateAlarm, person.Lang, []string{
		systemName, i18n.TranslateAlarm(person.Lang, a), device.HumanReadableName(),
	})

	return err == nil
}

func (w *WhatsApp) NotifyRecovery(person config.Person, systemName string, device device.Spec, zone zone.Zone) bool {
	err := w.send(person.WhatsApp, w.config.TemplateRecover, person.Lang, []string{
		systemName,
	})

	return err == nil
}

func (w *WhatsApp) NotifyBatteryLevel(person config.Person, systemName string, device device.Spec, zone zone.Zone, batteryLevel float32) bool {
	err := w.send(person.WhatsApp, w.config.TemplateNotification, person.Lang, []string{
		systemName, device.HumanReadableName(), i18n.Translate(person.Lang, i18n.WALowBattery),
	})

	return err == nil
}

func (w *WhatsApp) NotifyLinkQuality(person config.Person, systemName string, device device.Spec, zone zone.Zone, quality float32) bool {
	err := w.send(person.WhatsApp, w.config.TemplateNotification, person.Lang, []string{
		systemName, device.HumanReadableName(), i18n.Translate(person.Lang, i18n.WALowLinkQuality),
	})

	return err == nil
}

func (s *WhatsApp) NotifyDeviceAvailable(person config.Person, systemName string, device device.Spec, zone zone.Zone) bool {
	return true
}

func (s *WhatsApp) NotifyDeviceUnAvailable(person config.Person, systemName string, device device.Spec, zone zone.Zone) bool {
	return true
}

func (w *WhatsApp) NotifyHumidityValue(person config.Person, systemName string, device device.Spec, zone zone.Zone, humidity float32) bool {

	return true
}

func (w *WhatsApp) NotifyTemperatureValue(person config.Person, systemName string, device device.Spec, zone zone.Zone, temperature float32) bool {

	return true
}

func (w *WhatsApp) NotifyMotionSensor(person config.Person, systemName string, device device.Spec, zone zone.Zone, motion bool) bool {

	return true
}

func (w *WhatsApp) NotifyContactSensor(person config.Person, systemName string, device device.Spec, zone zone.Zone, contact bool) bool {

	return true
}

func (w *WhatsApp) NotifySmokeSensor(person config.Person, systemName string, device device.Spec, zone zone.Zone, smoke bool) bool {

	return true
}

func (w *WhatsApp) NotifyAutoArm(person config.Person, systemName string) bool {
	err := w.send(person.WhatsApp, w.config.TemplateAutoArm, person.Lang, []string{
		systemName,
	})

	return err == nil
}

func (w *WhatsApp) NotifyAutoDisarm(person config.Person, systemName string) bool {
	err := w.send(person.WhatsApp, w.config.TemplateAutoDisarm, person.Lang, []string{
		systemName,
	})

	return err == nil
}
