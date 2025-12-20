package hotkeys

import (
	"fmt"
	"strings"

	"golang.design/x/hotkey"
)

func ParseHotkeySpec(spec string) ([]hotkey.Modifier, hotkey.Key, string, error) {
	raw := strings.TrimSpace(spec)
	if raw == "" {
		return nil, 0, "", fmt.Errorf("empty hotkey")
	}

	parts := strings.Split(raw, "+")
	var mods []hotkey.Modifier
	var keyTok string

	addMod := func(m hotkey.Modifier) {
		for _, existing := range mods {
			if existing == m {
				return
			}
		}
		mods = append(mods, m)
	}

	for _, p := range parts {
		t := strings.ToUpper(strings.TrimSpace(p))
		if t == "" {
			continue
		}
		switch t {
		case "CTRL", "CONTROL":
			addMod(hotkey.ModCtrl)
		case "SHIFT":
			addMod(hotkey.ModShift)
		case "ALT", "OPTION":
			addMod(hotkey.Mod1)
		case "CMD", "COMMAND", "META", "SUPER", "WIN", "WINDOWS":
			addMod(hotkey.Mod4)
		default:
			if keyTok != "" {
				return nil, 0, "", fmt.Errorf("multiple key tokens: %q and %q", keyTok, t)
			}
			keyTok = t
		}
	}

	if keyTok == "" {
		return nil, 0, "", fmt.Errorf("missing key")
	}

	key, ok := keyMap[keyTok]
	if !ok {
		return nil, 0, "", fmt.Errorf("unsupported key %q", keyTok)
	}

	display := FormatHotkey(mods, keyTok)

	return mods, key, display, nil
}

func FormatHotkey(mods []hotkey.Modifier, key string) string {
	var dispParts []string
	has := func(m hotkey.Modifier) bool {
		for _, mm := range mods {
			if mm == m {
				return true
			}
		}
		return false
	}
	if has(hotkey.ModCtrl) {
		dispParts = append(dispParts, "CTRL")
	}
	if has(hotkey.ModShift) {
		dispParts = append(dispParts, "SHIFT")
	}
	if has(hotkey.Mod1) {
		dispParts = append(dispParts, "ALT")
	}
	if has(hotkey.Mod4) {
		dispParts = append(dispParts, "META")
	}
	dispParts = append(dispParts, strings.ToUpper(key))

	return strings.Join(dispParts, "+")
}

var keyMap = map[string]hotkey.Key{
	"A": hotkey.KeyA, "B": hotkey.KeyB, "C": hotkey.KeyC, "D": hotkey.KeyD, "E": hotkey.KeyE,
	"F": hotkey.KeyF, "G": hotkey.KeyG, "H": hotkey.KeyH, "I": hotkey.KeyI, "J": hotkey.KeyJ,
	"K": hotkey.KeyK, "L": hotkey.KeyL, "M": hotkey.KeyM, "N": hotkey.KeyN, "O": hotkey.KeyO,
	"P": hotkey.KeyP, "Q": hotkey.KeyQ, "R": hotkey.KeyR, "S": hotkey.KeyS, "T": hotkey.KeyT,
	"U": hotkey.KeyU, "V": hotkey.KeyV, "W": hotkey.KeyW, "X": hotkey.KeyX, "Y": hotkey.KeyY,
	"Z": hotkey.KeyZ,
	"0": hotkey.Key0, "1": hotkey.Key1, "2": hotkey.Key2, "3": hotkey.Key3, "4": hotkey.Key4,
	"5": hotkey.Key5, "6": hotkey.Key6, "7": hotkey.Key7, "8": hotkey.Key8, "9": hotkey.Key9,
	"SPACE":  hotkey.KeySpace,
	"TAB":    hotkey.KeyTab,
	"ESC":    hotkey.KeyEscape,
	"ESCAPE": hotkey.KeyEscape,
	"ENTER":  hotkey.KeyReturn,
	"RETURN": hotkey.KeyReturn,
	"DELETE": hotkey.KeyDelete,
	"LEFT":   hotkey.KeyLeft,
	"RIGHT":  hotkey.KeyRight,
	"UP":     hotkey.KeyUp,
	"DOWN":   hotkey.KeyDown,
	"F1":     hotkey.KeyF1,
	"F2":     hotkey.KeyF2,
	"F3":     hotkey.KeyF3,
	"F4":     hotkey.KeyF4,
	"F5":     hotkey.KeyF5,
	"F6":     hotkey.KeyF6,
	"F7":     hotkey.KeyF7,
	"F8":     hotkey.KeyF8,
	"F9":     hotkey.KeyF9,
	"F10":    hotkey.KeyF10,
	"F11":    hotkey.KeyF11,
	"F12":    hotkey.KeyF12,
	"F13":    hotkey.KeyF13,
	"F14":    hotkey.KeyF14,
	"F15":    hotkey.KeyF15,
	"F16":    hotkey.KeyF16,
	"F17":    hotkey.KeyF17,
	"F18":    hotkey.KeyF18,
	"F19":    hotkey.KeyF19,
	"F20":    hotkey.KeyF20,
}

