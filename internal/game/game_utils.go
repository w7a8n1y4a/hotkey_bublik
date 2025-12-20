package game

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"strings"
	"time"

	"picker/internal/hotkeys"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func FetchUnits(client *pepeunit.PepeunitClient) (UnitsByNodesResponse, error) {
	if client == nil || client.GetRESTClient() == nil {
		return UnitsByNodesResponse{}, fmt.Errorf("REST client is not initialized")
	}
	if client.GetSchema() == nil {
		return UnitsByNodesResponse{}, fmt.Errorf("schema is not initialized")
	}

	outputTopics := client.GetSchema().GetOutputTopic()
	topicURLs, ok := outputTopics["output_units_nodes/pepeunit"]
	if !ok || len(topicURLs) == 0 {
		return UnitsByNodesResponse{}, nil
	}
	topicURL := topicURLs[0]

	if _, err := url.Parse(topicURL); err != nil {
		return UnitsByNodesResponse{}, nil
	}

	ctx := context.Background()

	rawNodes, err := client.GetRESTClient().GetInputByOutput(ctx, topicURL, 100, 0)
	if err != nil {
		return UnitsByNodesResponse{}, err
	}
	nodesBytes, err := json.Marshal(rawNodes)
	if err != nil {
		return UnitsByNodesResponse{}, err
	}

	var unitNodesResp UnitNodesResponse
	if err := json.Unmarshal(nodesBytes, &unitNodesResp); err != nil {
		return UnitsByNodesResponse{}, err
	}

	if unitNodesResp.Count == 0 || len(unitNodesResp.UnitNodes) == 0 {
		return UnitsByNodesResponse{}, nil
	}

	unitNodeUUIDs := make([]string, 0, len(unitNodesResp.UnitNodes))
	for _, item := range unitNodesResp.UnitNodes {
		unitNodeUUIDs = append(unitNodeUUIDs, item.UUID)
	}

	rawUnits, err := client.GetRESTClient().GetUnitsByNodes(ctx, unitNodeUUIDs, 100, 0)
	if err != nil {
		return UnitsByNodesResponse{}, err
	}
	unitsBytes, err := json.Marshal(rawUnits)
	if err != nil {
		return UnitsByNodesResponse{}, err
	}

	var unitsResp UnitsByNodesResponse
	if err := json.Unmarshal(unitsBytes, &unitsResp); err != nil {
		return UnitsByNodesResponse{}, err
	}

	return unitsResp, nil
}

func (g *Game) saveStateRemote() error {
	if g.PepeClient == nil || g.PepeClient.GetRESTClient() == nil {
		return nil
	}

	ctx := context.Background()
	payload, err := json.Marshal(g.StateData)
	if err != nil {
		return err
	}

	payloadStr := string(payload)

	err = g.PepeClient.SetStateStorage(ctx, payloadStr)
	return err
}

func (g *Game) AddOption(unitNodeUUID, optionName, optionValue string) error {
	if _, ok := g.StateData[unitNodeUUID]; !ok {
		g.StateData[unitNodeUUID] = [][]string{}
	}

	for i, pair := range g.StateData[unitNodeUUID] {
		if len(pair) > 0 && pair[0] == optionName {
			if len(pair) == 1 {
				g.StateData[unitNodeUUID][i] = append(pair, optionValue)
			} else {
				g.StateData[unitNodeUUID][i][1] = optionValue
			}
			return g.saveStateRemote()
		}
	}

	g.StateData[unitNodeUUID] = append(g.StateData[unitNodeUUID], []string{optionName, optionValue})
	return g.saveStateRemote()
}

func (g *Game) RemoveOption(unitNodeUUID, optionName string) error {
	items, ok := g.StateData[unitNodeUUID]
	if !ok {
		return nil
	}
	filtered := make([][]string, 0, len(items))
	for _, pair := range items {
		if pair[0] != optionName {
			filtered = append(filtered, pair)
		}
	}
	g.StateData[unitNodeUUID] = filtered
	return g.saveStateRemote()
}

