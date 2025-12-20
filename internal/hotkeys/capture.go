package hotkeys

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.design/x/hotkey"
)

func CaptureHotkeyFromEbiten() string {
	var mods []hotkey.Modifier
	var key string

	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		mods = append(mods, hotkey.ModCtrl)
	}
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		mods = append(mods, hotkey.ModShift)
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) {
		mods = append(mods, hotkey.Mod1)
	}
	if ebiten.IsKeyPressed(ebiten.KeyMeta) {
		mods = append(mods, hotkey.Mod4)
	}

	key = captureKeyFromEbiten()

	if key == "" || len(mods) == 0 {
		return ""
	}

	return FormatHotkey(mods, key)
}

func captureKeyFromEbiten() string {
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		if ebiten.IsKeyPressed(k) {
			return string(rune('A' + (k - ebiten.KeyA)))
		}
	}

	for k := ebiten.Key0; k <= ebiten.Key9; k++ {
		if ebiten.IsKeyPressed(k) {
			return string(rune('0' + (k - ebiten.Key0)))
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyF1) {
		return "F1"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF2) {
		return "F2"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF3) {
		return "F3"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF4) {
		return "F4"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF5) {
		return "F5"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF6) {
		return "F6"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF7) {
		return "F7"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF8) {
		return "F8"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF9) {
		return "F9"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF10) {
		return "F10"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF11) {
		return "F11"
	}
	if ebiten.IsKeyPressed(ebiten.KeyF12) {
		return "F12"
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		return "SPACE"
	}
	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		return "TAB"
	}
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return "ESC"
	}
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		return "ENTER"
	}
	if ebiten.IsKeyPressed(ebiten.KeyDelete) {
		return "DELETE"
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		return "LEFT"
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		return "RIGHT"
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		return "UP"
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		return "DOWN"
	}

	return ""
}

func FormatHotkeyFromString(hotkeyStr string) string {
	if strings.TrimSpace(hotkeyStr) == "" {
		return "Не установлены"
	}

	_, _, display, err := ParseHotkeySpec(hotkeyStr)
	if err != nil {
		return strings.ToUpper(hotkeyStr)
	}

	return display
}

func ValidateHotkey(hotkeyStr string) error {
	if strings.TrimSpace(hotkeyStr) == "" {
		return fmt.Errorf("hotkey cannot be empty")
	}
	_, _, _, err := ParseHotkeySpec(hotkeyStr)
	return err
}
