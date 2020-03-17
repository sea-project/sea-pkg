package operators

var (
	GetLocal  = newPolymorphicOp(0x20, "get_local")
	SetLocal  = newPolymorphicOp(0x21, "set_local")
	TeeLocal  = newPolymorphicOp(0x22, "tee_local")
	GetGlobal = newPolymorphicOp(0x23, "get_global")
	SetGlobal = newPolymorphicOp(0x24, "set_global")
)