func (g *Game) SetOptionHotkey(unitNodeUUID, optionName, hotkey string) error {
	if hotkey == "" {
		items, ok := g.StateData[unitNodeUUID]
		if !ok {
			return fmt.Errorf("unit node %s not found in state", unitNodeUUID)
		}

		for i, pair := range items {
			if len(pair) > 0 && pair[0] == optionName {
				if len(pair) >= 3 {
					g.StateData[unitNodeUUID][i][2] = ""
				}
				return g.saveStateRemote()
			}
		}
		return fmt.Errorf("option %s not found for unit node %s", optionName, unitNodeUUID)
	}

	if err := hotkeys.ValidateHotkey(hotkey); err != nil {
		return fmt.Errorf("invalid hotkey: %w", err)
	}

	for nodeUUID, items := range g.StateData {
		for i, pair := range items {
			if len(pair) >= 3 && pair[2] == hotkey {
				g.StateData[nodeUUID][i][2] = ""
			}
		}
	}

	items, ok := g.StateData[unitNodeUUID]
	if !ok {
		return fmt.Errorf("unit node %s not found in state", unitNodeUUID)
	}

	for i, pair := range items {
		if len(pair) > 0 && pair[0] == optionName {
			switch len(pair) {
			case 1:
				g.StateData[unitNodeUUID][i] = []string{pair[0], "", hotkey}
			case 2:
				g.StateData[unitNodeUUID][i] = append(pair, hotkey)
			default:
				g.StateData[unitNodeUUID][i][2] = hotkey
			}
			return g.saveStateRemote()
		}
	}

	return fmt.Errorf("option %s not found for unit node %s", optionName, unitNodeUUID)
}

func (g *Game) resetSelection() {
	if len(g.SelectedSegments) >= 1 {
		g.SelectedSegments[0] = 0
	}
	if len(g.SelectedSegments) >= 2 {
		g.SelectedSegments[1] = 0
	}
	if len(g.SelectedSegments) >= 3 {
		g.SelectedSegments[2] = 0
	}
	g.ActiveLayer = 0
}

func (g *Game) readLogEntries() []string {
	if time.Since(g.lastLogUpdateTime) < time.Second {
		return g.lastLogEntries
	}

	file, err := os.Open("log.json")
	if err != nil {
		return g.lastLogEntries
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return g.lastLogEntries
	}

	for _, line := range lines {
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	start := len(entries) - 8
	if start < 0 {
		start = 0
	}
	lastEntries := entries[start:]

	var formatted []string
	for _, entry := range lastEntries {
		parsedTime, err := time.Parse(time.RFC3339, entry.CreateDatetime)
		if err != nil {
			formatted = append(formatted, fmt.Sprintf("%s - %s - %s", entry.CreateDatetime[:19], entry.Level, entry.Text))
		} else {
			formatted = append(formatted, fmt.Sprintf("%s - %s - %s", parsedTime.Format("2006-01-02 15:04:05"), entry.Level, entry.Text))
		}
	}

	g.lastLogEntries = formatted
	g.lastLogUpdateTime = time.Now()
	return formatted
}

func (g *Game) startSpinnerOp() {
	now := time.Now()
	if !g.spinnerActive {
		g.spinnerActive = true
		g.spinnerAngle = 0
		g.spinnerStart = now
		g.spinnerLastUpdate = now
	}
	g.spinnerOpsInFlight++
}

func (g *Game) finishSpinnerOp() {
	if g.spinnerOpsInFlight > 0 {
		g.spinnerOpsInFlight--
	}
}

func (g *Game) updateSpinner() {
	if !g.spinnerActive {
		return
	}

	now := time.Now()

	dt := now.Sub(g.spinnerLastUpdate).Seconds()
	if dt < 0 {
		dt = 0
	}
	g.spinnerLastUpdate = now
	g.spinnerAngle += 2 * math.Pi * dt

	if g.spinnerOpsInFlight == 0 && now.Sub(g.spinnerStart) >= g.spinnerMinDuration {
		g.spinnerActive = false
	}
}
