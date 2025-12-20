package main

import (
	"fmt"

	"github.com/getlantern/systray"
	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func onReady(icon []byte, client *pepeunit.PepeunitClient) {
	systray.SetIcon(icon)
	systray.SetTitle("Tray Example")
	systray.SetTooltip("Minimal Tray App")

	mButton := systray.AddMenuItem("Меню", "Нажмите для выполнения")
	go func() {
		for range mButton.ClickedCh {
			go startGame(client)
		}
	}()

	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	fmt.Println("Приложение завершено")
}
