package app

import (
	"strings"

	"picker/internal/config"
	"picker/internal/hotkeys"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
	"golang.design/x/hotkey"
)

func RegisterGlobalHotkey(client *pepeunit.PepeunitClient) {
	cfg := config.GetConfig()
	if cfg.LaunchHotkeyMain == nil {
		return
	}

	mods, key, _, err := hotkeys.ParseHotkeySpec(*cfg.LaunchHotkeyMain)
	if err != nil {
		return
	}

	hk := hotkey.New(mods, key)

	if err := hk.Register(); err != nil {
		return
	}
	go func() {
		for {
			select {
			case <-hk.Keydown():
				go StartGame(client)
			case <-hk.Keyup():
			}
		}
	}()
}

func RegisterOptionHotkeys(client *pepeunit.PepeunitClient) {
	if client == nil {
		return
	}

	stateData := loadStateData(client)

	if len(stateData) == 0 {
		return
	}

	settings := client.GetSettings()

	type hotkeyBinding struct {
		topic       string
		payload     string
		commandName string
		mods        []hotkey.Modifier
		key         hotkey.Key
	}

	bindings := make(map[string]hotkeyBinding)

	for nodeUUID, items := range stateData {
		for _, pair := range items {
			if len(pair) < 3 {
				continue
			}
			rawHotkey := strings.TrimSpace(pair[2])
			if rawHotkey == "" {
				continue
			}

			mods, key, display, err := hotkeys.ParseHotkeySpec(rawHotkey)
			if err != nil {
				continue
			}

			if _, exists := bindings[display]; exists {
				continue
			}

			if len(pair) < 2 {
				continue
			}

			topicName := settings.PU_DOMAIN + "/" + nodeUUID + "/pepeunit"
			bindings[display] = hotkeyBinding{
				topic:       topicName,
				payload:     pair[1],
				commandName: pair[0],
				mods:        mods,
				key:         key,
			}
		}
	}

	if len(bindings) == 0 {
		return
	}

	for _, bind := range bindings {
		hk := hotkey.New(bind.mods, bind.key)

		if err := hk.Register(); err != nil {
			continue
		}

		go func(hk *hotkey.Hotkey, bind hotkeyBinding) {
			for {
				select {
				case <-hk.Keydown():
					if client != nil && client.GetMQTTClient() != nil {
						client.GetLogger().Info("Send command '" + bind.commandName + "' to MQTT on topic '" + bind.topic + "'")
						client.GetMQTTClient().Publish(bind.topic, bind.payload)
					}
				case <-hk.Keyup():
				}
			}
		}(hk, bind)
	}
}


