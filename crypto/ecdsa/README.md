[![Travis CI](https://travis-ci.org/shengdoushi/base58.svg?branch=master)](http://10.1.1.28/go/crypto)
[![GoDoc](https://www.godoc.org/github.com/shengdoushi/base58?status.svg)](http://10.1.1.28/go/crypto/wikis/crypto%E6%A8%A1%E5%9D%97README)
[![Go Report Card](https://goreportcard.com/badge/github.com/shengdoushi/base58)](https://goreportcard.com/report/github.com/shengdoushi/base58)

[^1]:上面三个图标在每个README里面可有可不有，但如果项目在GitHub或固定网站上，或是你的项目包引用了网上资源，请使用这三个图标嵌入引用或说明链接地址。
[^1]:passing对应着项目所在网址,reference对应着参考网址，report对应着文档地址

包btcec实现了工作所需的椭圆曲线加密
比特币(secp256k1暂不适用)。它的设计目的是使它可以与
go附带的标准加密/ecdsa包。一套全面的测试
提供以确保适当的功能。

虽然这个包最初是为btcd编写的，但它是故意为btcd编写的
设计成可以作为任何需要的项目的独立包使用
使用secp256k1椭圆曲线密码学。


## 代码走读

### 各文件方法表
 序号 | Go文件/函数或方法 | 作用 
---|---|---
 1 | btcec.go | 该文件主要提缩短操作变量、实现secp256k1的曲线的方法
     &nbsp; | `NAF` | NAF取一个正整数k，并以两个字节片的形式返回非相邻表单(NAF)，使得最小化操作数量成为可能，因为返回的结果int至少为50%。
     &nbsp;| ==`S256`== | S256会返回一条实现secp256k1的曲线
 2 | ecdsa.go  | 主要提供了生成公私钥对、签名转公钥、使用hash和签名对公钥进行序列化、公钥转地址等方法
      &nbsp;| ==`PubKey`== | 私钥返公钥
      &nbsp;| ==`ToECDSA`== | 私钥返ecdsa私钥。
      &nbsp;| ==`Sign`== | 私钥签名
      &nbsp;| `Serialize` | 私钥序列化，==暂时未调用==
      &nbsp;| ==`IsEqual`==| 公钥对比
      &nbsp;| ==`PubkeyToAddress`== | 公钥转地址
      &nbsp;| `SerializeCompressed` | 序列化公钥为33字节的压缩格式
      &nbsp;| `SerializeHybrid` | 序列化公钥为65字节的混合格式
      &nbsp;| `SerializeUncompressed` | 序列化公钥为65字节的未压缩格式
      &nbsp;| ==`ToECDSA`== | 公钥返ecdsa公钥
      &nbsp;| ==`Decrypt`== | 私钥解密。
      &nbsp;| ==`DoubleHashB`== | 计算Hash，返回32位的固定字符串。
      &nbsp;| ==`Encrypt`== | 公钥加密
      &nbsp;| `FromECDSAPub` | 椭圆加密公钥转坐标，作用是为公钥转地址方法提供支持
      &nbsp;| ==`GenerateKeyPair`== | 生成公私钥对
      &nbsp;| `GenerateSharedSecret` | 基于公私钥生成共享密钥
      &nbsp;| ==`Keccak256`== | sha3 256加密内容
      &nbsp;| ==`ParsePubKey`== | 验证公钥是否有效
      &nbsp;| `PrivKeyFromBytes` | 根据传入的byte数组(密码)和椭圆曲线，返回一个公私钥
      &nbsp;| ==`UnmarshalPubkey`== | 将公钥的byte[]转换为secp256k1公钥。
      &nbsp;| `Keccak256Hash` | Keccak256Hash计算并返回输入数据的Keccak256散列，将其转换为内部散列数据结构，==暂时未调用==。
 3 | field.go  | 该文件主要提供了基于fieldVal结构体的一系列和算法有关的方法，该文件的作用是精度算法来提高性能，如果你没有很好的算法基础，建议你了解一下这个包的作用就可以了。
 4 | genprecomps.go  | 此文件在常规构建过程中由于以下构建标记而被忽略。它由go generate调用，用于自动生成用于加速操作的预计算表。
 5 | gensecp256k1.go  | 用于生成secp256k1.go文件
 6 | precompute.go  | 提供了包内可用调用的loadS256BytePoints方法，作用是用于加速secp256k1曲线标量基乘法的预计算字节点进行解压缩和反序列化，从而使用这种方法在init时生成内存中的数据结构非常快。
 7 | secp256k1.go  | 作用为在生成椭圆曲线时提供加速，由gensecp256k1文件生成
 8 | signature.go  | 提供了对签名的序列化、验签、签名对比等方法。
     &nbsp;| `IsEqual` | 将签名的实例和传入的签名进行对比，==暂时未调用==
     &nbsp;| ==`Serialize`== | 对签名进行序列化
     &nbsp;| ==`Verify`== | 通过调用ecdsa的公钥来验证哈希的签名是否正确
     &nbsp;| `RecoverCompact` | 验证压缩签名，正确就返回公钥，错误则返回错误信息
     &nbsp;| ==`SignCompact`== | 使用自定的私钥生成压缩签名
 

 
#### 单元测试 


序号 | Go文件/测试用例方法 | 说明
---|---|---     
 1 | ecdsa_test.go  | 主要测试了包含创建公私钥、加密解密、加签验签等流程
    &nbsp;| `Test_Sign` | 测试私钥加签
      &nbsp;| `Test_ToECDSA` | 私钥转ecdsa的私钥
      &nbsp;| `Test_PubKey` | 私钥返回对应的公钥
      &nbsp;| `Benchmark_Sign` | 对加签进行压测
     &nbsp;| `Test_GenerateKeyPair` | 测试生成公私钥对
     &nbsp;| `Test_PubkeyToAddress` | 测试公钥转地址、
      &nbsp;| `Test_Verify` | 测试通过公钥和hash对签名进行验签
      &nbsp;| `Test_ParsePubKey` | 测试验证公钥是否有效
      &nbsp;| `Test_SignAndVerify` | 私钥加签，用验证后的公钥进行解签，hash由一个hash算法生成。
      &nbsp;| `Test_EncryptAndDecode` | 测试公钥进行加密，私钥进行解密
      &nbsp;| `Benchmark_Verify` | 对加签和验签进行压测
      &nbsp;| `Benchmark_SerializeCompressed` | 对序列化一个33字节的公钥进行压测
      &nbsp;| `Benchmark_SerializeHybrid` | 对序列化一个65字节混合格式的公钥进行压测
      &nbsp;| `Benchmark_SerializeUncompressed` | 对序列化一个65字节的未压缩的公钥进行压测
     &nbsp;| `Test_SigToPub` | 测试签名转公钥(只对以太坊的有效)
     &nbsp;| `TestEcrecover` | 使用hsah和签名转换后的公钥，进行序列化(只对以太坊的有效)
     &nbsp;| `Test_Keccak256` | 测试sha3 256加密内容
     &nbsp;| `Test_FromECDSAPub` | 测试椭圆加密公钥转坐标
     &nbsp;| `Test_Decode` | 测椭圆加密验证 - 对x,y轴的值是否存在曲线上进行验证，待验证
     &nbsp;| `Test_SignAndVerify` | 测试使用secp256k1私钥对消息进行签名，该私钥首先从原始字节解析并序列化生成的签名。
     &nbsp;| `Test_UnmarshalPubkey` | 测试通过hash和签名反向推出公钥
     &nbsp;| `Test_Flow` | 测试整体流程
     &nbsp;| `Test_Flow2` | 测试整体流程，并采用第二种加签验签的方式
     &nbsp;| `Test_Flow3` | 测试整体流程，并采用第二种加签验签的方式，并使用通过数据和hash解析出的公钥来进行验签
