package imgcompress

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/KarpelesLab/gowebp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Options 圖片壓縮與轉碼選項
type Options struct {
	Quality  int // 1-100 (預設 80)
	MaxWidth int // 最大寬度，超過則等比例縮放（預設 1920，0 表示不縮放）
}

// CompressToWebP 將磁盤上的圖片（JPG/PNG/BMP 等）在伺服器端壓縮並轉為 WebP 格式
// 成功時返回新檔案的路徑（.webp），並自動刪除原始較大的原圖檔案；
// 失敗時記錄警告並返回原圖路徑，確保業務流程零中斷。
func CompressToWebP(srcPath string, opts Options) (string, error) {
	ext := strings.ToLower(filepath.Ext(srcPath))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".bmp":
		// 支援轉碼的格式
	default:
		// 其他格式（GIF 動圖、SVG、PDF、已是 WebP/AVIF）不轉碼
		return srcPath, nil
	}

	if opts.Quality <= 0 || opts.Quality > 100 {
		opts.Quality = 80
	}
	if opts.MaxWidth <= 0 {
		opts.MaxWidth = 1920
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return srcPath, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return srcPath, err
	}

	// 檢查尺寸：若超出 MaxWidth 則進行高品質等比例縮放
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if opts.MaxWidth > 0 && w > opts.MaxWidth {
		newW := opts.MaxWidth
		newH := int(float64(h) * float64(newW) / float64(w))
		if newH > 0 {
			dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
			draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
			img = dst
		}
	}

	// 產生新的 .webp 檔案路徑
	dir := filepath.Dir(srcPath)
	baseName := strings.TrimSuffix(filepath.Base(srcPath), ext)
	outPath := filepath.Join(dir, baseName+".webp")

	outFile, err := os.Create(outPath)
	if err != nil {
		return srcPath, err
	}

	encodeErr := gowebp.Encode(outFile, img, &gowebp.Options{
		Lossy:   true,
		Quality: float32(opts.Quality),
		Method:  4,
	})
	outFile.Close()

	if encodeErr != nil {
		_ = os.Remove(outPath) // 清理殘留未完成檔案
		return srcPath, encodeErr
	}

	// 檢查壓縮結果大小：如果 WebP 比原圖更小，則保留 WebP 並刪除原圖
	fiOrig, errOrig := os.Stat(srcPath)
	fiNew, errNew := os.Stat(outPath)
	if errOrig == nil && errNew == nil && fiNew.Size() < fiOrig.Size() {
		_ = f.Close() // 確保檔案句柄關閉後再刪除
		_ = os.Remove(srcPath)
		slog.Info("伺服器端 WebP 壓縮完成", "src", srcPath, "dst", outPath,
			"origSize", fiOrig.Size(), "newSize", fiNew.Size(),
			"ratio", fmt.Sprintf("%.1f%%", float64(fiNew.Size())/float64(fiOrig.Size())*100))
		return outPath, nil
	}

	// 特殊情況：如果轉為 WebP 反而變大（少數極小圖），保留原圖並移除 WebP
	_ = os.Remove(outPath)
	return srcPath, nil
}
