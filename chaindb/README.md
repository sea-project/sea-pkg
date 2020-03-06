[![Travis CI](https://travis-ci.org/shengdoushi/base58.svg?branch=master)](http://10.1.1.28/go/crypto)
[![GoDoc](https://www.godoc.org/github.com/shengdoushi/base58?status.svg)](http://10.1.1.28/go/crypto/wikis/crypto%E6%A8%A1%E5%9D%97README)
[![Go Report Card](https://goreportcard.com/badge/github.com/shengdoushi/base58)](https://goreportcard.com/report/github.com/shengdoushi/base58)

[^1]:上面三个图标在每个README里面可有可不有，但如果项目在GitHub或固定网站上，或是你的项目包引用了网上资源，请使用这三个图标嵌入引用或说明链接地址。
[^1]:passing对应着项目所在网址,reference对应着参考网址，report对应着文档地址

# 目录
 - [文件夹目录概要](#文件夹目录概要)

# 文件夹目录概要

文件夹目录：    

 序号 | 文件夹/go文件 | 作用 
---|-------|---
 1 | leveldb | leveldb数据库，分布式存储引擎，主要用于键值对儿的持久化文件存储。
 &nbsp;| `leveldb.go` | 封装数据库常用操作方法
 &nbsp;| `leveldb_test.go` | 数据库操作测试用例
 2 | memorydb | 基于内存的数据库引擎，主要用于键值对儿的临时存储。
 &nbsp;| `memorydb.go` | 封装了数据库常用操作方法
 &nbsp;| `memorydb_test.go` | 数据库操作测试用例
 3 | statedb | 基于内存的账户信息存储引擎，主要用于map类型的数据封装存储。
 &nbsp;| `statedb.go` | 封装了基于账户信息的数据操作接口
 &nbsp;| `objects.go` | 账户信息结构体以及结构体方法
 &nbsp;| `statedb_test.go` | 账户信息操作测试用例
 
