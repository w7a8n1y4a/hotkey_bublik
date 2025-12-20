package game

import (
	"bytes"
	"encoding/json"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

func DrawCenteredText(screen *ebiten.Image, face font.Face, textContent string, x, y, maxWidth, lineSpacing int, clr color.Color) {
	lines := wrapText(face, textContent, maxWidth)
	lineHeight := text.BoundString(face, "A").Dy() + lineSpacing
	totalHeight := len(lines) * lineHeight
	startY := y - totalHeight/2

	for i, line := range lines {
		lineWidth := text.BoundString(face, line).Dx()
		startX := x - lineWidth/2
		text.Draw(screen, line, face, startX, startY+(i*lineHeight), clr)
	}
}

func DrawLeftAlignedText(screen *ebiten.Image, face font.Face, textContent string, x, y, maxWidth, lineSpacing int, clr color.Color) {
	lineHeight := text.BoundString(face, "A").Dy() + lineSpacing
	currentY := y

	paragraphs := strings.Split(textContent, "\n")
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			currentY += lineHeight
			continue
		}

		lines := wrapText(face, p, maxWidth)
		for _, line := range lines {
			text.Draw(screen, line, face, x, currentY, clr)
			currentY += lineHeight
		}
	}
}

func wrapText(face font.Face, textContent string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{textContent}
	}

	runes := []rune(textContent)
	var lines []string
	var current []rune
	lastSpace := -1

	for i, r := range runes {
		current = append(current, r)
		if r == ' ' || r == '\t' {
			lastSpace = len(current) - 1
		}

		if text.BoundString(face, string(current)).Dx() > maxWidth {
			breakPos := len(current) - 1
			if lastSpace >= 0 {
				breakPos = lastSpace
			}

			lineRunes := current[:breakPos]
			line := strings.TrimSpace(string(lineRunes))
			if line != "" {
				lines = append(lines, line)
			}

			if breakPos < len(current) {
				current = current[breakPos:]
			} else {
				current = []rune{}
			}

			lastSpace = -1
			for j, rr := range current {
				if rr == ' ' || rr == '\t' {
					lastSpace = j
				}
			}
		}

		if i == len(runes)-1 {
			line := strings.TrimSpace(string(current))
			if line != "" {
				lines = append(lines, line)
			}
		}
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}

func tryPrettyJSON(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	if !json.Valid([]byte(raw)) {
		return "", false
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(raw), "", "    "); err != nil {
		return "", false
	}

	return buf.String(), true
}
