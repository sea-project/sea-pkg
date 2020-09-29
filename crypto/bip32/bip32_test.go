package bip32

import (
	"testing"
)

type test struct {
	Key []byte
}

func Test_Main(t *testing.T) {
	mkey, err := NewMasterKey([]byte("qiqi"))
	if err != nil {
		t.Log(err)
	}
	t.Log(JsonString(mkey))
	t.Log(mkey.String())
	t.Log(mkey.PublicKey())
	t.Log(PubKeyToAddr(mkey.PublicKey().Key))

}
