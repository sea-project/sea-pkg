package operators

import "github.com/sea-project/sea-pkg/wagon/wasm"

// These opcodes implement optimizations in wagon execution, and are invalid
// opcodes for any uses other than internal use. Expect them to change at any
// time.
// If these opcodes are ever used in future wasm instructions, feel free to
// reassign them to other free opcodes.
var (
	internalOpcodes = map[byte]bool{
		WagonNativeExec: true,
	}

	WagonNativeExec = newOp(0xfe, "wagon.nativeExec", []wasm.ValueType{wasm.ValueTypeI64}, noReturn)
)
