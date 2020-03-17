package exec

const (
	GasQuickStep   uint64 = 2
	GasFastestStep uint64 = 3
	GasFastStep    uint64 = 5
	GasMidStep     uint64 = 8
	GasSlowStep    uint64 = 10
	GasExtStep     uint64 = 20

	GasReturn       uint64 = 0
	GasStop         uint64 = 0
	GasContractByte uint64 = 200
)

func constGasFunc(gas uint64) gasFunc {
	return func(vm *VM) (uint64, error) {
		return gas, nil
	}
}

func gasGrowMemory(vm *VM) (uint64, error) {
	n := vm.popInt32()
	vm.pushInt32(n)
	return uint64(n * 1000), nil
}

func gasCall(vm *VM) (uint64, error) {
	index := vm.prefetchUint32()
	return vm.funcs[index].gas(vm, int64(index))
}
