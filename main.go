package main

import (
	"context"
	_ "embed"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
	"golang.design/x/hotkey/mainthread"

	"picker/internal/app"
)

func flatten(m map[string][]string) []string {
	out := make([]string, 0)
	for _, arr := range m {
		out = append(out, arr...)
	}
	return out
}

//go:embed assets/icons/64.png
var iconData []byte

func handleInputMessages(client *pepeunit.PepeunitClient, msg pepeunit.MQTTMessage, uiEnabled bool) {
	if !uiEnabled {
		return
	}

	topic := msg.Topic
	for key, topics := range client.GetSchema().GetInputBaseTopic() {
		for _, topicURL := range topics {
			if topicURL != topic {
				continue
			}
			switch key {
			case string(pepeunit.BaseInputTopicTypeEnvUpdatePepeunit):
				go func() {
					app.RegisterGlobalHotkey(client)
					app.RegisterOptionHotkeys(client)
				}()
			case string(pepeunit.BaseInputTopicTypeSchemaUpdatePepeunit):
				go app.RegisterOptionHotkeys(client)
			}
			client.GetLogger().Info("Handled base command topic: " + key)
			return
		}
	}
}

func setupMQTT(ctx context.Context, client *pepeunit.PepeunitClient, uiEnabled bool) error {
	mqttClient := client.GetMQTTClient()
	if mqttClient == nil {
		return nil
	}

	client.SetMQTTInputHandler(func(msg pepeunit.MQTTMessage) {
		handleInputMessages(client, msg, uiEnabled)
	})

	if err := mqttClient.Connect(ctx); err != nil {
		return err
	}

	client.GetLogger().Info("MQTT connected, subscribing to schema topics")
	if err := client.SubscribeAllSchemaTopics(ctx); err != nil {
		client.GetLogger().Error(err.Error())
	} else {
		client.GetLogger().Info("Subscribed to schema topics")
	}

	return nil
}

func disconnectMQTT(ctx context.Context, client *pepeunit.PepeunitClient) {
	if client == nil {
		return
	}
	mqttClient := client.GetMQTTClient()
	if mqttClient == nil {
		return
	}
	_ = mqttClient.Disconnect(ctx)
	client.GetLogger().Info("MQTT disconnected")
}

func main() {
	runWithoutTray := flag.Bool("no-tray", false, "run without system tray UI and hotkeys")
	flag.Parse()

	pepeClient, err := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
		EnvFilePath:          "env.json",
		SchemaFilePath:       "schema.json",
		LogFilePath:          "log.json",
		EnableMQTT:           true,
		EnableREST:           true,
		CycleSpeed:           100 * time.Millisecond,
		RestartMode:          pepeunit.RestartModeRestartExec,
		FFVersionCheckEnable: true,
		FFConsoleLogEnable:   true,
	})
	if err != nil {
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := setupMQTT(ctx, pepeClient, !*runWithoutTray); err != nil {
		os.Exit(1)
	}

	go pepeClient.RunMainCycle(ctx, nil)

	if *runWithoutTray {
		<-ctx.Done()
		disconnectMQTT(context.Background(), pepeClient)
		return
	}

	mainthread.Init(func() {
		app.RegisterGlobalHotkey(pepeClient)
		app.RegisterOptionHotkeys(pepeClient)
	})
	icon, err := app.LoadIcon(iconData)
	if err != nil {
		os.Exit(1)
	}

	systray.Run(func() {
		app.OnReady(icon, pepeClient)
	}, func() {
		stop()
		app.OnExit()
	})
	<-ctx.Done()
	disconnectMQTT(context.Background(), pepeClient)
}
