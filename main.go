package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/uikamu/pomodoro-tui/internal/config"
	"github.com/uikamu/pomodoro-tui/internal/storage"
	"github.com/uikamu/pomodoro-tui/internal/ui"
)

func main() {
	cfg := config.Parse()

	store, err := storage.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "履歴データベースを開けませんでした:", err)
		store = nil
	}
	if store != nil {
		defer store.Close()
	}

	p := tea.NewProgram(ui.New(cfg, store))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "エラー:", err)
		os.Exit(1)
	}
}
