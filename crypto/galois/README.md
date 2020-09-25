# 密钥分片算法相关

### 代码示例

```
    split, err := Split([]byte("0ddb327ad1059662da1f02f1b8521bf0f69cf5cecc09a4d8fc7f928fc9726818"), 5, 2)
    if err != nil {
        panic(err)
    }
    for i, v := range split {
        fmt.Println(i, hex.EncodeToString(v))
    }
    bytes, err := Combine([][]byte{split[0], split[1]})
    if err != nil {
        panic(err)
    }
    fmt.Println(string(bytes))
```
### 方法列表

####  Split 
密钥分割，指定分割个数及其合并需要的个数
```
func Split(secret []byte, parts, threshold int) ([][]byte, error)
```

#### Combine 
将分片密钥合并
```
func Combine(parts [][]byte) ([]byte, error) 
```