package main

import (
	"context"
	_ "embed"
	"log"
	"time"

	"github.com/getlantern/systray"
	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
	"golang.design/x/hotkey/mainthread"
)

//go:embed assets/icons/64.png
var iconData []byte

func main() {
	pepeClient, err := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
		EnvFilePath:      "env.json",
		SchemaFilePath:   "schema.json",
		LogFilePath:      "log.json",
		EnableMQTT:       true,
		EnableREST:       true,
		CycleSpeed:       100 * time.Millisecond,
		RestartMode:      pepeunit.RestartModeRestartExec,
		SkipVersionCheck: true,
	})
	if err != nil {
		log.Fatalf("init pepeunit client failed: %v", err)
	}

	log.Println("App starting...")

	ctx := context.Background()
	if pepeClient.GetMQTTClient() != nil {
		if err := pepeClient.GetMQTTClient().Connect(ctx); err != nil {
			log.Fatalf("mqtt connect failed: %v", err)
		}
		pepeClient.SetMQTTInputHandler(nil)
		if err := pepeClient.SubscribeAllSchemaTopics(ctx); err != nil {
			log.Printf("subscribe topics failed: %v", err)
		}
	}
	go pepeClient.RunMainCycle(ctx, nil)
	mainthread.Init(func() {
		registerGlobalHotkey(pepeClient)
		registerOptionHotkeys(pepeClient)
	})
	icon, err := loadIcon(iconData)
	if err != nil {
		log.Fatal("Ошибка загрузки иконки:", err)
	}

	systray.Run(func() {
		onReady(icon, pepeClient)
	}, onExit)
}
