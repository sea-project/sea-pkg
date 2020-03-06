package md5

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

// Md5Sum 常用的md5摘要算法
func Md5Sum(input string) string {

	h := md5.New()
	h.Write([]byte(input))
	sum := h.Sum(nil)
	sumStr := hex.EncodeToString(sum)
	sumStr = strings.ToLower(sumStr)
	return sumStr
}
