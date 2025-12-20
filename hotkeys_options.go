package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"picker/internal/hotkeys"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
	"golang.design/x/hotkey"
)

func registerOptionHotkeys(client *pepeunit.PepeunitClient) {
	if client == nil {
		return
	}

	log.Println("Trying to register option hotkeys from state...")

	ctx := context.Background()
	stateData := make(map[string][][]string)

	if stateStr, err := client.GetStateStorage(ctx); err == nil && stateStr != "" && stateStr != "\"\"" {
		if err := json.Unmarshal([]byte(stateStr), &stateData); err != nil {
			var wrapped string
			if err2 := json.Unmarshal([]byte(stateStr), &wrapped); err2 == nil && wrapped != "" {
				_ = json.Unmarshal([]byte(wrapped), &stateData)
			}
		}
	}

	if len(stateData) == 0 {
		log.Println("No state data found for option hotkeys")
		return
	}

	settings := client.GetSettings()

	type hotkeyBinding struct {
		topic   string
		payload string
		mods    []hotkey.Modifier
		key     hotkey.Key
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
				log.Printf("hotkey: invalid hotkey spec %q: %v; skipping", rawHotkey, err)
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
				topic:   topicName,
				payload: pair[1],
				mods:    mods,
				key:     key,
			}
		}
	}

	if len(bindings) == 0 {
		log.Println("No option hotkeys found in state")
		return
	}

	for display, bind := range bindings {
		hk := hotkey.New(bind.mods, bind.key)

		if err := hk.Register(); err != nil {
			log.Printf("hotkey: failed to register option hotkey %s: %v", display, err)
			continue
		}

		log.Printf("Option hotkey %s is registered\n", display)

		go func(hk *hotkey.Hotkey, bind hotkeyBinding, display string) {
			for {
				select {
				case <-hk.Keydown():
					log.Printf("Option hotkey %s is down, publishing to %s\n", display, bind.topic)
					if client != nil && client.GetMQTTClient() != nil {
						if err := client.GetMQTTClient().Publish(bind.topic, bind.payload); err != nil {
							log.Printf("failed to publish MQTT message for hotkey %s: %v", display, err)
						}
					}
				case <-hk.Keyup():
					log.Printf("Option hotkey %s is up\n", display)
				}
			}
		}(hk, bind, display)
	}
}
