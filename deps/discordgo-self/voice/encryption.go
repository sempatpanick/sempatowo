package voice

import (
	"golang.org/x/crypto/nacl/secretbox"
)

func EncryptXSalsa20Poly1305(opus []byte, header []byte, secretKey *[32]byte) []byte {
	var nonce [24]byte
	copy(nonce[:12], header)

	return secretbox.Seal(header, opus, &nonce, secretKey)
}
