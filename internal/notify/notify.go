package notify

import (
	"os/exec"
	"runtime"
	"strings"
)

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// Send shows a desktop notification. On unsupported platforms it is a no-op.
func Send(title, message string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	script := `display notification "` + escape(message) + `" with title "` + escape(title) + `" sound name "Glass"`
	return exec.Command("osascript", "-e", script).Run()
}
