[![Travis CI](https://travis-ci.org/shengdoushi/base58.svg?branch=master)](http://10.1.1.28/go/crypto)
[![GoDoc](https://www.godoc.org/github.com/shengdoushi/base58?status.svg)](http://10.1.1.28/go/crypto/wikis/crypto%E6%A8%A1%E5%9D%97README)
[![Go Report Card](https://goreportcard.com/badge/github.com/shengdoushi/base58)](https://goreportcard.com/report/github.com/shengdoushi/base58)

[^1]:上面三个图标在每个README里面可有可不有，但如果项目在GitHub或固定网站上，或是你的项目包引用了网上资源，请使用这三个图标嵌入引用或说明链接地址。
[^1]:passing对应着项目所在网址,reference对应着参考网址，report对应着文档地址

# 目录
 - [地址生成的过程](#地址生成的过程)
 - [文件夹目录概要](#文件夹目录概要)

# 地址生成的过程  

![image](https://github.com/SurvivalBoy/StaticResource/blob/master/images/Go/%E5%8C%BA%E5%9D%97%E9%93%BE%E5%9C%B0%E5%9D%80%E7%94%9F%E6%88%90%20.png?raw=true)

一个区块链地址生成过程：
1. 由secp256k1曲线生成私钥，是由随机的256bit组成
2. 采用椭圆曲线数字签名算法（ECDSA）将私钥映射成公钥。
3. 公钥经过Keccak-256单向散列函数变成了256bit，然后取160bit作为地址

# 文件夹目录概要

文件夹目录：    

 序号 | 文件夹/go文件 | 作用 
---|---|---
 1 | base58 | 该文件夹主要提供对字母表定义的方法、对数据加密、解密的方法；以及各种单元测试 
     &nbsp; | ==`base58.go`== | 提供了创建字母表、根据字母表加密、根据字母表解密的方法。
     &nbsp;| `base58_test.go` | 提供了测试创建字母表、字母表长度校验、双字节验证加密和解密、字母表加密、解密、压测加密和解密等测试用例
&nbsp;| `example_test.go` | 提供了通过使用自定义的字母表对数据进行加密和解密以及数据对比的完整测试用例
 2 | ecdsa | 该文件夹主要提供了基于比特币的一些公钥私钥创建方法、加密、解密、加签、验签等函数 
     &nbsp;| `btcec.go` | 该文件主要提缩短操作变量、实现secp256k1的曲线的方法
     &nbsp;| `field.go` | 该文件主要提供了基于fieldVal结构体的一系列和算法有关的方法，该文件的作用是精度算法来提高性能，如果你没有很好的算法基础，建议你了解一下这个包的作用就可以了。
     &nbsp;| `precompute.go` | 提供了包内可用调用的loadS256BytePoints方法，作用是用于加速secp256k1曲线标量基乘法的预计算字节点进行解压缩和反序列化，从而使用这种方法在init时生成内存中的数据结构非常快。
     &nbsp;| ==`signature.go`== | 提供了对签名的序列化、验签、签名对比等方法。
     &nbsp;| `signature_test.go` | 模拟发送和接收方，使用公私钥加签验签加密解密的过程
     &nbsp;| ==`ecdsa.go`== | 提供了生成公私钥、加密解密、加签验签、公钥对比、签名转地址、签名公私钥序列化、公钥转地址等重要方法。
     &nbsp;| ==`ecdsa_test.go`== | 提供了对包括生成公私钥、加密解密、加签验签、公钥对比、签名转地址、签名公私钥序列化、公钥转地址等重要方法的测试用例和压力测试。
 3 | sha3 | 实现了SHA-3固定输出长度哈希函数和FIPS-202定义的SHAKE可变输出长度哈希函数。
     &nbsp;| `keccakf.go` | 仅提供了一个内部调用的切片方法，实际上是加密算法的实现，属于底层代码，算法弱的不推荐关注
     &nbsp;| `keccakf_amd64.go` | 提供了一个内部函数，该函数在keccakf_amd64.s中实现。
     &nbsp;| ==`sha3.go`== | 提供了包括Hsah常用方法、创建SHA-3和SHAKE散列函数实例的函数。
     &nbsp;| `sha3_test.go` | 提供了一系列包含测试SHA-3和Shake实现、将数据写入具有较小输入缓冲区的任意模式、各种Hash加密方式的压力测试等测试用例。
     &nbsp;| `xor_Unaligned.go` | 提供了非对称的异或运算的内部方法
     
# 国密算法与原有crypo包中算法的性能比较
  
## SM2与椭圆曲线加密算法比较  
ECDSA椭圆曲线加密算法性能值| SM2国密算法性能值 |说明
---|---|---
S256( 300000000	&nbsp; 4.70 ns/op) | P256Sm2( 20000000	&nbsp;   126 ns/op) | 返回一条实现secp256k1的曲线
GenerateKey( 100000 &nbsp; 18930 ns/op) | GenerateKey( 5000	&nbsp;   284952 ns/op) | 返回一条实现secp256k1的曲线
PubKey( 5000000000 &nbsp; 0.4 ns/op(通过代码多次循环测试结果为0.57ns/op)) | Public (2000000000  &nbsp; 0.40 ns/op (通过代码多次循环测试0.53ns/op)) | 私钥返公钥
Sign( 20000 &nbsp; 80213 ns/op) | Sign( 5000 &nbsp;  326674 ns/op) | 私钥签名
SerializeCompressed( 20000000 &nbsp; 128 ns/op) | Compress( 5000000 &nbsp; 286 ns/op) | 序列化公钥为33字节的压缩格式
Decrypt( 30000000 &nbsp; 102 ns/op) | Decrypt( 1000 &nbsp; 1312490 ns/op) | 私钥解密
Encrypt(10000 &nbsp; 187094 ns/op) | Encrypt( 1000 &nbsp; 1595242 ns/op) | 公钥加密
  
## SM3与各种哈希算法比较  
序号| 常用方法名称 | 性能值 | 说明 
---|---|---|---
sha系列算法  | sm3国密算法 |  说明  
New256（200000000	  &nbsp;    9.63 ns/op ）| New（200000000	   &nbsp;    7.00 ns/op）  | 创建一个新的256位散列
Sum （500000	  &nbsp;    2580 ns/op  ） | Sum （2000000	   &nbsp;    3126 ns/op ）   | 返回哈希算法摘要值

> 注：为了便于快速查找，本文档中标黄的为对外提供公用方法的通用文件