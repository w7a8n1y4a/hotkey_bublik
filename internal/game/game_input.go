package game

import "github.com/hajimehoshi/ebiten/v2"

func (g *Game) handleKey(key ebiten.Key, action func()) {
	keyPressed := false
	if key == ebiten.Key(ebiten.MouseButtonLeft) {
		keyPressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	} else if key == ebiten.Key(ebiten.MouseButtonRight) {
		keyPressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	} else {
		keyPressed = ebiten.IsKeyPressed(key)
	}

	if keyPressed {
		if !g.KeyDownMap[key] {
			g.KeyDownMap[key] = true
			action()
		}
	} else {
		g.KeyDownMap[key] = false
	}
}

func (g *Game) handleKeyCombination(key ebiten.Key, modifier ebiten.Key, action func()) {
	if ebiten.IsKeyPressed(key) && ebiten.IsKeyPressed(modifier) {
		if !g.KeyDownMap[key] {
			g.KeyDownMap[key] = true
			action()
		}
	} else {
		g.KeyDownMap[key] = false
	}
}

func (g *Game) StartTextInput(callback func(string)) {
	g.InputMode = ModeTextInput
	g.TextInput = ""
	g.OnTextInputDone = callback
	g.OnTextInputCancel = nil
	g.CursorTick = 0
	g.BackspaceFrames = 0
}

func (g *Game) AwaitTextInput(isFirstWrite bool) (string, bool) {
	return g.awaitInput(ModeTextInput, func(resultChan chan textInputResult) {
		g.TextInput = ""
		g.IsFirstWrite = isFirstWrite
		g.CursorTick = 0
		g.BackspaceFrames = 0

		finish := func(res textInputResult) {
			g.OnTextInputDone = nil
			g.OnTextInputCancel = nil
			g.InputMode = ModeGame
			resultChan <- res
			close(resultChan)
		}

		g.OnTextInputDone = func(input string) { finish(textInputResult{text: input, cancelled: false}) }
		g.OnTextInputCancel = func() { finish(textInputResult{text: "", cancelled: true}) }
	})
}

func (g *Game) StartHotkeyInput(unitNodeUUID, optionName string, callback func(string)) {
	g.InputMode = ModeHotkeyInput
	g.HotkeyInputTargetUnitNodeUUID = unitNodeUUID
	g.HotkeyInputTargetOptionName = optionName
	g.HotkeyInputCurrent = ""
	g.OnHotkeyInputDone = callback
	g.OnHotkeyInputCancel = nil
	g.CursorTick = 0
}

func (g *Game) AwaitHotkeyInput(unitNodeUUID, optionName string) (string, bool) {
	return g.awaitInput(ModeHotkeyInput, func(resultChan chan textInputResult) {
		g.HotkeyInputTargetUnitNodeUUID = unitNodeUUID
		g.HotkeyInputTargetOptionName = optionName
		g.HotkeyInputCurrent = ""
		g.CursorTick = 0

		finish := func(res textInputResult) {
			g.OnHotkeyInputDone = nil
			g.OnHotkeyInputCancel = nil
			g.InputMode = ModeGame
			resultChan <- res
			close(resultChan)
		}

		g.OnHotkeyInputDone = func(hotkey string) { finish(textInputResult{text: hotkey, cancelled: false}) }
		g.OnHotkeyInputCancel = func() { finish(textInputResult{text: "", cancelled: true}) }
	})
}

func (g *Game) awaitInput(mode InputMode, setupFunc func(chan textInputResult)) (string, bool) {
	resultChan := make(chan textInputResult, 1)

	g.InputMode = mode
	setupFunc(resultChan)

	res := <-resultChan
	return res.text, res.cancelled
}


