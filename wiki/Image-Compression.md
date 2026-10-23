# 原生圖片壓縮管道 (Image Compression)

Gbootcms 徹底廢除肥大且不可靠的前端 Canvas 偽壓縮，實作了伺服器端原生純 Go WebP 轉碼與雙三次重採樣縮放管道（`apps/common/imgcompress`）。

---

## 1. 設計優勢

- **零 CGO 依賴**：完全基於純 Go VP8 編碼器（`github.com/KarpelesLab/gowebp`），保持 `CGO_ENABLED=0`，支援任意平台交叉編譯。
- **全入口統一**：後台單圖上傳、多圖相簿、UEditor 富文本貼圖、RESTful API 均自動觸發。
- **大幅縮減體積**：實測 890KB PNG 轉為 WebP 僅需 130KB（減幅高達 85%），顯著降低伺服器頻寬與 CDN 成本。
- **高品質自適應縮放**：使用 `golang.org/x/image/draw.BiCubic` 雙三次內插演算法，保持邊緣銳利與文字清晰。

---

## 2. 伺服器端處理流程

```
客戶端上傳原始圖片 (JPG / PNG / GIF / BMP)
                  │
                  ▼
          儲存原始暫存檔案
                  │
                  ▼
       檢查 ay_config 配置開關:
       image_compress_enable == 1 ?
            ├── 是 ──► [imgcompress 處理管道]
            │               │
            │               ├─ 讀取圖片尺寸，超過 image_max_width 則雙三次平滑縮放
            │               ├─ 轉換色彩模型 (RGBA) 並呼叫 gowebp 編碼為 .webp
            │               └─ 刪除原檔案，替換路徑為新 .webp 檔案
            │
            └── 否 ──► 保持原始檔案
                  │
                  ▼
          應用浮水印 (可選)
                  │
                  ▼
          返回最終 URL 給前端
```

---

## 3. 核心代碼調用案例

在控制器或其他後台服務中，調用 `imgcompress` 僅需一行代碼：

```go
package main

import (
    "fmt"
    "log"
    "gbootcms/apps/common/imgcompress"
)

func ProcessUserImage(rawFilePath string) {
    // 呼叫伺服器端 WebP 轉碼管道
    newPath, err := imgcompress.CompressToWebP(rawFilePath, imgcompress.Options{
        Quality:  80,   // 壓縮品質 (1-100，推薦 80)
        MaxWidth: 1920, // 最大像素寬度（超過自動等比例縮小）
    })
    if err != nil {
        log.Printf("壓縮失敗，退回原檔案: %v", err)
        return
    }

    fmt.Printf("WebP 檔案已生成: %s\n", newPath)
}
```

---

## 4. 後台配置項說明

於後台「系統配置」->「基本設置」中動態控制：

| 配置鍵名 | 預設值 | 說明 |
| :--- | :--- | :--- |
| `image_compress_enable` | `1` | `1` 啟用伺服器端 WebP 壓縮；`0` 停用 |
| `image_quality` | `80` | WebP 壓縮質量（1~100） |
| `image_max_width` | `1920` | 最大寬度（像素），超過則等比下採樣 |
