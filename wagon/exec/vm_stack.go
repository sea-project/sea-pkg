// +build !debugstack

package exec

// debugStackDepth enables runtime checks of the stack depth. If
// the stack every would exceed or underflow its expected bounds,
// a panic is thrown.
const debugStackDepth = false
