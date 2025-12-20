package main

import (
	"context"
	_ "embed"
	"os"
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
		os.Exit(1)
	}

	ctx := context.Background()
	if pepeClient.GetMQTTClient() != nil {
		if err := pepeClient.GetMQTTClient().Connect(ctx); err != nil {
			os.Exit(1)
		}
		pepeClient.SetMQTTInputHandler(nil)
		pepeClient.SubscribeAllSchemaTopics(ctx)
	}
	go pepeClient.RunMainCycle(ctx, nil)
	mainthread.Init(func() {
		registerGlobalHotkey(pepeClient)
		registerOptionHotkeys(pepeClient)
	})
	icon, err := loadIcon(iconData)
	if err != nil {
		os.Exit(1)
	}

	systray.Run(func() {
		onReady(icon, pepeClient)
	}, onExit)
}
