package utils

import "github.com/jmcvetta/randutil"

const (
	Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func GetRandString(length int) (string, error) {
	return randutil.String(length, Alphabet)
}

func String(s string) *string { return &s }
