package bip39

import (
	"encoding/hex"
	"github.com/sea-project/sea-pkg/crypto/bip32"
	"testing"
)

type vector struct {
	entropy  string
	mnemonic string
	seed     string
}

func Test_Main(t *testing.T) {
	seed := NewSeed("的的的的的的的的的的的的", "123456")
	t.Log(seed)
	t.Log(hex.EncodeToString(seed))
	mkey, _ := bip32.NewMasterKey(seed)
	t.Log(bip32.JsonString(mkey))
	t.Log(mkey.String())
	t.Log(mkey.PublicKey())
	t.Log(bip32.PubKeyToAddr(mkey.PublicKey().Key))
}

func TestNewEntropy(t *testing.T) {
	entropy, _ := NewEntropy(128)
	t.Logf("entropy:%v", entropy)
	mnemonic, _ := NewMnemonic(entropy)
	t.Logf("mnemonic:%v", mnemonic)
	//seed := NewSeed(mnemonic, "123456")

}

func TestNewMnemonic(t *testing.T) {
	for _, vector := range testVectors() {
		entropy, err := hex.DecodeString(vector.entropy)
		if err != nil {
			t.Log(err)
		}
		t.Log(entropy)

		mnemonic, err := NewMnemonic(entropy)
		if err != nil {
			t.Log(err)
		}
		t.Log(mnemonic)

		_, err = NewSeedWithErrorChecking(mnemonic, "TREZOR")
		if err != nil {
			t.Log(err)
		}

		seed := NewSeed(mnemonic, "TREZOR")
		t.Log(vector.seed)
		t.Log(hex.EncodeToString(seed))
	}
}

func testVectors() []vector {
	return []vector{
		{
			entropy:  "00000000000000000000000000000000",
			mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
			seed:     "c55257c360c07c72029aebc1b53c05ed0362ada38ead3e3e9efa3708e53495531f09a6987599d18264c1e1c92f2cf141630c7a3c4ab7c81b2f001698e7463b04",
		},
		{
			entropy:  "7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f",
			mnemonic: "legal winner thank year wave sausage worth useful legal winner thank yellow",
			seed:     "2e8905819b8723fe2c1d161860e5ee1830318dbf49a83bd451cfb8440c28bd6fa457fe1296106559a3c80937a1c1069be3a3a5bd381ee6260e8d9739fce1f607",
		},
	}
}
