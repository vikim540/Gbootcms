package common

import (
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gbootcms/apps/admin/model"
)

// === 統一驗證碼模塊 ===
// 替代原先散落在 admin/IndexController 和 home/front 中的兩套獨立驗證碼實現。
// 所有場景（後台登錄、留言、自定義表單）共用同一存儲、同一生成函數、同一校驗函數。

// codeStore 統一驗證碼存儲（sessionID → 答案）
var codeStore = make(map[string]string)
var codeMu sync.Mutex

// GenerateCaptcha 生成數學驗證碼圖片並輸出 PNG
// 統一入口，後台 /admin/index/checkCode 和前台 /api/checkcode 共用
func GenerateCaptcha(c *gin.Context) {
	a := randInt(9) + 1
	b := randInt(9) + 1
	op := randInt(2)
	var expr string
	var answer int
	if op == 0 {
		expr = fmt.Sprintf("%d + %d = ?", a, b)
		answer = a + b
	} else {
		if a < b {
			a, b = b, a
		}
		expr = fmt.Sprintf("%d - %d = ?", a, b)
		answer = a - b
	}

	sessionID := getCaptchaSessionID(c)
	codeMu.Lock()
	codeStore[sessionID] = fmt.Sprintf("%d", answer)
	codeMu.Unlock()

	width := 200
	height := 70
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bgColor := color.RGBA{245, 250, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// 噪點
	for i := 0; i < 80; i++ {
		x := randInt(width)
		y := randInt(height)
		r := uint8(180 + randInt(60))
		g := uint8(180 + randInt(60))
		b2 := uint8(180 + randInt(60))
		img.Set(x, y, color.RGBA{r, g, b2, 255})
	}

	// 干擾線
	for i := 0; i < 4; i++ {
		x1 := randInt(width)
		y1 := randInt(height)
		x2 := randInt(width)
		y2 := randInt(height)
		lineColor := color.RGBA{uint8(100 + randInt(100)), uint8(100 + randInt(100)), uint8(100 + randInt(100)), 255}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}

	// 點陣字體繪製
	drawCaptchaText(img, expr, width, height)

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	png.Encode(c.Writer, img)
}

// VerifyCaptcha 統一驗證碼校驗
// configName 為配置項名稱（如 message_check_code、form_check_code、admin_check_code）
// defaultVal 為配置項不存在時的默認值
// 通過返回 true（含驗證碼已關閉的情況），失敗返回 false 並已寫入 JSON 響應
func VerifyCaptcha(c *gin.Context, configName, defaultVal string) bool {
	if model.GetConfigValue(configName, defaultVal) == "0" {
		return true // 驗證碼已關閉
	}
	checkcode := strings.ToLower(c.PostForm("checkcode"))
	if checkcode == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": NoticeCaptchaEmpty, "tourl": ""})
		return false
	}
	sessionID := getCaptchaSessionID(c)
	codeMu.Lock()
	saved, ok := codeStore[sessionID]
	delete(codeStore, sessionID) // 一次性消費
	codeMu.Unlock()
	if !ok || strings.ToLower(saved) != checkcode {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": NoticeCaptchaError, "tourl": ""})
		return false
	}
	return true
}

// getCaptchaSessionID 獲取或創建驗證碼 session ID（基於 PbootGo cookie）
func getCaptchaSessionID(c *gin.Context) string {
	cookie, err := c.Cookie("PbootGo")
	if err != nil || cookie == "" {
		cookie = generateCaptchaSessionID()
		SetSecureCookie(c, "PbootGo", cookie, 86400, "/")
	}
	return cookie
}

