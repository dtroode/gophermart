package auth

import (
	"context"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

type Argon2Id struct {
	salt    []byte
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func NewArgon2Id(
	salt []byte,
	time uint32,
	memory uint32,
	threads uint8,
	keyLen uint32,
) *Argon2Id {
	return &Argon2Id{
		salt:    salt,
		time:    time,
		memory:  memory,
		threads: threads,
		keyLen:  keyLen,
	}
}

func (a *Argon2Id) Hash(_ context.Context, password []byte) (string, error) {
	key := argon2.IDKey(password, a.salt, a.time, a.memory, a.threads, a.keyLen)

	hash := make([]byte, base64.StdEncoding.EncodedLen(len(key)))
	base64.StdEncoding.Encode(hash, key)
	return string(hash), nil
}
