package exec

import "errors"

// ErrUnreachable is the error value used while trapping the VM when
// an unreachable operator is reached during execution.
var ErrUnreachable = errors.New("exec: reached unreachable")

func (vm *VM) unreachable() {
	panic(ErrUnreachable)
}

func (vm *VM) nop() {}
