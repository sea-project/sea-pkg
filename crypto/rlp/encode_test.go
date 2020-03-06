package rlp

import (
	"testing"
)

type Block struct {
	// 父块的hash
	ParentHash string `json:"parentHash"`
	// 当前块的hash
	Hash string `json:"Hash"`
	// 区块的序号。Block的Number等于其父区块Number +1
	Number uint64 `json:"number"`
	// 区块开始时间
	Timestamp uint64 `json:"timestamp"`
	Data      string `json:"data"`
}

var newBlock = &Block{ParentHash: "123", Hash: "456", Number: 111, Timestamp: 222, Data: "asdfasdf"}

func Test_RLP(t *testing.T) {
	res, err := EncodeToBytes(newBlock)
	if err != nil {
		panic(err.Error())
	}

	t.Log(res)

	block := new(Block)
	DecodeBytes(res, block)
	// Decode(bytes.NewReader(res),block)
	t.Log(block)
}
