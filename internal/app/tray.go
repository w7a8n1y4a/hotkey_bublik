package app

import (
	"bytes"
	"image"
	"image/png"

	"github.com/getlantern/systray"
	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func OnReady(icon []byte, client *pepeunit.PepeunitClient) {
	systray.SetIcon(icon)
	systray.SetTitle("Tray Example")
	systray.SetTooltip("Minimal Tray App")

	mButton := systray.AddMenuItem("Меню", "Нажмите для выполнения")
	go func() {
		for range mButton.ClickedCh {
			go StartGame(client)
		}
	}()

	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func OnExit() {
}

func LoadIcon(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
