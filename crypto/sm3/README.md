# 国密GM算法相关

## SM3密码杂凑算法 - SM3 cryptographic hash algorithm

遵循的SM3标准号为： GM/T 0004-2012

### 代码示例

```
    data := "test"
    h := sm3.New()
    h.Write([]byte(data))
    sum := h.Sum(nil)
    fmt.Printf("digest value is: %x\n",sum)
```
### 方法列表

####  New 
创建哈希计算实例
```
func New() hash.Hash 
```

#### Sum 
返回SM3哈希算法摘要值
```
func Sum() []byte 
```