[TOC]
# 概述
这是elasticsearch的客户端SDK

# 快速开始
```
import es "github.com/sea-project/sea-pkg/elasticsearch"

client, err := es.Dial("http://127.0.0.1:9200/")
if err != nil {
	fmt.Println(err)
    return
}
indexName := "userinfo"
result, err := client.QueryIndexMappingInfo(indexName)
if err != nil {
	fmt.Println(err.Error())
    return 
}
fmt.Println("result:%v", result)
```

# 接口方法列表
**测试代码位置**

```shell
./client_test.go
```
## 1.创建索引
**函数原型**

```
func CreateIndex(indexName string, requestBody string) error
```

**参数说明**

参数     | 类型   | 说明
-------- | ------ | --------
indexName | string | 索引名称
requestBody | string | http请求参数body内容

**返回值说明**

参数 | 类型   | 说明     
---- | ----   | --- |
0    | error  | 错误信息(nil为创建成功，其它均为失败) 

**使用示例**

```
client, _ := NewClient("http://127.0.0.1:9200/")
err := client.CreateIndex("xzy100218", `{
	"mappings": {
		"properties": {
			"name": {
				"type": "keyword"
			},
			"sex": {
				"type": "boolean"
			},
			"age": {
				"type": "short"
			}
		}
	},
	"settings": {
		"index": {
			"number_of_shards": 1,
			"number_of_replicas": 1
		}
	}
}`)
if err != nil {
    fmt.Println(err.Error())
}
```

## 2.查询索引的mapping信息
**函数原型**

```
func QueryIndexMappingInfo(indexName string) (string, error)
```

**参数说明**

参数     | 类型   | 说明
-------- | ------ | --------
indexName | string | 索引名称

**返回值说明**

参数 | 类型   | 说明     
---- | ----   | --- |
0    | string | 索引的mapping信息(是所有字段对象json序列化后的字符串)
1    | error  | 错误信息(nil为查询成功，其它均为失败) 

**使用示例**

```
client, _ := NewClient("http://127.0.0.1:9200/")
indexName := "userinfo"
result, err := client.QueryIndexMappingInfo(indexName)
if err != nil {
    fmt.Println(err.Error())
    return
}
fmt.Println("result:%v", result)
```

## 3.删除索引
**函数原型**

```
func DeleteIndex(indexName string) error
```

**参数说明**

参数     | 类型   | 说明
-------- | ------ | --------
indexName | string | 索引名称

**返回值说明**

参数 | 类型   | 说明     
---- | ----   | --- |
0    | error  | 错误信息(nil为删除成功，其它均为失败) 

**使用示例**

```
client, _ := NewClient("http://127.0.0.1:9200/")
indexName := "userinfo"
result, err := client.DeleteIndex(indexName)
if err != nil {
    fmt.Println(err.Error())
    return
}
```

## 4.添加记录
**函数原型**

```
func AddRecord(indexName string, requestBody string) error
```

**参数说明**

参数     | 类型   | 说明
-------- | ------ | --------
indexName | string | 索引名称
requestBody | string | 记录对象json序列化后得到的字符串

**返回值说明**

参数 | 类型   | 说明     
---- | ----   | --- |
0    | error  | 错误信息(nil为成功，其它均为失败) 

**使用示例**

```
client, _ := NewClient("http://127.0.0.1:9200/")
indexName := "userinfo"
result, err := client.AddRecord(indexName, "{  \"name\": \"xtt\",  \"sex\": true,  \"age\": 30}")
if err != nil {
    fmt.Println(err.Error())
    return
}
```

## 4.批量添加记录
**函数原型**

```
func BatchAddRecord(indexName string, requestBody []string) (int, error)
```

**参数说明**

参数     | 类型   | 说明
-------- | ------ | --------
indexName | string | 索引名称
requestBody | []string | 每条记录对象json序列化后得到的字符串数组

**返回值说明**

参数 | 类型   | 说明     
---- | ----   | --- |
0    | int    | 添加记录成功条数
1    | error  | 错误信息(nil为成功，其它均为失败) 

**使用示例**

```
client, _ := NewClient("http://127.0.0.1:9200/")
indexName := "userinfo"
body := make([]string, 0)
body = append(body, `{ "name": "xxx", "sex": true, "age": 29}`)
body = append(body, `{ "name": "yyy", "sex": false, "age": 30}`)
n, err := client.BatchAddRecord(indexName, body)
if err != nil {
    fmt.Println(err.Error())
    return
}
fmt.Println("%v", n)
```


## 5.根据条件更新记录
**函数原型**

```
func UpdateRecord(indexName string, requestBody string) (int, error)
```

**参数说明**

参数     | 类型   | 说明
-------- | ------ | --------
indexName | string | 索引名称
requestBody | []string | 查询条件及其修改规则对象json序列化后得到的字符串

**返回值说明**

参数 | 类型   | 说明     
---- | ----   | --- |
0    | int    | 修改记录成功条数
1    | error  | 错误信息(nil为成功，其它均为失败) 

**使用示例**

```
client, _ := NewClient("http://127.0.0.1:9200/")
indexName := "userinfo"
n, err := client.UpdateRecord(indexName, `{  "query": {     "match": {      "_id": "uX2aR3QBKSuPjy8yQPSJ"    }  },  "script": {    "source": "ctx._source.age = 36"  }}`)
if err != nil {
    fmt.Println(err.Error())
    return
}
fmt.Println("%v", n)
```

## 6.根据条件查询记录
**函数原型**

```
func QueryRecord(indexName string, requestBody string) (*model.RetQueryRecord, error)
```

**参数说明**

参数     | 类型   | 说明
-------- | ------ | --------
indexName | string | 索引名称
requestBody | string | 查询条件对象json序列化后得到的字符串

**返回值说明**

参数 | 类型   | 说明     
---- | ----   | --- |
0    | model.RetQueryRecord    | 查询返回的记录信息
1    | error  | 错误信息(nil为查询成功，其它均为失败) 

**使用示例**

```
client, _ := NewClient("http://127.0.0.1:9200/")
indexName := "userinfo"
retQueryRecord, err := client.QueryRecord(indexName, `{  "query": {    "match_all": {}  },  "from": 1,  "size": 10}`)
if err != nil {
    fmt.Println(err.Error())
    return
}
fmt.Println("%v", retQueryRecord)
```


## 7.删除指定记录
**函数原型**

```
func DeleteRecord(indexName string, id string) error
```

**参数说明**

参数     | 类型   | 说明
-------- | ------ | --------
indexName | string | 索引名称
id | string | 记录的id

**返回值说明**

参数 | 类型   | 说明     
---- | ----   | --- |
0    | error  | 错误信息(nil为删除成功，其它均为失败) 

**使用示例**

```
client, _ := NewClient("http://127.0.0.1:9200/")
indexName := "userinfo"
id := "4H3CTnQBKSuPjy8yifQ8"
err := client.DeleteRecord(indexName, id)
if err != nil {
    fmt.Println(err.Error())
    return
}
```