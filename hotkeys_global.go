package main

import (
	"picker/internal/config"
	"picker/internal/hotkeys"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
	"golang.design/x/hotkey"
)

func registerGlobalHotkey(client *pepeunit.PepeunitClient) {
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
				go startGame(client)
			case <-hk.Keyup():
			}
		}
	}()
}
