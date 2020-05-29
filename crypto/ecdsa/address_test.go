package ecdsa

import (
	"strings"
	"testing"
)

func TestAddress_ToICAP(t *testing.T) {

	address := HexToAddress("0x73da1a18ed4c58223fb8c2a54d9833df5329e6bf")
	icap := address.ToICAP("sea", "0001")
	t.Log(icap)

	icap = strings.ToUpper("sea930001dj6idn541mivgk94pfyb87pitdekb0f")
	addr, err := ConvertICAPToAddress(icap)
	if err != nil {
		t.Fatal("解析错误：", err)
	}
	t.Log(addr.Hex())
}
