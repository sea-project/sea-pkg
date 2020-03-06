[![Travis CI](https://travis-ci.org/shengdoushi/base58.svg?branch=master)](http://10.1.1.28/go/crypto)
[![GoDoc](https://www.godoc.org/github.com/shengdoushi/base58?status.svg)](http://10.1.1.28/go/crypto/wikis/crypto%E6%A8%A1%E5%9D%97README)
[![Go Report Card](https://goreportcard.com/badge/github.com/shengdoushi/base58)](https://goreportcard.com/report/github.com/shengdoushi/base58)

[^1]:上面三个图标在每个README里面可有可不有，但如果项目在GitHub或固定网站上，或是你的项目包引用了网上资源，请使用这三个图标嵌入引用或说明链接地址。
[^1]:passing对应着项目所在网址,reference对应着参考网址，report对应着文档地址


## 代码走读

### 各文件方法表
 序号 | Go文件/函数或方法 | 作用 
---|---|---
 1 | leveldb.go | 该文件主要实现数据库操作方法
 &nbsp; | `Init` | 数据库连接初始化操作
 &nbsp; | `Put`  | 存储键值对
 &nbsp; | `Get`  | 根据键获取值
 &nbsp; | `Del`  | 根据键删除键值对
 &nbsp; | `Has`  | 判断键是否存储
 &nbsp; | `Path`  |获取数据目录地址
 &nbsp; | `GetDB`  | 获取数据库连接句柄
 &nbsp; | `Close`  | 关闭数据库
 &nbsp; | `NewIterator`  | 批量获取键
 &nbsp; | `NewBatch`  | 批量存储初始化
 &nbsp; | `Put`   | 写入暂存区
 &nbsp; | `Save`  | 提交写入暂存区的数据
 &nbsp; | `Size`  | 获取暂存区存储值的总长度
 
 
#### 单元测试 


序号 | Go文件/测试用例方法 | 说明
---|---|---
 1 | leveldb_test.go | 数据库操作测试用例
&nbsp; | `Test_Put`  | 存储测试用例
&nbsp; | `Test_Get`  | 获取测试用例
&nbsp; | `Test_Has`  | 是否存在测试用例
&nbsp; | `Test_Del`  | 删除测试用例
&nbsp; | `Test_Path` | 获取数据库目录测试用例
&nbsp; | `TestLdbBatch_Put`  | 批量存储测试用例
&nbsp; | `BenchmarkLdbBatch_Put` | 批量存储压测
