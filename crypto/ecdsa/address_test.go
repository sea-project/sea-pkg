package ecdsa

import (
	"testing"
)

func TestAddress_ToICAP(t *testing.T) {

	address := HexToAddress("0x73da1a18ed4c58223fb8c2a54d9833df5329e6bf")
	icap := address.ToICAP("SEA", "0001")
	t.Log(icap)
	addr, err := ConvertICAPToAddress("SEA930001DJ6IDN541MIVGK94PFYB87PITDEKB0F", "SEA", "0001")
	if err != nil {
		t.Fatal("解析错误：", err)
	}
	t.Log(addr.Hex())
}
