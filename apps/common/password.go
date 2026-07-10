package common

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// SecureRandomInt 使用密碼學安全隨機數生成 [0, max) 範圍內的整數
func SecureRandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	b := make([]byte, 8)
	rand.Read(b)
	n := binary.BigEndian.Uint64(b)
	return int(n % uint64(max))
}

// HashPassword 使用 bcrypt 雜湊密碼（成本因子 10，約 60ms）
// 返回的雜湊以 "$2a$" 開頭，可與舊版雙 MD5 區分
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword 校驗密碼，支援 bcrypt 和舊版雙 MD5 向後兼容
// 1. 若資料庫密碼以 "$2" 開頭 → bcrypt 校驗
// 2. 否則 → 雙 MD5 校驗（向後兼容舊資料）
// 返回 (是否匹配, 是否需要升級)
func VerifyPassword(plainPassword, dbPassword string) (matched bool, needUpgrade bool) {
	// bcrypt 雜湊以 $2a$/$2b$/$2y$ 開頭
	if strings.HasPrefix(dbPassword, "$2") {
		err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(plainPassword))
		return err == nil, false
	}

	// 舊版雙 MD5 向後兼容
	firstMd5 := fmt.Sprintf("%x", md5.Sum([]byte(plainPassword)))
	doubleMd5 := fmt.Sprintf("%x", md5.Sum([]byte(firstMd5)))
	if doubleMd5 == dbPassword {
		return true, true // 匹配但需要升級
	}

	return false, false
}

// DoubleMD5 雙 MD5 雜湊（保留向後兼容用）
func DoubleMD5(password string) string {
	firstMd5 := fmt.Sprintf("%x", md5.Sum([]byte(password)))
	return fmt.Sprintf("%x", md5.Sum([]byte(firstMd5)))
}
