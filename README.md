# pomodoro-tui

ターミナルで動くポモドーロタイマー。[Bubble Tea](https://github.com/charmbracelet/bubbletea) で作っています。

## 機能

- 作業時間 / 短い休憩 / 長い休憩を自動でループ
- 進捗バー表示
- 一時停止・リセット・スキップ
- セッション履歴を SQLite (`~/.pomodoro-tui/history.db`) に保存し、TUI 上で確認可能
- タスクの追加・完了切替・削除（同じ DB に保存され、次回起動時も保持されます）

## インストール

```bash
go install github.com/uikamu/pomodoro-tui@latest
```

## 使い方

```bash
pomodoro-tui
```

オプションで時間をカスタマイズできます。

```bash
pomodoro-tui --work 50m --break 10m --long-break 30m --cycles 3
```

## キー操作

| キー    | 動作               |
|---------|--------------------|
| space   | 一時停止 / 再開（タイマー画面） |
| n       | 現在のフェーズをスキップ |
| r       | 現在のフェーズをリセット |
| h       | 履歴表示の切り替え   |
| t       | タスク一覧の切り替え |
| q       | 終了               |

タスク一覧画面では `a` で追加、`↑/↓` または `j/k` で移動、`space`/`x` で完了切替、`d` で削除できます。

## 開発

```bash
go build ./...
go run .
```
