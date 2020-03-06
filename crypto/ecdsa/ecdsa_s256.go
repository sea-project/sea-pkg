package ecdsa

/**
这个包在内部对雅可比矩阵坐标进行操作。对于一个给定的曲线(x,y)位置,雅可比矩阵坐标是(x1,y1,z1)x = x1 / z1²和y = y₁/ z1³。
当整个计算可以在转换中执行时(如在ScalarMult和中)，速度提高最大ScalarBaseMult)。但即使是加法和乘法，应用和反转变换也比在仿射坐标下运算要快。
*/

import (
	"crypto/elliptic"
	"math/big"
	"sync"
)

var (
	// fieldOne只是字段表示中的整数1。它用于避免在内部运算期间多次创建它。
	fieldOne = new(fieldVal).SetInt(1)
)

// KoblitzCurve 支持koblitz曲线实现，它适合从crypto/elliptic到ECC曲线的接口。
type KoblitzCurve struct {
	*elliptic.CurveParams
	q *big.Int
	H int // 曲线的余子式。

	// 字节大小就是位大小/ 8，它是为了方便而提供的，因为它是重复计算的。
	byteSize int

	// 字节点
	bytePoints *[32][256][3]fieldVal

	// 下面6个值专门用于ScalarMult中的自同态优化。
	lambda *big.Int
	beta   *fieldVal
	a1     *big.Int
	b1     *big.Int
	a2     *big.Int
	b2     *big.Int
}

// Params 参数返回曲线的参数
func (curve *KoblitzCurve) Params() *elliptic.CurveParams {
	return curve.CurveParams
}

// bigAffineToField 将仿射点(x, y)作为大整数，并将其转换为仿射点作为字段值。
func (curve *KoblitzCurve) bigAffineToField(x, y *big.Int) (*fieldVal, *fieldVal) {
	x3, y3 := new(fieldVal), new(fieldVal)
	x3.SetByteSlice(x.Bytes())
	y3.SetByteSlice(y.Bytes())
	return x3, y3
}

// fieldJacobianToBigAffine 以雅可比矩阵点(x, y, z)作为字段值，并将其转换为大整数形式的仿射点。
func (curve *KoblitzCurve) fieldJacobianToBigAffine(x, y, z *fieldVal) (*big.Int, *big.Int) {
	// 反转代价高昂，而且当处理z值为1的点时，点加法和点倍增都更快。所以，如果这个点需要转换为仿射，那么在计算相同的同时，对这个点本身进行归一化。
	var zInv, tempZ fieldVal
	zInv.Set(z).Inverse()   // zInv = Z^-1
	tempZ.SquareVal(&zInv)  // tempZ = Z^-2
	x.Mul(&tempZ)           // X = X/Z^2 (mag: 1)
	y.Mul(tempZ.Mul(&zInv)) // Y = Y/Z^3 (mag: 1)
	z.SetInt(1)             // Z = 1 (mag: 1)
	x.Normalize()
	y.Normalize()
	x3, y3 := new(big.Int), new(big.Int)
	x3.SetBytes(x.Bytes()[:])
	y3.SetBytes(y.Bytes()[:])
	return x3, y3
}

// IsOnCurve 通过传入公钥的X轴和Y轴的值，判断该公钥是否在secp256k1曲线上
// 如果点(x,y)在曲线上，IsOnCurve返回布尔值。椭圆的一部分。曲线界面。这个函数不同于加密/椭圆算法，因为a = 0而不是-3。
func (curve *KoblitzCurve) IsOnCurve(x, y *big.Int) bool {
	fx, fy := curve.bigAffineToField(x, y)
	y2 := new(fieldVal).SquareVal(fy).Normalize()
	result := new(fieldVal).SquareVal(fx).Mul(fx).AddInt(7).Normalize()
	return y2.Equals(result)
}

