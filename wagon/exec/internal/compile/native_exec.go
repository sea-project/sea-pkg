// +build !appengine

package compile

import "unsafe"

type asmBlock struct {
	mem unsafe.Pointer
}

func (b *asmBlock) Invoke(stack, locals, globals *[]uint64, mem *[]byte) JITExitSignal {
	return JITExitSignal(jitcall(unsafe.Pointer(&b.mem), stack, locals, globals, mem))
}
