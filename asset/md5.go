// Package utils is retained for compatibility with the historical asset helper.
package utils

import coreutils "goChat/utils"

func Md5Encode(data string) string {
	return coreutils.Md5Encode(data)
}

func MD5Encode(data string) string {
	return coreutils.MD5Encode(data)
}

func MakePassword(plainPassword, salt string) string {
	return coreutils.MakePassword(plainPassword, salt)
}

func ValidPassword(plainPassword, salt, password string) bool {
	return coreutils.ValidPassword(plainPassword, salt, password)
}
