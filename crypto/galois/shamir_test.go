package galois

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestSplit(t *testing.T) {
	split, err := Split([]byte("0ddb327ad1059662da1f02f1b8521bf0f69cf5cecc09a4d8fc7f928fc9726818"), 4, 2)
	if err != nil {
		panic(err)
	}
	for i, v := range split {
		fmt.Println(i, hex.EncodeToString(v))
	}
	bytes, err := Combine([][]byte{split[0], split[1]})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(bytes))
}
