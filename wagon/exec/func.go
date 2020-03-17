package exec

import (
	"fmt"

	"github.com/sea-project/sea-pkg/wagon/exec/internal/compile"
)

type function interface {
	call(vm *VM, index int64)
	gas(vm *VM, index int64) (uint64, error)
}

type compiledFunction struct {
	code           []byte
	codeMeta       *compile.BytecodeMetadata
	branchTables   []*compile.BranchTable
	maxDepth       int  // maximum stack depth reached while executing the function body
	totalLocalVars int  // number of local variables used by the function
	args           int  // number of arguments the function accepts
	returns        bool // whether the function returns a value

	asm []asmBlock // CAREFUL
}

type asmBlock struct {
	// Compiled unit in native machine code.
	nativeUnit compile.NativeCodeUnit
	// where in the instruction stream to resume after native execution.
	resumePC uint
}

type goFunction struct {
	//val reflect.Value
	//typ reflect.Type

	//fn func(index int64, ops interface{}, args []uint64) (uint64, error)
}

func (gfn goFunction) gas(vm *VM, index int64) (uint64, error) {
	sig := vm.module.FunctionIndexSpace[index].Sig
	if vm.stackLen() < len(sig.ParamTypes) {
		vm.abort = true
		panic(fmt.Sprintf("stack_len(%d) < args_len(%d)", vm.stackLen(), len(sig.ParamTypes)))
	}
	args := make([]uint64, len(sig.ParamTypes))
	for i := 0; i < len(sig.ParamTypes); i++ {
		args[len(sig.ParamTypes)-1-i] = vm.backUint64(i)
	}
	vm.ops.Trace("host function gas", "index", index, "args", args)
	hostFn := vm.module.FunctionIndexSpace[index].Host
	ret, err := hostFn.Gas(index, vm.ops, args)

	return ret, err
}

func (gfn goFunction) call(vm *VM, index int64) {
	sig := vm.module.FunctionIndexSpace[index].Sig

	if vm.stackLen() < len(sig.ParamTypes) {
		vm.abort = true
		panic(fmt.Sprintf("stack_len(%d) < args_len(%d)", vm.stackLen(), len(sig.ParamTypes)))
	}
	args := make([]uint64, len(sig.ParamTypes))

	//for i := 0; i < len(sig.ParamTypes); i++ {
	//	args[i] = vm.popUint64()
	//}
	for i := len(sig.ParamTypes) - 1; i >= 0; i-- {
		args[i] = vm.popUint64()
	}

	vm.ops.Trace("host function call begin", "index", index, "args", args, "returns", len(sig.ReturnTypes))
	hostFn := vm.module.FunctionIndexSpace[index].Host
	ret, err := hostFn.Call(index, vm.ops, args)
	if err != nil {
		// TODO: error handling, terminate VM execution
		vm.abort = true
		// panic(fmt.Sprintf("goFunction call fail: %s", err))
		panic(err)
	}

	//tcExit terminate the program and need to return a value
	if len(sig.ReturnTypes) > 0 || vm.abort {
		vm.pushUint64(ret)
	}
	vm.ops.Trace("host function call end", "index", index, "ret", ret)
}

/*
func (fn goFunction) call(vm *VM, index int64) {
	// numIn = # of call inputs + vm, as the function expects
	// an additional *VM argument
	numIn := fn.typ.NumIn()
	args := make([]reflect.Value, numIn)
	proc := NewProcess(vm)

	// Pass proc as an argument. Check that the function indeed
	// expects a *Process argument.
	if reflect.ValueOf(proc).Kind() != fn.typ.In(0).Kind() {
		panic(fmt.Sprintf("exec: the first argument of a host function was %s, expected %s", fn.typ.In(0).Kind(), reflect.ValueOf(vm).Kind()))
	}
	args[0] = reflect.ValueOf(proc)

	for i := numIn - 1; i >= 1; i-- {
		val := reflect.New(fn.typ.In(i)).Elem()
		raw := vm.popUint64()
		kind := fn.typ.In(i).Kind()

		switch kind {
		case reflect.Float64, reflect.Float32:
			val.SetFloat(math.Float64frombits(raw))
		case reflect.Uint32, reflect.Uint64:
			val.SetUint(raw)
		case reflect.Int32, reflect.Int64:
			val.SetInt(int64(raw))
		default:
			panic(fmt.Sprintf("exec: args %d invalid kind=%v", i, kind))
		}

		args[i] = val
	}

	rtrns := fn.val.Call(args)
	for i, out := range rtrns {
		kind := out.Kind()
		switch kind {
		case reflect.Float64, reflect.Float32:
			vm.pushFloat64(out.Float())
		case reflect.Uint32, reflect.Uint64:
			vm.pushUint64(out.Uint())
		case reflect.Int32, reflect.Int64:
			vm.pushInt64(out.Int())
		default:
			panic(fmt.Sprintf("exec: return value %d invalid kind=%v", i, kind))
		}
	}
}
*/

func (compiled compiledFunction) gas(vm *VM, index int64) (uint64, error) {
	return 0, nil
}

func (compiled compiledFunction) call(vm *VM, index int64) {
	// Make space on the stack for all intermediate values and
	// a possible return value.
	newStack := make([]uint64, 0, compiled.maxDepth+1)
	locals := make([]uint64, compiled.totalLocalVars)

	for i := compiled.args - 1; i >= 0; i-- {
		locals[i] = vm.popUint64()
	}

	//save execution context
	prevCtxt := vm.ctx

	vm.ctx = context{
		stack:   newStack,
		locals:  locals,
		code:    compiled.code,
		asm:     compiled.asm,
		pc:      0,
		curFunc: index,
	}

	rtrn := vm.execCode(compiled)

	//restore execution context
	vm.ctx = prevCtxt

	if compiled.returns {
		vm.pushUint64(rtrn)
	}
}
