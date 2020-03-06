# 国密GM算法相关

## SM2椭圆曲线公钥密码算法 - Public key cryptographic algorithm SM2 based on elliptic curves

遵循的SM2标准号为： GM/T 0003.1-2012、GM/T 0003.2-2012、GM/T 0003.3-2012、GM/T 0003.4-2012、GM/T 0003.5-2012、GM/T 0009-2012、GM/T 0010-2012

### 方法列表
 
#### GenerateKey
生成随机密钥。
```
func GenerateKey() (*PrivateKey, error) 
```

#### Sign
用私钥签名数据，成功返回以两个大数表示的签名结果，否则返回错误。
```
func Sign(priv *PrivateKey, hash []byte) (r, s *big.Int, err error)
```

#### Verify
用公钥验证数据签名, 验证成功返回True，否则返回False。
```
func Verify(pub *PublicKey, hash []byte, r, s *big.Int) bool 
```

#### Encrypt
用公钥加密数据,成功返回密文错误，否则返回错误。
```
func Encrypt(pub *PublicKey, data []byte) ([]byte, error) 
```

#### Decrypt
用私钥解密数据，成功返回原始明文数据，否则返回错误。
```
func Decrypt(priv *PrivateKey, data []byte) ([]byte, error)
```

#### Public
通过私钥获得公钥
```
func (priv *PrivateKey) Public() crypto.PublicKey
```

#### Compress
将公钥序列化为33位的[]byte.
```
func Compress(a *PublicKey) []byte
```

#### RawBytesToPublicKey
[]byte转公钥
```
func RawBytesToPublicKey(bytes []byte) (*PublicKey, error)
```

#### RawBytesToPrivateKey
[]byte转私钥
```
func RawBytesToPrivateKey(bytes []byte) (*PrivateKey, error)
```