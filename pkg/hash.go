package pkg

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash/fnv"
)

func md5Hash(path string) string {
	h := md5.Sum([]byte(path))
	return hex.EncodeToString(h[:])
}

func fnvHash(path string) (string, error) {
	h := fnv.New64a()
	_, err := h.Write([]byte(path))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum64()), nil
}

func HashKey(path string) string {
	hash, err := fnvHash(path)
	if err != nil {
		return md5Hash(path)
	}
	return hash
}