// addZ1AndZ2EqualsOne 添加两个已知z值为1的雅可比矩阵点，并将结果存储在(x3, y3, z3)中。
// 也就是说(x1, y1, 1) + (x2, y2, 1) = (x3, y3, z3)与一般的加法程序相比，它执行加法的速度更快，因为它能够避免z值乘法，因此所需的算术更少。
func (curve *KoblitzCurve) addZ1AndZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3 *fieldVal) {
	x1.Normalize()
	y1.Normalize()
	x2.Normalize()
	y2.Normalize()
	if x1.Equals(x2) {
		if y1.Equals(y2) {
			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}
	var h, i, j, r, v fieldVal
	var negJ, neg2V, negX3 fieldVal
	h.Set(x1).Negate(1).Add(x2)                // H = X2-X1 (mag: 3)
	i.SquareVal(&h).MulInt(4)                  // I = 4*H^2 (mag: 4)
	j.Mul2(&h, &i)                             // J = H*I (mag: 1)
	r.Set(y1).Negate(1).Add(y2).MulInt(2)      // r = 2*(Y2-Y1) (mag: 6)
	v.Mul2(x1, &i)                             // V = X1*I (mag: 1)
	negJ.Set(&j).Negate(1)                     // negJ = -J (mag: 2)
	neg2V.Set(&v).MulInt(2).Negate(2)          // neg2V = -(2*V) (mag: 3)
	x3.Set(&r).Square().Add(&negJ).Add(&neg2V) // X3 = r^2-J-2*V (mag: 6)
	negX3.Set(x3).Negate(6)                    // negX3 = -X3 (mag: 7)
	j.Mul(y1).MulInt(2).Negate(2)              // J = -(2*Y1*J) (mag: 3)
	y3.Set(&v).Add(&negX3).Mul(&r).Add(&j)     // Y3 = r*(V-X3)-2*Y1*J (mag: 4)
	z3.Set(&h).MulInt(2)                       // Z3 = 2*H (mag: 6)
	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

// addZ1EqualsZ2 添加了两个已知具有相同z值的雅可比点，并将结果存储在(x3, y3, z3)中。也就是说(x1, y1, z1) + (x2, y2, z1) = (x3, y3, z3)它执行的加法比一般的加法程序快，因为已知的等价性使得所需的算术更少。
func (curve *KoblitzCurve) addZ1EqualsZ2(x1, y1, z1, x2, y2, x3, y3, z3 *fieldVal) {
	x1.Normalize()
	y1.Normalize()
	x2.Normalize()
	y2.Normalize()
	if x1.Equals(x2) {
		if y1.Equals(y2) {
			// 由于x1 == x2和y1 == y2，必须进行点乘，否则加法最终会除以0。
			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}
	var a, b, c, d, e, f fieldVal
	var negX1, negY1, negE, negX3 fieldVal
	negX1.Set(x1).Negate(1)                // negX1 = -X1 (mag: 2)
	negY1.Set(y1).Negate(1)                // negY1 = -Y1 (mag: 2)
	a.Set(&negX1).Add(x2)                  // A = X2-X1 (mag: 3)
	b.SquareVal(&a)                        // B = A^2 (mag: 1)
	c.Set(&negY1).Add(y2)                  // C = Y2-Y1 (mag: 3)
	d.SquareVal(&c)                        // D = C^2 (mag: 1)
	e.Mul2(x1, &b)                         // E = X1*B (mag: 1)
	negE.Set(&e).Negate(1)                 // negE = -E (mag: 2)
	f.Mul2(x2, &b)                         // F = X2*B (mag: 1)
	x3.Add2(&e, &f).Negate(3).Add(&d)      // X3 = D-E-F (mag: 5)
	negX3.Set(x3).Negate(5).Normalize()    // negX3 = -X3 (mag: 1)
	y3.Set(y1).Mul(f.Add(&negE)).Negate(3) // Y3 = -(Y1*(F-E)) (mag: 4)
	y3.Add(e.Add(&negX3).Mul(&c))          // Y3 = C*(E-X3)+Y3 (mag: 5)
	z3.Mul2(z1, &a)                        // Z3 = Z1*A (mag: 1)
	x3.Normalize()
	y3.Normalize()
}

// addZ2EqualsOne 当已知第二个点的z值为1时(第一个点的z值不是1)，addZ2EqualsOne添加两个雅可比矩阵点，并将结果存储在(x3, y3, z3)中。
// 也就是说(x1, y1, z1) + (x2, y2, 1) = (x3, y3, z3)它执行的加法比一般的加法程序要快，因为由于能够避免用第二个点的z值进行乘法运算，因此需要更少的算术运算。
func (curve *KoblitzCurve) addZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3 *fieldVal) {
	var z1z1, u2, s2 fieldVal
	x1.Normalize()
	y1.Normalize()
	z1z1.SquareVal(z1)                        // Z1Z1 = Z1^2 (mag: 1)
	u2.Set(x2).Mul(&z1z1).Normalize()         // U2 = X2*Z1Z1 (mag: 1)
	s2.Set(y2).Mul(&z1z1).Mul(z1).Normalize() // S2 = Y2*Z1*Z1Z1 (mag: 1)
	if x1.Equals(&u2) {
		if y1.Equals(&s2) {
			// 由于x1 == x2和y1 == y2，必须进行点乘，否则加法最终会除以0。
			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}
	var h, hh, i, j, r, rr, v fieldVal
	var negX1, negY1, negX3 fieldVal
	negX1.Set(x1).Negate(1)                // negX1 = -X1 (mag: 2)
	h.Add2(&u2, &negX1)                    // H = U2-X1 (mag: 3)
	hh.SquareVal(&h)                       // HH = H^2 (mag: 1)
	i.Set(&hh).MulInt(4)                   // I = 4 * HH (mag: 4)
	j.Mul2(&h, &i)                         // J = H*I (mag: 1)
	negY1.Set(y1).Negate(1)                // negY1 = -Y1 (mag: 2)
	r.Set(&s2).Add(&negY1).MulInt(2)       // r = 2*(S2-Y1) (mag: 6)
	rr.SquareVal(&r)                       // rr = r^2 (mag: 1)
	v.Mul2(x1, &i)                         // V = X1*I (mag: 1)
	x3.Set(&v).MulInt(2).Add(&j).Negate(3) // X3 = -(J+2*V) (mag: 4)
	x3.Add(&rr)                            // X3 = r^2+X3 (mag: 5)
	negX3.Set(x3).Negate(5)                // negX3 = -X3 (mag: 6)
	y3.Set(y1).Mul(&j).MulInt(2).Negate(2) // Y3 = -(2*Y1*J) (mag: 3)
	y3.Add(v.Add(&negX3).Mul(&r))          // Y3 = r*(V-X3)+Y3 (mag: 4)
	z3.Add2(z1, &h).Square()               // Z3 = (Z1+H)^2 (mag: 1)
	z3.Add(z1z1.Add(&hh).Negate(2))        // Z3 = Z3-(Z1Z1+HH) (mag: 4)
	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

// addGeneric 添加两个雅可比矩阵点(x1, y1, z1)和(x2, y2, z2)，不需要对这两个点的z值做任何假设，并将结果存储为(x3, y3, z3)。也就是说(x1, y1, z1) + (x2, y2, z2) = (x3, y3, z3)它是添加例程中最慢的，因为需要最多的算术。
func (curve *KoblitzCurve) addGeneric(x1, y1, z1, x2, y2, z2, x3, y3, z3 *fieldVal) {
	var z1z1, z2z2, u1, u2, s1, s2 fieldVal
	z1z1.SquareVal(z1)                        // Z1Z1 = Z1^2 (mag: 1)
	z2z2.SquareVal(z2)                        // Z2Z2 = Z2^2 (mag: 1)
	u1.Set(x1).Mul(&z2z2).Normalize()         // U1 = X1*Z2Z2 (mag: 1)
	u2.Set(x2).Mul(&z1z1).Normalize()         // U2 = X2*Z1Z1 (mag: 1)
	s1.Set(y1).Mul(&z2z2).Mul(z2).Normalize() // S1 = Y1*Z2*Z2Z2 (mag: 1)
	s2.Set(y2).Mul(&z1z1).Mul(z1).Normalize() // S2 = Y2*Z1*Z1Z1 (mag: 1)
	if u1.Equals(&u2) {
		if s1.Equals(&s2) {
			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}
	var h, i, j, r, rr, v fieldVal
	var negU1, negS1, negX3 fieldVal
	negU1.Set(&u1).Negate(1)               // negU1 = -U1 (mag: 2)
	h.Add2(&u2, &negU1)                    // H = U2-U1 (mag: 3)
	i.Set(&h).MulInt(2).Square()           // I = (2*H)^2 (mag: 2)
	j.Mul2(&h, &i)                         // J = H*I (mag: 1)
	negS1.Set(&s1).Negate(1)               // negS1 = -S1 (mag: 2)
	r.Set(&s2).Add(&negS1).MulInt(2)       // r = 2*(S2-S1) (mag: 6)
	rr.SquareVal(&r)                       // rr = r^2 (mag: 1)
	v.Mul2(&u1, &i)                        // V = U1*I (mag: 1)
	x3.Set(&v).MulInt(2).Add(&j).Negate(3) // X3 = -(J+2*V) (mag: 4)
	x3.Add(&rr)                            // X3 = r^2+X3 (mag: 5)
	negX3.Set(x3).Negate(5)                // negX3 = -X3 (mag: 6)
	y3.Mul2(&s1, &j).MulInt(2).Negate(2)   // Y3 = -(2*S1*J) (mag: 3)
	y3.Add(v.Add(&negX3).Mul(&r))          // Y3 = r*(V-X3)+Y3 (mag: 4)
	z3.Add2(z1, z2).Square()               // Z3 = (Z1+Z2)^2 (mag: 1)
	z3.Add(z1z1.Add(&z2z2).Negate(2))      // Z3 = Z3-(Z1Z1+Z2Z2) (mag: 4)
	z3.Mul(&h)                             // Z3 = Z3*H (mag: 1)
	x3.Normalize()
	y3.Normalize()
}

// addJacobian 将传递的雅可比矩阵点(x1, y1, z1)和(x2, y2, z2)相加，并将结果存储在(x3, y3, z3)中。
func (curve *KoblitzCurve) addJacobian(x1, y1, z1, x2, y2, z2, x3, y3, z3 *fieldVal) {
	if (x1.IsZero() && y1.IsZero()) || z1.IsZero() {
		x3.Set(x2)
		y3.Set(y2)
		z3.Set(z2)
		return
	}
	if (x2.IsZero() && y2.IsZero()) || z2.IsZero() {
		x3.Set(x1)
		y3.Set(y1)
		z3.Set(z1)
		return
	}
	z1.Normalize()
	z2.Normalize()
	isZ1One := z1.Equals(fieldOne)
	isZ2One := z2.Equals(fieldOne)
	switch {
	case isZ1One && isZ2One:
		curve.addZ1AndZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3)
		return
	case z1.Equals(z2):
		curve.addZ1EqualsZ2(x1, y1, z1, x2, y2, x3, y3, z3)
		return
	case isZ2One:
		curve.addZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3)
		return
	}
	curve.addGeneric(x1, y1, z1, x2, y2, z2, x3, y3, z3)
}

// Add 加法返回(x1,y1)和(x2,y2)的和。椭圆的一部分。曲线界面。
func (curve *KoblitzCurve) Add(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {
	if x1.Sign() == 0 && y1.Sign() == 0 {
		return x2, y2
	}
	if x2.Sign() == 0 && y2.Sign() == 0 {
		return x1, y1
	}
	fx1, fy1 := curve.bigAffineToField(x1, y1)
	fx2, fy2 := curve.bigAffineToField(x2, y2)
	fx3, fy3, fz3 := new(fieldVal), new(fieldVal), new(fieldVal)
	fOne := new(fieldVal).SetInt(1)
	curve.addJacobian(fx1, fy1, fOne, fx2, fy2, fOne, fx3, fy3, fz3)
	return curve.fieldJacobianToBigAffine(fx3, fy3, fz3)
}

func (curve *KoblitzCurve) doubleZ1EqualsOne(x1, y1, x3, y3, z3 *fieldVal) {
	var a, b, c, d, e, f fieldVal
	z3.Set(y1).MulInt(2)                     // Z3 = 2*Y1 (mag: 2)
	a.SquareVal(x1)                          // A = X1^2 (mag: 1)
	b.SquareVal(y1)                          // B = Y1^2 (mag: 1)
	c.SquareVal(&b)                          // C = B^2 (mag: 1)
	b.Add(x1).Square()                       // B = (X1+B)^2 (mag: 1)
	d.Set(&a).Add(&c).Negate(2)              // D = -(A+C) (mag: 3)
	d.Add(&b).MulInt(2)                      // D = 2*(B+D)(mag: 8)
	e.Set(&a).MulInt(3)                      // E = 3*A (mag: 3)
	f.SquareVal(&e)                          // F = E^2 (mag: 1)
	x3.Set(&d).MulInt(2).Negate(16)          // X3 = -(2*D) (mag: 17)
	x3.Add(&f)                               // X3 = F+X3 (mag: 18)
	f.Set(x3).Negate(18).Add(&d).Normalize() // F = D-X3 (mag: 1)
	y3.Set(&c).MulInt(8).Negate(8)           // Y3 = -(8*C) (mag: 9)
	y3.Add(f.Mul(&e))                        // Y3 = E*F+Y3 (mag: 10)

	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

// doubleGeneric 对经过的雅可比矩阵点执行点加倍，而不需要对z值进行任何假设，并将结果存储为(x3, y3, z3)。
func (curve *KoblitzCurve) doubleGeneric(x1, y1, z1, x3, y3, z3 *fieldVal) {

	var a, b, c, d, e, f fieldVal
	z3.Mul2(y1, z1).MulInt(2)                // Z3 = 2*Y1*Z1 (mag: 2)
	a.SquareVal(x1)                          // A = X1^2 (mag: 1)
	b.SquareVal(y1)                          // B = Y1^2 (mag: 1)
	c.SquareVal(&b)                          // C = B^2 (mag: 1)
	b.Add(x1).Square()                       // B = (X1+B)^2 (mag: 1)
	d.Set(&a).Add(&c).Negate(2)              // D = -(A+C) (mag: 3)
	d.Add(&b).MulInt(2)                      // D = 2*(B+D)(mag: 8)
	e.Set(&a).MulInt(3)                      // E = 3*A (mag: 3)
	f.SquareVal(&e)                          // F = E^2 (mag: 1)
	x3.Set(&d).MulInt(2).Negate(16)          // X3 = -(2*D) (mag: 17)
	x3.Add(&f)                               // X3 = F+X3 (mag: 18)
	f.Set(x3).Negate(18).Add(&d).Normalize() // F = D-X3 (mag: 1)
	y3.Set(&c).MulInt(8).Negate(8)           // Y3 = -(8*C) (mag: 9)
	y3.Add(f.Mul(&e))                        // Y3 = E*F+Y3 (mag: 10)

	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

func (curve *KoblitzCurve) doubleJacobian(x1, y1, z1, x3, y3, z3 *fieldVal) {

	if y1.IsZero() || z1.IsZero() {
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}
	if z1.Normalize().Equals(fieldOne) {
		curve.doubleZ1EqualsOne(x1, y1, x3, y3, z3)
		return
	}

	curve.doubleGeneric(x1, y1, z1, x3, y3, z3)
}

// Double 返回2 * (x1, y1)。椭圆的一部分。曲线界面。
func (curve *KoblitzCurve) Double(x1, y1 *big.Int) (*big.Int, *big.Int) {
	if y1.Sign() == 0 {
		return new(big.Int), new(big.Int)
	}
	fx1, fy1 := curve.bigAffineToField(x1, y1)
	fx3, fy3, fz3 := new(fieldVal), new(fieldVal), new(fieldVal)
	fOne := new(fieldVal).SetInt(1)
	curve.doubleJacobian(fx1, fy1, fOne, fx3, fy3, fz3)
	return curve.fieldJacobianToBigAffine(fx3, fy3, fz3)
}

// splitK 返回一个平衡的长度为2的k及其符号表示。
func (curve *KoblitzCurve) splitK(k []byte) ([]byte, []byte, int, int) {

	bigIntK := new(big.Int)
	c1, c2 := new(big.Int), new(big.Int)
	tmp1, tmp2 := new(big.Int), new(big.Int)
	k1, k2 := new(big.Int), new(big.Int)

	bigIntK.SetBytes(k)
	c1.Mul(curve.b2, bigIntK)
	c1.Div(c1, curve.N)
	c2.Mul(curve.b1, bigIntK)
	c2.Div(c2, curve.N)
	tmp1.Mul(c1, curve.a1)
	tmp2.Mul(c2, curve.a2)
	k1.Sub(bigIntK, tmp1)
	k1.Add(k1, tmp2)
	tmp1.Mul(c1, curve.b1)
	tmp2.Mul(c2, curve.b2)
	k2.Sub(tmp2, tmp1)

	return k1.Bytes(), k2.Bytes(), k1.Sign(), k2.Sign()
}

// moduloReduce 将k从32字节以上减少到32字节以下。这是通过做一个简单的模曲线来实现的。我们可以这样做因为G ^ N = 1,因此任何其他有效点椭圆曲线有相同的顺序。
func (curve *KoblitzCurve) moduloReduce(k []byte) []byte {

	if len(k) > curve.byteSize {
		tmpK := new(big.Int).SetBytes(k)
		tmpK.Mod(tmpK, curve.N)
		return tmpK.Bytes()
	}

	return k
}

// naf 取一个正整数k，使得最小化操作数量成为可能，因为返回的结果int至少为50%。
func naf(k []byte) ([]byte, []byte) {

	var carry, curIsOne, nextIsOne bool
	// 这些默认值为0
	retPos := make([]byte, len(k)+1)
	retNeg := make([]byte, len(k)+1)
	for i := len(k) - 1; i >= 0; i-- {
		curByte := k[i]
		for j := uint(0); j < 8; j++ {
			curIsOne = curByte&1 == 1
			if j == 7 {
				if i == 0 {
					nextIsOne = false
				} else {
					nextIsOne = k[i-1]&1 == 1
				}
			} else {
				nextIsOne = curByte&2 == 2
			}
			if carry {
				if curIsOne {
				} else {
					if nextIsOne {
						retNeg[i+1] += 1 << j
					} else {
						carry = false
						retPos[i+1] += 1 << j
					}
				}
			} else if curIsOne {
				if nextIsOne {
					retNeg[i+1] += 1 << j
					carry = true
				} else {
					// This is a singleton, not consecutive
					// 1s.
					retPos[i+1] += 1 << j
				}
			}
			curByte >>= 1
		}
	}
	if carry {
		retPos[0] = 1
	}

	return retPos, retNeg
}

// ScalarMult 返回k*(Bx, By)，其中k是一个大端整数。椭圆的一部分。曲线界面。
func (curve *KoblitzCurve) ScalarMult(Bx, By *big.Int, k []byte) (*big.Int, *big.Int) {
	qx, qy, qz := new(fieldVal), new(fieldVal), new(fieldVal)

	k1, k2, signK1, signK2 := curve.splitK(curve.moduloReduce(k))

	p1x, p1y := curve.bigAffineToField(Bx, By)
	p1yNeg := new(fieldVal).NegateVal(p1y, 1)
	p1z := new(fieldVal).SetInt(1)

	p2x := new(fieldVal).Mul2(p1x, curve.beta)
	p2y := new(fieldVal).Set(p1y)
	p2yNeg := new(fieldVal).NegateVal(p2y, 1)
	p2z := new(fieldVal).SetInt(1)

	if signK1 == -1 {
		p1y, p1yNeg = p1yNeg, p1y
	}
	if signK2 == -1 {
		p2y, p2yNeg = p2yNeg, p2y
	}

	k1PosNAF, k1NegNAF := naf(k1)
	k2PosNAF, k2NegNAF := naf(k2)
	k1Len := len(k1PosNAF)
	k2Len := len(k2PosNAF)

	m := k1Len
	if m < k2Len {
		m = k2Len
	}

	var k1BytePos, k1ByteNeg, k2BytePos, k2ByteNeg byte
	for i := 0; i < m; i++ {

		if i < m-k1Len {
			k1BytePos = 0
			k1ByteNeg = 0
		} else {
			k1BytePos = k1PosNAF[i-(m-k1Len)]
			k1ByteNeg = k1NegNAF[i-(m-k1Len)]
		}
		if i < m-k2Len {
			k2BytePos = 0
			k2ByteNeg = 0
		} else {
			k2BytePos = k2PosNAF[i-(m-k2Len)]
			k2ByteNeg = k2NegNAF[i-(m-k2Len)]
		}

		for j := 7; j >= 0; j-- {

			curve.doubleJacobian(qx, qy, qz, qx, qy, qz)

			if k1BytePos&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p1x, p1y, p1z,
					qx, qy, qz)
			} else if k1ByteNeg&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p1x, p1yNeg, p1z,
					qx, qy, qz)
			}

			if k2BytePos&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p2x, p2y, p2z,
					qx, qy, qz)
			} else if k2ByteNeg&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p2x, p2yNeg, p2z,
					qx, qy, qz)
			}
			k1BytePos <<= 1
			k1ByteNeg <<= 1
			k2BytePos <<= 1
			k2ByteNeg <<= 1
		}
	}

	return curve.fieldJacobianToBigAffine(qx, qy, qz)
}

