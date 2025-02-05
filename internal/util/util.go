package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const (
	lowercaseAlphabet    = "abcdefghijklmnopqrstuvwxyz"
	uppercaseAlphabet    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numericAlphabet      = "0123456789"
	alphanumericAlphabet = lowercaseAlphabet + uppercaseAlphabet + numericAlphabet
)

// RandomID generates a random ID with the given length
// using the alphanumeric alphabet, no special characters or symbols
//
// Example:
//
// id := RandomID(20)
func RandomID(length int) string {
	return gonanoid.MustGenerate(alphanumericAlphabet, length)
}

// RandomStringPrefixed generates a random ID with the given length
// using the alphanumeric alphabet, no special characters or symbols
// and prefixes it with the given string
//
// Example:
//
// id := RandomStringPrefixed("user", 20)
func RandomStringPrefixed(prefix string, length int) string {
	return strings.Join([]string{prefix, RandomID(length)}, "")
}

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func Hash(s string) string {
	hash := sha256.New()
	hash.Write([]byte(s))

	return hex.EncodeToString(hash.Sum(nil))
}

func GetMachineHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func IsValidJSON(b []byte) bool {
	var js map[string]interface{}
	return json.Unmarshal(b, &js) == nil
}
