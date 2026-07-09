package watermark

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gbootcms/apps/admin/model"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// 水印位置常量（對齊 PHP watermark_position 配置）
const (
	PosTopLeft     = 1 // 左上角
	PosTopRight    = 2 // 右上角
	PosBottomLeft  = 3 // 左下角
	PosBottomRight = 4 // 右下角（預設）
	PosCenter      = 5 // 居中
)

// ApplyWatermark 對圖片應用水印（對齊 PHP watermark_img 函數）
// 讀取 watermark_open/watermark_text/watermark_pic 等配置
// 如果水印功能未開啟或圖片格式不支援，直接返回 nil
func ApplyWatermark(srcPath string) error {
	// 檢查水印開關
	if model.GetConfigValue("watermark_open", "0") != "1" {
		return nil
	}

	// 讀取水印配置
	watermarkText := model.GetConfigValue("watermark_text", "")
	watermarkPic := model.GetConfigValue("watermark_pic", "")

	// 如果沒有文字也沒有圖片水印，直接返回
	if watermarkText == "" && watermarkPic == "" {
		return nil
	}

	// 載入源圖片
	srcImg, format, err := loadImage(srcPath)
	if err != nil {
		return fmt.Errorf("載入圖片失敗: %w", err)
	}

	// 準備水印圖層
	var wmImg image.Image
	if watermarkPic != "" {
		// 圖片水印（對齊 PHP: $watermark_image = ROOT_PATH . $watermark_image）
		wmPath := watermarkPic
		if !filepath.IsAbs(wmPath) {
			wmPath = filepath.Join(".", wmPath)
		}
		wmImg, _, err = loadImage(wmPath)
		if err != nil {
			return fmt.Errorf("載入水印圖片失敗: %w", err)
		}
	} else {
		// 文字水印
		wmImg, err = createTextWatermark(watermarkText)
		if err != nil {
			return fmt.Errorf("創建文字水印失敗: %w", err)
		}
	}

	// 計算水印位置（對齊 PHP 位置邏輯）
	pos := parseInt(model.GetConfigValue("watermark_position", "4"), 4)
	bounds := srcImg.Bounds()
	wmBounds := wmImg.Bounds()
	wmW := wmBounds.Dx()
	wmH := wmBounds.Dy()

	// 自動縮放：如果源圖片太小（對齊 PHP: $width1 < $width2 * 3 || $height1 < $height2）
	if bounds.Dx() < wmW*3 || bounds.Dy() < wmH {
		scale := minFloat(float64(bounds.Dx())/3/float64(wmW), float64(bounds.Dy())/2/float64(wmH))
		if scale < 1 {
			newW := int(float64(wmW) * scale)
			newH := int(float64(wmH) * scale)
			wmImg = resizeImage(wmImg, newW, newH)
			wmBounds = wmImg.Bounds()
			wmW = newW
			wmH = newH
		}
	}

	x, y := calcPosition(pos, bounds.Dx(), bounds.Dy(), wmW, wmH)

	// 合成水印到源圖片
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, srcImg, image.Point{}, draw.Src)
	draw.Draw(rgba, image.Rect(x, y, x+wmW, y+wmH), wmImg, image.Point{}, draw.Over)

	// 保存圖片
	return saveImage(rgba, srcPath, format)
}

// createTextWatermark 創建文字水印圖層（對齊 PHP imagettftext 邏輯）
func createTextWatermark(text string) (image.Image, error) {
	if text == "" {
		text = "Gbootcms"
	}

	// 讀取字體配置
	fontSize := parseInt(model.GetConfigValue("watermark_text_size", "16"), 16)
	fontColorStr := model.GetConfigValue("watermark_text_color", "100,100,100")
	fontPath := model.GetConfigValue("watermark_text_font", "")

	if fontPath == "" {
		return nil, fmt.Errorf("未配置水印字體文件")
	}

	// 載入字體文件
	if !filepath.IsAbs(fontPath) {
		fontPath = filepath.Join(".", fontPath)
	}
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("載入字體文件失敗: %w", err)
	}

	// 解析字體
	parsedFont, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("解析字體失敗: %w", err)
	}

	// 創建字體面
	face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size:    float64(fontSize),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("創建字體面失敗: %w", err)
	}
	defer face.Close()

	// 解析顏色（對齊 PHP: $colors = explode(',', $watermark_text_color)）
	colors := parseColor(fontColorStr)

	// 測量文字寬高
	metrics := face.Metrics()
	textW := font.MeasureString(face, text).Ceil()
	textH := metrics.Height.Ceil()
	if textW <= 0 {
		textW = len(text) * (fontSize + 10)
	}
	if textH <= 0 {
		textH = fontSize + 10
	}

	// 創建透明背景的水印圖片（對齊 PHP: imagecreatetruecolor + imagecolortransparent）
	padding := 5
	wmW := textW + padding*2
	wmH := textH + padding*2
	rgba := image.NewRGBA(image.Rect(0, 0, wmW, wmH))

	// 繪製文字
	drawer := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(color.RGBA{R: colors[0], G: colors[1], B: colors[2], A: 255}),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(padding), Y: fixed.I(padding + textH - metrics.Descent.Ceil())},
	}
	drawer.DrawString(text)

	return rgba, nil
}

// calcPosition 計算水印坐標（對齊 PHP watermark_img 位置邏輯）
func calcPosition(pos, srcW, srcH, wmW, wmH int) (int, int) {
	const margin = 15
	switch pos {
	case PosTopLeft:
		return margin, margin
	case PosTopRight:
		return srcW - wmW - margin, 20
	case PosBottomLeft:
		return 20, srcH - wmH - margin
	case PosCenter:
		return (srcW - wmW) / 2, (srcH - wmH) / 2
	default: // PosBottomRight
		return srcW - wmW - margin, srcH - wmH - margin
	}
}

// loadImage 載入圖片（支援 gif/jpeg/png）
func loadImage(path string) (image.Image, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return nil, "", err
	}
	return img, format, nil
}

// saveImage 保存圖片（保持原格式）
func saveImage(img image.Image, path, format string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	switch format {
	case "jpeg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	case "png":
		return png.Encode(f, img)
	case "gif":
		return gif.Encode(f, img, &gif.Options{NumColors: 256})
	default:
		return png.Encode(f, img)
	}
}

// resizeImage 簡易縮放圖片（最近鄰插值）
func resizeImage(src image.Image, newW, newH int) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	scaleX := float64(bounds.Dx()) / float64(newW)
	scaleY := float64(bounds.Dy()) / float64(newH)
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			sx := int(float64(x) * scaleX)
			sy := int(float64(y) * scaleY)
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

// parseColor 解析 "R,G,B" 格式的顏色（對齊 PHP: explode(',', $watermark_text_color)）
func parseColor(s string) [3]uint8 {
	parts := strings.Split(s, ",")
	var c [3]uint8
	for i := 0; i < 3 && i < len(parts); i++ {
		v, _ := strconv.Atoi(strings.TrimSpace(parts[i]))
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		c[i] = uint8(v)
	}
	return c
}

// parseInt 安全解析整數
func parseInt(s string, defaultVal int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// minFloat 返回兩個浮點數的最小值
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
