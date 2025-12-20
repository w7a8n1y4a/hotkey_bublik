package main

import (
	"log"

	"picker/internal/config"
	"picker/internal/hotkeys"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
	"golang.design/x/hotkey"
)

func registerGlobalHotkey(client *pepeunit.PepeunitClient) {
	cfg := config.GetConfig()
	if cfg.LaunchHotkeyMain == nil {
		log.Println("Global hotkey is disabled (HOTKEY_MAIN is null/empty/missing)")
		return
	}

	mods, key, display, err := hotkeys.ParseHotkeySpec(*cfg.LaunchHotkeyMain)
	if err != nil {
		log.Printf("hotkey: invalid HOTKEY_MAIN=%q: %v; global hotkey is disabled", *cfg.LaunchHotkeyMain, err)
		return
	}

	log.Printf("Trying to register global hotkey %s...", display)
	hk := hotkey.New(mods, key)

	if err := hk.Register(); err != nil {
		log.Printf("hotkey: failed to register global hotkey %s: %v", display, err)
		return
	}

	log.Printf("Global hotkey %s is registered\n", display)
	go func() {
		for {
			select {
			case <-hk.Keydown():
				log.Printf("Global hotkey %s is down\n", display)
				go startGame(client)
			case <-hk.Keyup():
				log.Printf("Global hotkey %s is up\n", display)
			}
		}
	}()
}
