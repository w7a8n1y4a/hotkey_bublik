package game

import (
	"fmt"
	"math"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/hajimehoshi/ebiten/v2"

	"picker/internal/config"
	"picker/internal/hotkeys"
)

func (g *Game) Update() error {
	g.updateSpinner()

	switch g.InputMode {
	case ModeGame:
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			if !g.KeyDownMap[ebiten.KeyEscape] {
				g.KeyDownMap[ebiten.KeyEscape] = true
				return fmt.Errorf("game closed by user")
			}
		} else {
			g.KeyDownMap[ebiten.KeyEscape] = false
		}

		select {
		case res := <-g.refreshResultCh:
			g.refreshInProgress = false
			if res.err != nil {
				// Log error through PepeunitClient
				if g.PepeClient != nil {
					g.PepeClient.GetLogger().Error(fmt.Sprintf("Failed to refresh units: %v", res.err))
				}
			} else {
				g.Units = res.data
				g.resetSelection()
			}
			g.finishSpinnerOp()
		default:
		}

		select {
		case res := <-g.mqttResultCh:
			g.mqttInProgress = false
			if res.err != nil {
				// Log error through PepeunitClient
				if g.PepeClient != nil {
					g.PepeClient.GetLogger().Error(fmt.Sprintf("Failed to publish MQTT message: %v", res.err))
				}
				g.MQTTStatus = "MQTT: error: " + res.err.Error()
			} else {
				g.MQTTStatus = "MQTT: last publish OK"
			}
			g.finishSpinnerOp()
		default:
		}

		cfg := config.GetConfig()
		mouseX, mouseY := ebiten.CursorPosition()
		dx, dy := mouseX-cfg.PickerCenterX, mouseY-cfg.PickerCenterY
		angle := math.Atan2(-float64(dy), -float64(dx)) + math.Pi

		var currentLayerLength int
		switch g.ActiveLayer {
		case 0:
			currentLayerLength = len(g.Units.Units) + 1
		case 1:
			unitIdx := g.SelectedSegments[0] - 1
			if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
				currentLayerLength = len(g.Units.Units[unitIdx].UnitNodes)
			}
		case 2:
			unitIdx := g.SelectedSegments[0] - 1
			if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
				selectedUnit := g.Units.Units[unitIdx]
				if g.SelectedSegments[1] < len(selectedUnit.UnitNodes) {
					selectedNode := selectedUnit.UnitNodes[g.SelectedSegments[1]]
					stateData := g.StateData[selectedNode.UUID]
					currentLayerLength = len(stateData) + 1
				}
			}
		}

		if currentLayerLength > 0 {
			segmentAngle := 2 * math.Pi / float64(currentLayerLength)
			g.SelectedSegments[g.ActiveLayer] = int(angle/segmentAngle) % currentLayerLength
		}

		g.handleKey(ebiten.KeySpace, func() {
			switch g.ActiveLayer {
			case 0:
				settings := g.PepeClient.GetSettings()
				if g.PepeClient == nil {
					return
				}
				unitIdx := g.SelectedSegments[0] - 1
				if unitIdx < 0 || unitIdx >= len(g.Units.Units) {
					return
				}
				selectedUnit := g.Units.Units[unitIdx]
				unitURL := fmt.Sprintf("%s://%s/unit/%s", settings.PU_HTTP_TYPE, settings.PU_DOMAIN, selectedUnit.UUID)

				go func(url string) {
					cmd := exec.Command("xdg-open", url)
					_ = cmd.Start()
				}(unitURL)

			case 1:
				settings := g.PepeClient.GetSettings()
				if g.PepeClient == nil {
					return
				}
				unitIdx := g.SelectedSegments[0] - 1
				selectedNodeIdx := g.SelectedSegments[1]

				if unitIdx < 0 || unitIdx >= len(g.Units.Units) {
					return
				}
				selectedUnit := g.Units.Units[unitIdx]
				if selectedNodeIdx < 0 || selectedNodeIdx >= len(selectedUnit.UnitNodes) {
					return
				}
				selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]

				unitNodeURL := fmt.Sprintf("%s://%s/unit-node/%s", settings.PU_HTTP_TYPE, settings.PU_DOMAIN, selectedNode.UUID)

				go func(url string) {
					cmd := exec.Command("xdg-open", url)
					_ = cmd.Start()
				}(unitNodeURL)

			case 2:
				unitIdx := g.SelectedSegments[0] - 1
				selectedNodeIdx := g.SelectedSegments[1]
				if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
					selectedUnit := g.Units.Units[unitIdx]
					if selectedNodeIdx < len(selectedUnit.UnitNodes) {
						selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
						stateData := g.StateData[selectedNode.UUID]
						if g.SelectedSegments[2] > 0 && g.SelectedSegments[2]-1 < len(stateData) {
							optionName := stateData[g.SelectedSegments[2]-1][0]

							if ebiten.IsKeyPressed(ebiten.KeyControl) {
								if err := g.SetOptionHotkey(selectedNode.UUID, optionName, ""); err != nil {
									// Log error through PepeunitClient
									if g.PepeClient != nil {
										g.PepeClient.GetLogger().Error(fmt.Sprintf("Error clearing hotkey for command '%s' node '%s': %v", optionName, selectedNode.UUID, err))
									}
								} else {
									// Логирование очистки hotkey
									if g.PepeClient != nil {
										g.PepeClient.GetLogger().Info(fmt.Sprintf("Delete hotkey for command '%s'", optionName))
									}
								}
							} else {
								go func() {
									hotkey, cancelled := g.AwaitHotkeyInput(selectedNode.UUID, optionName)
									if cancelled {
										return
									}
									if err := g.SetOptionHotkey(selectedNode.UUID, optionName, hotkey); err != nil {
										// Log error through PepeunitClient
										if g.PepeClient != nil {
											g.PepeClient.GetLogger().Error(fmt.Sprintf("Error setting hotkey for command '%s' node '%s': %v", optionName, selectedNode.UUID, err))
									}
									} else {
										// Логирование установки hotkey
										if g.PepeClient != nil {
											g.PepeClient.GetLogger().Info(fmt.Sprintf("Set hotkey '%s' for command '%s'", hotkey, optionName))
										}
									}
								}()
							}
						}
					}
				}
			}
		})

		g.handleKey(ebiten.KeyDelete, func() {
			if g.ActiveLayer == 2 {
				unitIdx := g.SelectedSegments[0] - 1
				selectedNodeIdx := g.SelectedSegments[1]
				if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
					selectedUnit := g.Units.Units[unitIdx]
					if selectedNodeIdx < len(selectedUnit.UnitNodes) {
						selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
						stateData := g.StateData[selectedNode.UUID]
						if g.SelectedSegments[2] != 0 && g.SelectedSegments[2]-1 < len(stateData) {
							optionName := stateData[g.SelectedSegments[2]-1][0]
							err := g.RemoveOption(selectedNode.UUID, optionName)
							if err != nil {
								// Log error through PepeunitClient
								if g.PepeClient != nil {
									g.PepeClient.GetLogger().Error(fmt.Sprintf("Error removing option '%s' from node '%s': %v", optionName, selectedNode.UUID, err))
								}
							}
						}
					}
				}
			}
		})

		g.handleKey(ebiten.Key(ebiten.MouseButtonLeft), func() {
			if g.ActiveLayer == 0 {
				if g.SelectedSegments[0] == 0 {
					// Логирование обновления списка юнитов
					if g.PepeClient != nil {
						g.PepeClient.GetLogger().Info("Run update units list")
					}
					g.refreshUnits()
					return
				}

				if len(g.Units.Units) > 0 {
					g.ActiveLayer = 1
					g.SelectedSegments[1] = 0
				}
			} else if g.ActiveLayer == 1 {
				g.ActiveLayer = 2
				g.SelectedSegments[2] = 0
			} else if g.ActiveLayer == 2 {
				unitIdx := g.SelectedSegments[0] - 1
				selectedNodeIdx := g.SelectedSegments[1]
				if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
					selectedUnit := g.Units.Units[unitIdx]
					if selectedNodeIdx < len(selectedUnit.UnitNodes) {
						selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
						stateData := g.StateData[selectedNode.UUID]
						if g.SelectedSegments[2] == 0 {
							go func() {
								optionName, cancelled := g.AwaitTextInput(true)
								if cancelled {
									return
								}
								optionContent, cancelled := g.AwaitTextInput(false)
								if cancelled {
									return
								}
								if strings.TrimSpace(optionName) == "" {
									return
								}
								err := g.AddOption(selectedNode.UUID, optionName, optionContent)
								if err != nil {
									// Log error through PepeunitClient
									if g.PepeClient != nil {
										g.PepeClient.GetLogger().Error(fmt.Sprintf("Failed to add option '%s' to node '%s': %v", optionName, selectedNode.UUID, err))
									}
								}
							}()
						} else {
							if stateData != nil && g.SelectedSegments[2]-1 < len(stateData) {
								settings := g.PepeClient.GetSettings()
								topicName := settings.PU_DOMAIN + "/" + selectedNode.UUID + "/pepeunit"
								if g.PepeClient != nil && g.PepeClient.GetMQTTClient() != nil {
									payload := stateData[g.SelectedSegments[2]-1][1]
									// Логирование отправки команды по MQTT
									commandName := stateData[g.SelectedSegments[2]-1][0]
									g.PepeClient.GetLogger().Info(fmt.Sprintf("Send command '%s' to MQTT on topic '%s'", commandName, topicName))
									g.sendMQTT(topicName, payload)
								}
							}
						}
					}
				}
			}
		})

		g.handleKey(ebiten.Key(ebiten.MouseButtonRight), func() {
			if g.ActiveLayer > 0 {
				g.ActiveLayer--
				g.SelectedSegments[g.ActiveLayer] = 0
			}
		})

	case ModeTextInput:
		g.CursorTick++

		g.handleKey(ebiten.KeyEscape, func() {
			if g.OnTextInputCancel != nil {
				g.OnTextInputCancel()
			} else {
				g.InputMode = ModeGame
			}
		})

		for _, char := range ebiten.AppendInputChars(nil) {
			if char != '\n' && char != '\r' {
				g.TextInput += string(char)
			}
		}

		if ebiten.IsKeyPressed(ebiten.KeyBackspace) {
			g.BackspaceFrames++

			const initialDelay = 15
			const repeatInterval = 3

			if g.BackspaceFrames == 1 ||
				(g.BackspaceFrames > initialDelay && (g.BackspaceFrames-initialDelay)%repeatInterval == 0) {
				if len(g.TextInput) > 0 {
					g.TextInput = g.TextInput[:len(g.TextInput)-1]
				}
			}
		} else {
			g.BackspaceFrames = 0
		}

		g.handleKeyCombination(ebiten.KeyV, ebiten.KeyControl, func() {
			clipboardText, err := clipboard.ReadAll()
			if err == nil {
				g.TextInput += clipboardText
			}
		})

		g.handleKey(ebiten.KeyEnter, func() {
			if g.OnTextInputDone != nil {
				g.OnTextInputDone(g.TextInput)
			}
			g.InputMode = ModeGame

		})

	case ModeHotkeyInput:
		g.CursorTick++

		g.handleKey(ebiten.KeyEscape, func() {
			if g.OnHotkeyInputCancel != nil {
				g.OnHotkeyInputCancel()
			} else {
				g.InputMode = ModeGame
			}
		})

		currentHotkey := hotkeys.CaptureHotkeyFromEbiten()
		if currentHotkey != "" {
			g.HotkeyInputCurrent = currentHotkey
		}

		g.handleKey(ebiten.KeyEnter, func() {
			if g.OnHotkeyInputDone != nil {
				hotkeyToSave := g.HotkeyInputCurrent
				g.OnHotkeyInputDone(hotkeyToSave)
			}
			g.InputMode = ModeGame
		})

		if ebiten.IsKeyPressed(ebiten.KeyBackspace) || ebiten.IsKeyPressed(ebiten.KeyDelete) {
			g.HotkeyInputCurrent = ""
		}

	}

	return nil
}

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
