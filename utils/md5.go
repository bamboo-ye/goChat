package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// 小写
func Md5Encode(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	tempStr := h.Sum(nil)
	return hex.EncodeToString(tempStr)
}

// 大写
func MD5Encode(data string) string {
	return strings.ToUpper(Md5Encode(data))
}

// 加密
func MakePassword(plainpwd, salt string) string {
	password, err := bcrypt.GenerateFromPassword([]byte(plainpwd), bcrypt.DefaultCost)
	if err != nil {
		return Md5Encode(plainpwd + salt)
	}
	return string(password)
}

// ValidPassword keeps compatibility with legacy MD5 passwords.
func ValidPassword(plainpwd, salt string, password string) bool {
	if strings.HasPrefix(password, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(password), []byte(plainpwd)) == nil
	}
	return Md5Encode(plainpwd+salt) == password
}

func IsLegacyPassword(password string) bool {
	return !strings.HasPrefix(password, "$2")
}

func NewToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(data), nil
}
