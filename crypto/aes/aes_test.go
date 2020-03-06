package aes

import (
	"crypto/aes"
	"crypto/rand"
	"github.com/sea-project/sea-pkg/crypto/ecdsa"
	"github.com/sea-project/sea-pkg/util/math"
	"golang.org/x/crypto/scrypt"
	"io"
	"testing"
)

func TestAesDecrypt(t *testing.T) {
	password := "111"
	str := "武小永"

	rows, err := AesEncrypt([]byte(str), []byte(password))
	if err != nil {
		t.Fatal("加密失败：", err)
	}
	t.Log(rows)

	bytes, err := AesDecrypt(rows, []byte("111"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(bytes))

}

func TestPKCS5Padding(t *testing.T) {

	test := "111"
	t.Log(len(test))
	a := make([]byte, 16)
	copy(a, []byte(test))
	t.Log(a)
	t.Log(len(a))
}

func BenchmarkAesEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AesEncrypt([]byte("04f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f2304f4ad7ce3e68dce929d066f8d37bdf6d55e4a5a520ddbccf8a206c75444d135c7bcd525e736492966500184101b77feb00f59589ddda085e7439364b286c39f23"), []byte("1111111111111111"))
	}
}

func TestAesCTRXOR(t *testing.T) {

	authArray := []byte("123456")
	salt := GetEntropyCSPRNG(32)
	scryptN := 2048
	scryptP := 6
	scryptR := 8
	scryptDKLen := 32
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		t.Fatal(err)
	}

	prv, _ := ecdsa.GenerateKey()

	t.Log(len(derivedKey))

	encryptKey := derivedKey[:32]
	keyBytes := math.PaddedBigBytes(prv.D, 32)

	iv := GetEntropyCSPRNG(aes.BlockSize) // 16
	key, err := AesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(len(key))
}

func GetEntropyCSPRNG(n int) []byte {
	mainBuff := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, mainBuff)
	if err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	return mainBuff
}
