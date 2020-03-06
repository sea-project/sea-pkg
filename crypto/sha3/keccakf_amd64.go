// +build amd64,!appengine,!gccgo

package sha3

// 该函数在keccakf_amd64.s中实现。

//go:noescape

func keccakF1600(state *[25]uint64)
