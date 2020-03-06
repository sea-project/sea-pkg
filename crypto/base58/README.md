[![Travis CI](https://travis-ci.org/shengdoushi/base58.svg?branch=master)](http://10.1.1.28/go/crypto/tree/master/base58)
[![GoDoc](https://www.godoc.org/github.com/shengdoushi/base58?status.svg)](https://www.godoc.org/github.com/shengdoushi/base58)
[![Go Report Card](https://goreportcard.com/badge/github.com/shengdoushi/base58)](https://goreportcard.com/report/github.com/shengdoushi/base58)

[^1]:上面三个图标在每个README里面可有可不有，但如果项目在GitHub或固定网站上，或是你的项目包引用了网上资源，请使用这三个图标嵌入引用或说明链接地址。
[^1]:passing对应着项目所在网址,reference对应着参考网址，report对应着文档地址

## 特点

 * 快速轻量
 * API 语法简单
 * 内置常用的几种编码表: 比特币, IPFS, Flickr, Ripple
 * 可以自定义编码表
 * 自定义编码表可以是unicode字符串

## API Doc

[Godoc](https://www.godoc.org/github.com/shengdoushi/base58)

## base58 算法

类似base64编码算法， 但是去掉了几个看起来相同的字符(数字0, 大写字母O, 字母i的大写字母I, 字母L的小写字母l), 以及非字母数字字符(+,/).只含有字母，数字。优点是不易看错字符，且在大部分字符显示场景中，可以双击复制。

## 主要API

提供了 2 个API:

```
// 编码
func Encode(input []byte, alphabet *Alphabet)string

// 解码
func Decode(input string, alphabet *Alphabet)([]byte, error)
```

## 使用

```golang
import "github.com/shengdoushi/base58"
	
// 指定符号表
// myAlphabet := base58.BitcoinAlphabet // 使用 bitcoin 的符号表
myAlphabet := base58.NewAlphabet("ABCDEFGHJKLMNPQRSTUVWXYZ123456789abcdefghijkmnopqrstuvwxyz") // 自定义符号表
	
// 编码成 string 
var encodedStr string = base58.Encode([]byte{1,2,3,4}, myAlphabet)
	
// 解码为 []byte 
var encodedString string = "Xsdfjs123D"
decodedBytes, err := base58.Decode(encodedString, myAlphabet)
if err != nil {
	// error occurred
}
```

## 示例

```golang
package main

import (
	"fmt"
	"github.com/shengdoushi/base58"
)

func main(){
	// 这里使用比特币的符号表, 同 base58.NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	myAlphabet := base58.BitcoinAlphabet
	
	// 编码
	input := []byte{0,0,0,1,2,3}
	var encodedString string = base58.Encode(input, myAlphabet)
	fmt.Printf("base58encode(%v) = %s\n", input, encodedString)
	
	// 解码， 如果输入的字符中有符号表中不含的字符会返回错误
	decodedBytes, err := base58.Decode(encodedString, myAlphabet)
	if err != nil {
		fmt.Println("error occurred: ", err)
	}else{
		fmt.Printf("base58decode(%s) = %v\n", encodedString, decodedBytes)
	}	
}
```

示例输出如下：

```
base58encode([0 0 0 1 2 3]) = 111Ldp
base58decode(111Ldp) = [0 0 0 1 2 3]
```

# 代码详解
### base58.go   

#### 全局变量

变量名 | 变量类型 | 变量值 | 注解
---|---|---|---
BscAlphabet | *Alphabet | NewAlphabet("1Aa2Bb3Cc4Dd5Ee6Ff7Gg8Hh9jJKkLlMmNnoPpQqrRSsTtUuVvWwXxYyZz") | 自定义BSC字母表
BitcoinAlphabet | *Alphabet | NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz") | 比特币的字母表
IPFSAlphabet | *Alphabet | NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz") | IPFS的字母表
FlickrAlphabet | *Alphabet | NewAlphabet("123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ") | Flickr的字母表  
RippleAlphabet | *Alphabet | NewAlphabet("rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz") | Ripple的字母表

#### 结构体 Alphabet  

概要：  
> 结构体Alphabet又称为base58字母表对象，它是通过NewAlphabet函数根据自定义的规则字符串生成的，作用是用来加密和解密。


 参数名 | 参数类型 | 值 |  注解
---|---|---|---
encodeTable | [58]rune | 调用NewAlphabet方法时赋值 | 编码表 
decodeTable | [256]int | 调用NewAlphabet方法时赋值 | 解码表
unicodeDecodeTable | []rune | 调用NewAlphabet方法时赋值 | 双字节解码表



#### 函数/方法

 序号 | 方法名 | 作用 | 参数 | 返回值
---|---|---|---|---
 1 | String | 将Alphabet字母表转换为字符串返回 | alphabet Alphabet | string
 2 | NewAlphabet | 根据长度为58位的字符串创建一个自定义字母表 | alphabet string | *Alphabet
 3 | Encode | 根据传进来的参数进行加密 | input []byte, alphabet *Alphabet | string
 4 | Decode | 使用指定的字母表对密文进行解码 | input string, alphabet *Alphabet | []byte, error

### base58_test.go   

#### 全局变量
变量名 | 变量类型 | 变量值 | 注解
---|---|---|---
testCases | [][][]byte | 通过init方法进行赋值 | 在加密、解密的TPS测试用例中使用

#### 测试用例  
 序号 | 方法名 | 作用 | 测试结果 
---|---|---|---
 1 | init | 做初始化，对testCases进行赋值 | 因为有rand.Read()方法参与，最后的值是随机生成值，不固定
 2 | TestAlphabetImplStringer | 测试创建自定义的字母表 | 若正确就输出字母表，否则输出错误提示信息 
 3 | TestAlphabetFix58Length | 测试自定义的字母表长度是否为58位 | 若正确就输出字母表，否则输出错误提示信息 
 4 | TestUnicodeAlphabet | 使用双字节验证加密和解密方法 | 若正确则输出的解密参数和预定的参数符合，若错误则输出提示信息
 5 | TestRandCases | 使用指定的字母表对密文进行解码 | input string, alphabet *Alphabet | []byte, error
 
### example_test.go  
#### 测试用例  
 序号 | 方法名 | 作用 | 测试结果 
---|---|---|---
 1 | TestExample_basic | 通过加密和解密，测试base58的可用性 | 可以对加密后的数据进行还原，并输出加密前后的数据以供对比；若数据被篡改，会输出错误提示信息

