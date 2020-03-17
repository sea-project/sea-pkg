package exec

import (
	ops "github.com/sea-project/sea-pkg/wagon/wasm/operators"
)

type (
	execFunc func(vm *VM)
	gasFunc  func(vm *VM) (uint64, error)
)

type operation struct {
	execute execFunc
	gasCost gasFunc
	err     error
}

var (
	opSet [256]operation
)

func init() {
	opSet[ops.I32Clz] = operation{
		execute: func(vm *VM) { vm.i32Clz() },
		gasCost: constGasFunc(GasQuickStep),
	}
	opSet[ops.I32Ctz] = operation{
		execute: func(vm *VM) { vm.i32Ctz() },
		gasCost: constGasFunc(GasQuickStep),
	}
	opSet[ops.I32Popcnt] = operation{
		execute: func(vm *VM) { vm.i32Popcnt() },
		gasCost: constGasFunc(GasQuickStep),
	}
	opSet[ops.I32Add] = operation{
		execute: func(vm *VM) { vm.i32Add() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Sub] = operation{
		execute: func(vm *VM) { vm.i32Sub() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Mul] = operation{
		execute: func(vm *VM) { vm.i32Mul() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32DivS] = operation{
		execute: func(vm *VM) { vm.i32DivS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32DivU] = operation{
		execute: func(vm *VM) { vm.i32DivU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32RemS] = operation{
		execute: func(vm *VM) { vm.i32RemS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32RemU] = operation{
		execute: func(vm *VM) { vm.i32RemU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32And] = operation{
		execute: func(vm *VM) { vm.i32And() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Or] = operation{
		execute: func(vm *VM) { vm.i32Or() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Xor] = operation{
		execute: func(vm *VM) { vm.i32Xor() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Shl] = operation{
		execute: func(vm *VM) { vm.i32Shl() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32ShrS] = operation{
		execute: func(vm *VM) { vm.i32ShrS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32ShrU] = operation{
		execute: func(vm *VM) { vm.i32ShrU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Rotl] = operation{
		execute: func(vm *VM) { vm.i32Rotl() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Rotr] = operation{
		execute: func(vm *VM) { vm.i32Rotr() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Eqz] = operation{
		execute: func(vm *VM) { vm.i32Eqz() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Eq] = operation{
		execute: func(vm *VM) { vm.i32Eq() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Ne] = operation{
		execute: func(vm *VM) { vm.i32Ne() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32LtS] = operation{
		execute: func(vm *VM) { vm.i32LtS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32LtU] = operation{
		execute: func(vm *VM) { vm.i32LtU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32GtS] = operation{
		execute: func(vm *VM) { vm.i32GtS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32GtU] = operation{
		execute: func(vm *VM) { vm.i32GtU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32LeS] = operation{
		execute: func(vm *VM) { vm.i32LeS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32LeU] = operation{
		execute: func(vm *VM) { vm.i32LeU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32GeS] = operation{
		execute: func(vm *VM) { vm.i32GeS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32GeU] = operation{
		execute: func(vm *VM) { vm.i32GeU() },
		gasCost: constGasFunc(GasFastestStep),
	}

	opSet[ops.I64Clz] = operation{
		execute: func(vm *VM) { vm.i64Clz() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Ctz] = operation{
		execute: func(vm *VM) { vm.i64Ctz() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Popcnt] = operation{
		execute: func(vm *VM) { vm.i64Popcnt() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Add] = operation{
		execute: func(vm *VM) { vm.i64Add() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Sub] = operation{
		execute: func(vm *VM) { vm.i64Sub() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Mul] = operation{
		execute: func(vm *VM) { vm.i64Mul() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64DivS] = operation{
		execute: func(vm *VM) { vm.i64DivS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64DivU] = operation{
		execute: func(vm *VM) { vm.i64DivU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64RemS] = operation{
		execute: func(vm *VM) { vm.i64RemS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64RemU] = operation{
		execute: func(vm *VM) { vm.i64RemU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64And] = operation{
		execute: func(vm *VM) { vm.i64And() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Or] = operation{
		execute: func(vm *VM) { vm.i64Or() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Xor] = operation{
		execute: func(vm *VM) { vm.i64Xor() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Shl] = operation{
		execute: func(vm *VM) { vm.i64Shl() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64ShrS] = operation{
		execute: func(vm *VM) { vm.i64ShrS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64ShrU] = operation{
		execute: func(vm *VM) { vm.i64ShrU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Rotl] = operation{
		execute: func(vm *VM) { vm.i64Rotl() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Rotr] = operation{
		execute: func(vm *VM) { vm.i64Rotr() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Eqz] = operation{
		execute: func(vm *VM) { vm.i64Eqz() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Eq] = operation{
		execute: func(vm *VM) { vm.i64Eq() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Ne] = operation{
		execute: func(vm *VM) { vm.i64Ne() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64LtS] = operation{
		execute: func(vm *VM) { vm.i64LtS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64LtU] = operation{
		execute: func(vm *VM) { vm.i64LtU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64GtS] = operation{
		execute: func(vm *VM) { vm.i64GtS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64GtU] = operation{
		execute: func(vm *VM) { vm.i64GtU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64LeS] = operation{
		execute: func(vm *VM) { vm.i64LeS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64LeU] = operation{
		execute: func(vm *VM) { vm.i64LeU() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64GeS] = operation{
		execute: func(vm *VM) { vm.i64GeS() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64GeU] = operation{
		execute: func(vm *VM) { vm.i64GeU() },
		gasCost: constGasFunc(GasFastestStep),
	}

	// opSet[ops.F32Eq] = operation{
	// 	execute: vm.f32Eq,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Ne] = operation{
	// 	execute: vm.f32Ne,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Lt] = operation{
	// 	execute: vm.f32Lt,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Gt] = operation{
	// 	execute: vm.f32Gt,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Le] = operation{
	// 	execute: vm.f32Le,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Ge] = operation{
	// 	execute: vm.f32Ge,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Abs] = operation{
	// 	execute: vm.f32Abs,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Neg] = operation{
	// 	execute: vm.f32Neg,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Ceil] = operation{
	// 	execute: vm.f32Ceil,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Floor] = operation{
	// 	execute: vm.f32Floor,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Trunc] = operation{
	// 	execute: vm.f32Trunc,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Nearest] = operation{
	// 	execute: vm.f32Nearest,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Sqrt] = operation{
	// 	execute: vm.f32Sqrt,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Add] = operation{
	// 	execute: vm.f32Add,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Sub] = operation{
	// 	execute: vm.f32Sub,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Mul] = operation{
	// 	execute: vm.f32Mul,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Div] = operation{
	// 	execute: vm.f32Div,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Min] = operation{
	// 	execute: vm.f32Min,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Max] = operation{
	// 	execute: vm.f32Max,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32Copysign] = operation{
	// 	execute: vm.f32Copysign,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }

	// opSet[ops.F64Eq] = operation{
	// 	execute: vm.f64Eq,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Ne] = operation{
	// 	execute: vm.f64Ne,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Lt] = operation{
	// 	execute: vm.f64Lt,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Gt] = operation{
	// 	execute: vm.f64Gt,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Le] = operation{
	// 	execute: vm.f64Le,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Ge] = operation{
	// 	execute: vm.f64Ge,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Abs] = operation{
	// 	execute: vm.f64Abs,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Neg] = operation{
	// 	execute: vm.f64Neg,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Ceil] = operation{
	// 	execute: vm.f64Ceil,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Floor] = operation{
	// 	execute: vm.f64Floor,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Trunc] = operation{
	// 	execute: vm.f64Trunc,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Nearest] = operation{
	// 	execute: vm.f64Nearest,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Sqrt] = operation{
	// 	execute: vm.f64Sqrt,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Add] = operation{
	// 	execute: vm.f64Add,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Sub] = operation{
	// 	execute: vm.f64Sub,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Mul] = operation{
	// 	execute: vm.f64Mul,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Div] = operation{
	// 	execute: vm.f64Div,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Min] = operation{
	// 	execute: vm.f64Min,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Max] = operation{
	// 	execute: vm.f64Max,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Copysign] = operation{
	// 	execute: vm.f64Copysign,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }

	opSet[ops.I32Const] = operation{
		execute: func(vm *VM) { vm.i32Const() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Const] = operation{
		execute: func(vm *VM) { vm.i64Const() },
		gasCost: constGasFunc(GasFastestStep),
	}
	// opSet[ops.F32Const] = operation{
	// 	execute: vm.f32Const,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Const] = operation{
	// 	execute: vm.f64Const,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }

	// opSet[ops.I32ReinterpretF32] = operation{
	// 	execute: vm.i32ReinterpretF32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.I64ReinterpretF64] = operation{
	// 	execute: vm.i64ReinterpretF64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32ReinterpretI32] = operation{
	// 	execute: vm.f32ReinterpretI32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64ReinterpretI64] = operation{
	// 	execute: vm.f64ReinterpretI64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }

	opSet[ops.I32WrapI64] = operation{
		execute: func(vm *VM) { vm.i32Wrapi64() },
		gasCost: constGasFunc(GasFastestStep),
	}
	// opSet[ops.I32TruncSF32] = operation{
	// 	execute: vm.i32TruncSF32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.I32TruncUF32] = operation{
	// 	execute: vm.i32TruncUF32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.I32TruncSF64] = operation{
	// 	execute: vm.i32TruncSF64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.I32TruncUF64] = operation{
	// 	execute: vm.i32TruncUF64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	opSet[ops.I64ExtendSI32] = operation{
		execute: func(vm *VM) { vm.i64ExtendSI32() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64ExtendUI32] = operation{
		execute: func(vm *VM) { vm.i64ExtendUI32() },
		gasCost: constGasFunc(GasFastestStep),
	}
	// opSet[ops.I64TruncSF32] = operation{
	// 	execute: vm.i64TruncSF32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.I64TruncUF32] = operation{
	// 	execute: vm.i64TruncUF32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.I64TruncSF64] = operation{
	// 	execute: vm.i64TruncSF64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.I64TruncUF64] = operation{
	// 	execute: vm.i64TruncUF64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32ConvertSI32] = operation{
	// 	execute: vm.f32ConvertSI32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32ConvertUI32] = operation{
	// 	execute: vm.f32ConvertUI32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32ConvertSI64] = operation{
	// 	execute: vm.f32ConvertSI64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32ConvertUI64] = operation{
	// 	execute: vm.f32ConvertUI64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F32DemoteF64] = operation{
	// 	execute: vm.f32DemoteF64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64ConvertSI32] = operation{
	// 	execute: vm.f64ConvertSI32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64ConvertUI32] = operation{
	// 	execute: vm.f64ConvertUI32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64ConvertSI64] = operation{
	// 	execute: vm.f64ConvertSI64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64ConvertUI64] = operation{
	// 	execute: vm.f64ConvertUI64,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64PromoteF32] = operation{
	// 	execute: vm.f64PromoteF32,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }

	opSet[ops.I32Load] = operation{
		execute: func(vm *VM) { vm.i32Load() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Load] = operation{
		execute: func(vm *VM) { vm.i64Load() },
		gasCost: constGasFunc(GasFastestStep),
	}
	// opSet[ops.F32Load] = operation{
	// 	execute: vm.f32Load,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Load] = operation{
	// 	execute: vm.f64Load,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	opSet[ops.I32Load8s] = operation{
		execute: func(vm *VM) { vm.i32Load8s() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Load8u] = operation{
		execute: func(vm *VM) { vm.i32Load8u() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Load16s] = operation{
		execute: func(vm *VM) { vm.i32Load16s() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Load16u] = operation{
		execute: func(vm *VM) { vm.i32Load16u() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Load8s] = operation{
		execute: func(vm *VM) { vm.i64Load8s() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Load8u] = operation{
		execute: func(vm *VM) { vm.i64Load8u() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Load16s] = operation{
		execute: func(vm *VM) { vm.i64Load16s() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Load16u] = operation{
		execute: func(vm *VM) { vm.i64Load16u() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Load32s] = operation{
		execute: func(vm *VM) { vm.i64Load32s() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Load32u] = operation{
		execute: func(vm *VM) { vm.i64Load32u() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Store] = operation{
		execute: func(vm *VM) { vm.i32Store() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Store] = operation{
		execute: func(vm *VM) { vm.i64Store() },
		gasCost: constGasFunc(GasFastestStep),
	}
	// opSet[ops.F32Store] = operation{
	// 	execute: vm.f32Store,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	// opSet[ops.F64Store] = operation{
	// 	execute: vm.f64Store,
	// 	gasCost: constGasFunc(GasFastestStep),
	// }
	opSet[ops.I32Store8] = operation{
		execute: func(vm *VM) { vm.i32Store8() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I32Store16] = operation{
		execute: func(vm *VM) { vm.i32Store16() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Store8] = operation{
		execute: func(vm *VM) { vm.i64Store8() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Store16] = operation{
		execute: func(vm *VM) { vm.i64Store16() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.I64Store32] = operation{
		execute: func(vm *VM) { vm.i64Store32() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.CurrentMemory] = operation{
		execute: func(vm *VM) { vm.currentMemory() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.GrowMemory] = operation{
		execute: func(vm *VM) { vm.growMemory() },
		gasCost: constGasFunc(GasFastestStep),
	}

	opSet[ops.Drop] = operation{
		execute: func(vm *VM) { vm.drop() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.Select] = operation{
		execute: func(vm *VM) { vm.selectOp() },
		gasCost: constGasFunc(GasFastestStep),
	}

	opSet[ops.GetLocal] = operation{
		execute: func(vm *VM) { vm.getLocal() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.SetLocal] = operation{
		execute: func(vm *VM) { vm.setLocal() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.TeeLocal] = operation{
		execute: func(vm *VM) { vm.teeLocal() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.GetGlobal] = operation{
		execute: func(vm *VM) { vm.getGlobal() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.SetGlobal] = operation{
		execute: func(vm *VM) { vm.setGlobal() },
		gasCost: constGasFunc(GasFastestStep),
	}

	opSet[ops.Unreachable] = operation{
		execute: func(vm *VM) { vm.unreachable() },
		gasCost: constGasFunc(GasFastestStep),
	}
	opSet[ops.Nop] = operation{
		execute: func(vm *VM) { vm.nop() },
		gasCost: constGasFunc(GasFastestStep),
	}

	opSet[ops.Call] = operation{
		execute: func(vm *VM) { vm.call() },
		gasCost: gasCall, //constGasFunc(GasFastestStep),
	}
	opSet[ops.CallIndirect] = operation{
		execute: func(vm *VM) { vm.callIndirect() },
		gasCost: constGasFunc(GasFastestStep),
	}

}