func generateCaptchaSessionID() string {
	b := make([]byte, 32)
	crand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func randInt(max int) int {
	if max <= 0 {
		return 0
	}
	return rand.Intn(max)
}

// drawLine 繪製干擾線（Bresenham 算法）
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	sy := 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy
	for {
		if x1 >= 0 && x1 < img.Bounds().Dx() && y1 >= 0 && y1 < img.Bounds().Dy() {
			img.Set(x1, y1, col)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// 點陣字體表（5x7 像素，支持數字和運算符）
var captchaFont = map[byte][][]bool{
	'0': {{false, true, true, true, false}, {true, false, false, false, true}, {true, false, false, true, true}, {true, false, true, false, true}, {true, true, false, false, true}, {true, false, false, false, true}, {false, true, true, true, false}},
	'1': {{false, false, true, false, false}, {false, true, true, false, false}, {true, false, true, false, false}, {false, false, true, false, false}, {false, false, true, false, false}, {false, false, true, false, false}, {true, true, true, true, true}},
	'2': {{false, true, true, true, false}, {true, false, false, false, true}, {false, false, false, false, true}, {false, false, true, true, false}, {false, true, false, false, false}, {true, false, false, false, false}, {true, true, true, true, true}},
	'3': {{false, true, true, true, false}, {true, false, false, false, true}, {false, false, false, false, true}, {false, false, true, true, false}, {false, false, false, false, true}, {true, false, false, false, true}, {false, true, true, true, false}},
	'4': {{false, false, false, true, false}, {false, false, true, true, false}, {false, true, false, true, false}, {true, false, false, true, false}, {true, true, true, true, true}, {false, false, false, true, false}, {false, false, false, true, false}},
	'5': {{true, true, true, true, true}, {true, false, false, false, false}, {true, true, true, true, false}, {false, false, false, false, true}, {false, false, false, false, true}, {true, false, false, false, true}, {false, true, true, true, false}},
	'6': {{false, true, true, true, false}, {true, false, false, false, false}, {true, false, false, false, false}, {true, true, true, true, false}, {true, false, false, false, true}, {true, false, false, false, true}, {false, true, true, true, false}},
	'7': {{true, true, true, true, true}, {false, false, false, false, true}, {false, false, false, true, false}, {false, false, true, false, false}, {false, false, true, false, false}, {false, false, true, false, false}, {false, false, true, false, false}},
	'8': {{false, true, true, true, false}, {true, false, false, false, true}, {true, false, false, false, true}, {false, true, true, true, false}, {true, false, false, false, true}, {true, false, false, false, true}, {false, true, true, true, false}},
	'9': {{false, true, true, true, false}, {true, false, false, false, true}, {true, false, false, false, true}, {false, true, true, true, true}, {false, false, false, false, true}, {false, false, false, false, true}, {false, true, true, true, false}},
	' ': {{false, false}, {false, false}, {false, false}, {false, false}, {false, false}, {false, false}, {false, false}},
	'+': {{false, false, false, false, false}, {false, false, true, false, false}, {false, false, true, false, false}, {true, true, true, true, true}, {false, false, true, false, false}, {false, false, true, false, false}, {false, false, false, false, false}},
	'-': {{false, false, false, false, false}, {false, false, false, false, false}, {false, false, false, false, false}, {true, true, true, true, true}, {false, false, false, false, false}, {false, false, false, false, false}, {false, false, false, false, false}},
	'=': {{false, false, false, false, false}, {true, true, true, true, true}, {false, false, false, false, false}, {false, false, false, false, false}, {true, true, true, true, true}, {false, false, false, false, false}, {false, false, false, false, false}},
	'?': {{false, true, true, true, false}, {true, false, false, false, true}, {false, false, false, false, true}, {false, false, true, true, false}, {false, false, true, false, false}, {false, false, false, false, false}, {false, false, true, false, false}},
}

// drawCaptchaText 在圖片上繪製驗證碼文字
func drawCaptchaText(img *image.RGBA, expr string, width, height int) {
	pixelSize := 6
	startX := 10
	startY := 8

	colors := []color.RGBA{
		{220, 50, 50, 255},
		{50, 120, 220, 255},
		{40, 160, 60, 255},
		{180, 100, 20, 255},
		{130, 40, 180, 255},
		{20, 150, 150, 255},
		{200, 60, 120, 255},
	}

	for ci, ch := range []byte(expr) {
		pattern, ok := captchaFont[ch]
		if !ok {
			continue
		}
		charColor := colors[ci%len(colors)]
		rotateOffset := randInt(5) - 2

		for py, row := range pattern {
			for px, on := range row {
				if on {
					for sx := 0; sx < pixelSize; sx++ {
						for sy := 0; sy < pixelSize; sy++ {
							px2 := startX + px*pixelSize + sx
							py2 := startY + (py+rotateOffset)*pixelSize + sy
							if px2 >= 0 && px2 < width && py2 >= 0 && py2 < height {
								img.Set(px2, py2, charColor)
							}
						}
					}
				}
			}
		}
		patWidth := 0
		for _, row := range pattern {
			if len(row) > patWidth {
				patWidth = len(row)
			}
		}
		startX += patWidth*pixelSize + 4
	}
}