//ScalarBaseMult 返回k*G，其中G是组的基点，k是一个大端整数。 椭圆的一部分。曲线界面。
func (curve *KoblitzCurve) ScalarBaseMult(k []byte) (*big.Int, *big.Int) {
	newK := curve.moduloReduce(k)
	diff := len(curve.bytePoints) - len(newK)

	qx, qy, qz := new(fieldVal), new(fieldVal), new(fieldVal)

	for i, byteVal := range newK {
		p := curve.bytePoints[diff+i][byteVal]
		curve.addJacobian(qx, qy, qz, &p[0], &p[1], &p[2], qx, qy, qz)
	}
	return curve.fieldJacobianToBigAffine(qx, qy, qz)
}

var initonce sync.Once
var secp256k1 KoblitzCurve

func fromHex(s string) *big.Int {
	r, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("invalid hex in source file: " + s)
	}
	return r
}

func initS256() {
	secp256k1.CurveParams = new(elliptic.CurveParams)
	secp256k1.P = fromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F")
	secp256k1.N = fromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141")
	secp256k1.B = fromHex("0000000000000000000000000000000000000000000000000000000000000007")
	secp256k1.Gx = fromHex("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798")
	secp256k1.Gy = fromHex("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8")
	secp256k1.BitSize = 256
	secp256k1.H = 1
	secp256k1.q = new(big.Int).Div(new(big.Int).Add(secp256k1.P, big.NewInt(1)), big.NewInt(4))
	secp256k1.byteSize = secp256k1.BitSize / 8

	if err := loadS256BytePoints(); err != nil {
		panic(err)
	}
	secp256k1.lambda = fromHex("5363AD4CC05C30E0A5261C028812645A122E22EA20816678DF02967C1B23BD72")
	secp256k1.beta = new(fieldVal).SetHex("7AE96A2B657C07106E64479EAC3434E99CF0497512F58995C1396C28719501EE")
	secp256k1.a1 = fromHex("3086D221A7D46BCDE86C90E49284EB15")
	secp256k1.b1 = fromHex("-E4437ED6010E88286F547FA90ABFE4C3")
	secp256k1.a2 = fromHex("114CA50F7A8E2F3F657C1108D9D44CFD8")
	secp256k1.b2 = fromHex("3086D221A7D46BCDE86C90E49284EB15")
}

// S256 返回一条实现secp256k1的曲线。.
func S256() *KoblitzCurve {
	initonce.Do(initS256) //获取到了一个初始化后的曲线，并通过initonce这个同步对象调用了Do方法来对这个曲线进行了初始化操作
	return &secp256k1
}
