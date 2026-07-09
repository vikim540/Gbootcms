package common

import (
	"log/slog"
	"os"
)

// Logger 全域結構化日誌器（基於 Go 1.21+ log/slog 標準庫）
// 替代散落的 log.Printf，支援級別分離和結構化輸出
var Logger *slog.Logger

func init() {
	Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// SetLogLevel 動態調整日誌級別（開發時可設為 Debug）
func SetLogLevel(level slog.Level) {
	Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(Logger)
}
