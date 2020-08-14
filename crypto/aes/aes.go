package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"github.com/sea-project/sea-pkg/crypto/sha3"
)

const BlockSise = aes.BlockSize

// PKCS5Padding PKCS5补位
func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// PKCS5UnPadding PKCS5不补位
func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	if length < unpadding {
		return nil
	}
	return origData[:(length - unpadding)]
}

// AesEncrypt aes算法加密
func AesEncrypt(origData, key []byte) ([]byte, error) {

	defer func() {
		if e := recover(); e != nil {
			fmt.Println("AesEncrypt error,", e)
		}
	}()

	if len(key) == 0 {
		return nil, errors.New("The password cannot be empty")
	}

	key16 := make([]byte, 16)
	copy(key16, key)

	block, err := aes.NewCipher(key16)
	if err != nil {
		return nil, err
	}

	hash := sha3.Keccak256(origData)
	blockSize := block.BlockSize()
	origData = PKCS5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key16[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)

	hash = append(hash, crypted...)
	return hash, nil
}

// AesDecrypt aes算法解密
func AesDecrypt(crypted, key []byte) ([]byte, error) {

	defer func() {
		if e := recover(); e != nil {
			fmt.Println("AesDecrypt error,", e)
		}
	}()

	key16 := make([]byte, 16)
	copy(key16, key)

	hash := crypted[:32]
	crypted = crypted[32:]

	block, err := aes.NewCipher(key16)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key16[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS5UnPadding(origData)
	if bytes.Equal(hash, sha3.Keccak256(origData)) {
		return origData, nil
	}
	return nil, errors.New("wrong password")
}

// AesCTRXOR aes异或加密
func AesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}
