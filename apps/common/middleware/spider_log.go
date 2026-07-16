package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/core/db"
)

// === SpiderLog Worker Pool ===
// 對標 Swoole 6 的連線池 + 批量操作理念
// 解決 C3：每次蜘蛛訪問建立 goroutine 直接寫 DB，導致 goroutine 無上限 + 寫鎖競爭

const (
	spiderLogQueueSize = 1000  // channel 緩衝容量
	spiderLogWorkers   = 3     // 固定 worker 數量
	spiderLogBatchSize = 50    // 批量插入條數
	spiderLogBatchTimeout = 5 * time.Second // 批量插入超時
)

// spiderLogEntry 蜘蛛日誌條目
type spiderLogEntry struct {
	spider string
	url    string
	ip     string
}

var (
	spiderLogCh   chan spiderLogEntry
	spiderLogWg   sync.WaitGroup
	spiderLogCtx  context.Context
	spiderLogCancel context.CancelFunc
)

// StartSpiderLogWorkers 啟動蜘蛛日誌 worker pool
// 應在 main() 中伺服器啟動前呼叫
func StartSpiderLogWorkers() {
	spiderLogCh = make(chan spiderLogEntry, spiderLogQueueSize)
	spiderLogCtx, spiderLogCancel = context.WithCancel(context.Background())

	for i := 0; i < spiderLogWorkers; i++ {
		spiderLogWg.Add(1)
		go spiderLogWorker(i)
	}
	slog.Info("SpiderLog worker pool 已啟動", "workers", spiderLogWorkers, "queue_size", spiderLogQueueSize)
}

// StopSpiderLogWorkers 優雅關閉 spider log worker pool
// 關閉 channel 並等待所有 worker 處理完剩餘日誌
func StopSpiderLogWorkers() {
	if spiderLogCancel != nil {
		spiderLogCancel()
	}
	if spiderLogCh != nil {
		close(spiderLogCh)
	}
	spiderLogWg.Wait()
	slog.Info("SpiderLog worker pool 已關閉")
}

// spiderLogWorker 單個 worker：從 channel 讀取日誌，批量積累後寫入 DB
func spiderLogWorker(id int) {
	defer spiderLogWg.Done()

	batch := make([]spiderLogEntry, 0, spiderLogBatchSize)
	timer := time.NewTimer(spiderLogBatchTimeout)
	defer timer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		batchInsertSpiderLogs(batch)
		batch = batch[:0] // 清空但保留底層陣列
	}

	for {
		select {
		case entry, ok := <-spiderLogCh:
			if !ok {
				// channel 已關閉，刷入剩餘日誌後退出
				flush()
				return
			}
			batch = append(batch, entry)
			if len(batch) >= spiderLogBatchSize {
				flush()
				// 重置 timer
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(spiderLogBatchTimeout)
			}

		case <-timer.C:
			flush()
			timer.Reset(spiderLogBatchTimeout)

		case <-spiderLogCtx.Done():
			flush()
			// 排空 channel 中的剩餘條目
			for {
				select {
				case entry, ok := <-spiderLogCh:
					if !ok {
						return
					}
					batch = append(batch, entry)
					if len(batch) >= spiderLogBatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

// batchInsertSpiderLogs 批量插入蜘蛛日誌到 ay_syslog
// 使用單一事務批量插入，減少 SQLite 寫鎖競爭
func batchInsertSpiderLogs(entries []spiderLogEntry) {
	if len(entries) == 0 || db.DB == nil {
		return
	}

	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")
	logs := make([]model.Syslog, len(entries))
	for i, e := range entries {
		logs[i] = model.Syslog{
			Level:      "spider",
			Event:      fmt.Sprintf("%s 爬行 %s", e.spider, e.url),
			UserIP:     e.ip,
			UserOS:     "Spider",
			UserBs:     e.spider,
			CreateUser: e.spider,
			CreateTime: nowStr,
			Username:   e.spider,
			URL:        e.url,
			Content:    fmt.Sprintf("%s 爬行 %s", e.spider, e.url),
			IP:         e.ip,
			LogTime:    now,
		}
	}

	// 使用事務批量插入，失敗時記錄但不影響服務
	if err := db.DB.CreateInBatches(&logs, len(logs)).Error; err != nil {
		slog.Warn("SpiderLog 批量寫入失敗", "count", len(logs), "error", err)
	}
}

// enqueueSpiderLog 將蜘蛛日誌加入佇列（非阻塞，佇列滿時丟棄）
func enqueueSpiderLog(spider, url, ip string) {
	if spiderLogCh == nil {
		return
	}
	select {
	case spiderLogCh <- spiderLogEntry{spider: spider, url: url, ip: ip}:
	default:
		// 佇列已滿，丟棄此條日誌（避免阻塞請求）
		slog.Warn("SpiderLog 佇列已滿，丟棄日誌", "spider", spider, "url", url)
	}
}
