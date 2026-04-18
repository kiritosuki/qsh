package utils

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/mattn/go-tty"
)

const (
	TermMaxWidth        = 100
	TermSafeZonePadding = 10
)

func GetTermSafeMaxWidth() int {
	maxWidth := TermMaxWidth
	termWidth, err := getTermWidth()
	if err != nil || termWidth < maxWidth {
		maxWidth = termWidth - TermSafeZonePadding
	}
	return maxWidth
}

func getTermWidth() (width int, err error) {
	t, err := tty.Open()
	if err != nil {
		return 0, err
	}
	defer t.Close()
	width, _, err = t.Size()
	return width, err
}

func IsLikelyBillingError(s string) bool {
	return strings.Contains(s, "429 Too Many Requests")
}

func ExtractFirstCodeBlock(s string) (string, bool) {
	isOnlyCode := true
	if len(s) <= 3 {
		return "", false
	}
	start := strings.Index(s, "```")
	if start == -1 {
		return "", false
	}
	if start != 0 {
		isOnlyCode = false
	}
	fromStart := s[start:]
	content := strings.TrimPrefix(fromStart, "```")
	newLinePos := strings.Index(content, "\n")
	if newLinePos != -1 {
		if content[0:newLinePos] == strings.TrimSpace(content[0:newLinePos]) {
			content = content[newLinePos+1:]
		}
	}
	end := strings.Index(content, "```")
	if end < len(content)-3 {
		isOnlyCode = false
	}
	if end != -1 {
		content = content[:end]
	}
	if len(content) == 0 {
		return "", false
	}
	if content[len(content)-1] == '\n' {
		content = content[:len(content)-1]
	}
	return content, isOnlyCode
}

func StartsWithCodeBlock(s string) bool {
	if len(s) <= 3 {
		return strings.Repeat("`", len(s)) == s
	}
	return strings.HasPrefix(s, "```")
}

func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // For Linux or anything else
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}
