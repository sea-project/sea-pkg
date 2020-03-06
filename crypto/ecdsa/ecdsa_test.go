package ecdsa

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	prv, pub := GenerateKey()
	t.Log("私钥原型：", prv)
	t.Log("公钥原型：", pub)
	t.Log("私钥转公钥：", prv.ToPubKey().ToHex())

	pubHex := pub.ToHex()
	t.Log("公钥哈希：", pubHex)

	pubs, err := HexToPubKey(pubHex)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("公钥还原：", pubs)

	prvHex := prv.ToHex()
	t.Log("私钥哈希：", prvHex)

	prvs, err := HexToPrvKey(prvHex)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("还原私钥：", prvs)
}

func TestStringToAddress(t *testing.T) {
	randm := rand.Reader
	randBytes := make([]byte, 64)
	randm.Read(randBytes)
	reader := bytes.NewReader(randBytes)
	prv, err := ecdsa.GenerateKey(S256(), reader)
	t.Log(prv)
	t.Log(prv.Public())
	return

	prvHex := "7b74e5aa26a3177e73bdd4b1c98135b02276637c255185c7f4cb09ef475ac88f"
	pubHex := "04ae39fae5f042d13fa3f3876c556cb974d6df44ac2195c6c7d9cb19d64c6e5a1c13662b98ba3da41d4803b80700ad794d07114f82029bb233375ee93480b065d6"

	prvKey, _ := HexToPrvKey(prvHex)
	t.Log(prvKey)

	pubKey, err := HexToPubKey(pubHex)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(pubKey)
}

func TestPrivateKey_Sign(t *testing.T) {
	prv, pub := GenerateKey()
	t.Log("原公钥：", pub)
	hash := []byte("1111111111111")

	// 第一种签名
	signature, err := prv.Sign(hash)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("第一种签名：", signature)

	isCheck := signature.Verify(hash, pub)
	t.Log(isCheck)

	// 第二种签名方式
	sign, err := SignCompact(S256(), prv, hash, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("第二种签名：", hex.EncodeToString(sign))
	t.Log("第二种签名：", len(sign))

	pubs, istrue, err := RecoverCompact(S256(), sign, hash)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("结果：", istrue)
	t.Log("导公钥：", pubs)
}

func TestSignature_IsEqual(t *testing.T) {

	d, _ := new(big.Int).SetString("28910270166077127595387180251412222235812151718408128539921370547058240323255", 0)
	fmt.Println("D:", d)
	_, pub := PrivKeyFromBytes(S256(), d.Bytes())
	fmt.Println("私钥：", pub.X)
	fmt.Println("私钥：", pub.Y)
	return

	sign := "30450221008628C8B5A7EEDC12A2E4EBCBC4F1E976D865AE6BD0C13A368886D32F198008AD0220793328612CFB6E9ADD37CC85D561E41BB083C5F7A25748F635C8743C8B5AB989"

	signature, _ := hex.DecodeString(sign)
	fmt.Println("signature: ", []byte(sign))

	pubs, istrue, err := RecoverCompact(S256(), signature, []byte("123"))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("result: ", istrue)
	fmt.Println("pubkey: ", pubs)

}
