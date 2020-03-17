// +build !appengine

package exec

import (
	"encoding/binary"

	"github.com/sea-project/sea-pkg/wagon/exec/internal/compile"
)

func init() {
	supportedNativeArchs = append(supportedNativeArchs, nativeArch{
		Arch: "amd64",
		OS:   "linux",
		make: makeAMD64NativeBackend,
	})
}

func makeAMD64NativeBackend(endianness binary.ByteOrder) *nativeCompiler {
	be := &compile.AMD64Backend{EmitBoundsChecks: debugStackDepth}
	return &nativeCompiler{
		Builder:   be,
		Scanner:   be.Scanner(),
		allocator: &compile.MMapAllocator{},
	}
}
