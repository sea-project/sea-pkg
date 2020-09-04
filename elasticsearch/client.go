package elasticsearch

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Client 客户端对象
type Client struct {
	addr string
}

// NewClient 创建一个新的实例
func NewClient(addr string) (*Client, error) {
	if len(addr) == 0 {
		return nil, fmt.Errorf("parameter error")
	}
	s := new(Client)
	s.addr = addr
	return s, nil
}

// CreateIndex 创建索引
func (c *Client) CreateIndex(indexName string, requestBody string) error {
	// 参数判断
	if len(indexName) == 0 {
		return fmt.Errorf("parameter indexName error")
	}
	if len(requestBody) == 0 {
		return fmt.Errorf("parameter requestBody error")
	}

	// http请求
	result := new(ResponseCreateIndex)
	err := HTTPRequest("PUT", c.addr+indexName, requestBody, OKCode, result)
	if err != nil {
		return err
	}
	if !result.Acknowledged {
		return fmt.Errorf("create index fail unknown error")
	}
	return nil
}

// QueryIndexMappingInfo 查询索引的mapping信息
func (c *Client) QueryIndexMappingInfo(indexName string) (string, error) {
	// 参数判断
	if len(indexName) == 0 {
		return "", fmt.Errorf("QueryIndexMappingInfo parameter error")
	}

	// http请求
	var result map[string]ResponseQueryIndex
	err := HTTPRequest("GET", c.addr+indexName, "", OKCode, &result)
	if err != nil {
		return "", err
	}

	// 获取返回数据返回
	for name, value := range result {
		if strings.EqualFold(name, indexName) {
			str, _ := json.Marshal(value.Mappings.Properties)
			return string(str), nil
		}
	}
	return "", fmt.Errorf("QueryIndexMappingInfo response error")
}

// DeleteIndex 删除索引
func (c *Client) DeleteIndex(indexName string) error {
	// 参数判断
	if len(indexName) == 0 {
		return fmt.Errorf("parameter indexName error")
	}

	// http请求
	result := new(ResponseDeleteIndex)
	err := HTTPRequest("DELETE", c.addr+indexName, "", OKCode, result)
	if err != nil {
		return err
	}
	// 通过判断Acknowledged字段判断是否创建成功
	if !result.Acknowledged {
		return fmt.Errorf("delete index fail unknown error")
	}
	return nil
}

// AddRecord 添加记录
func (c *Client) AddRecord(indexName string, requestBody string) error {
	// 参数判断
	if len(indexName) == 0 {
		return fmt.Errorf("parameter indexName error")
	}
	if len(requestBody) == 0 {
		return fmt.Errorf("parameter requestBody error")
	}
	result := new(ResponseAddRecord)
	err := HTTPRequest("POST", c.addr+indexName+"/_doc", requestBody, 201, result)
	if err != nil {
		return fmt.Errorf("http.NewRequest err:%v", err.Error())
	}
	return nil
}

func buildBatchAddRecordReqParam(indexName string, requestBody []string) (string, error) {
	requestFromBatchAddRecord := new(RequestFromBatchAddRecord)
	requestFromBatchAddRecord.Index.Index = indexName
	requestIndex, err := json.Marshal(requestFromBatchAddRecord)
	if err != nil {
		return "", fmt.Errorf("json.Marshal err:%v", err.Error())
	}
	param := ""
	for i := 0; i < len(requestBody); i++ {
		param += string(requestIndex)
		param += "\n"
		param += requestBody[i]
		param += "\n"
	}
	return param, nil
}

// BatchAddRecord 批量添加记录
func (c *Client) BatchAddRecord(indexName string, requestBody []string) (int, error) {
	// 参数判断
	if len(indexName) == 0 {
		return 0, fmt.Errorf("parameter indexName error")
	}
	if len(requestBody) == 0 {
		return 0, fmt.Errorf("parameter requestBody error")
	}

	// 封装请求参数
	param, err := buildBatchAddRecordReqParam(indexName, requestBody)
	if err != nil {
		return 0, nil
	}

	// http请求
	result := new(ResponseBatchAddRecord)
	err = HTTPRequest("POST", c.addr+"_bulk", param, OKCode, result)
	if err != nil {
		return 0, err
	}

	// 处理请求返回数据然后返回给上层
	num := 0
	// 判断是否有未插入成功的
	if result.Errors {
		// 有插入未成功的，则判断插入成功条数
		for _, v := range result.Items {
			if len(v.Index.Result) != 0 {
				num++
			}
		}
		return num, nil
	}
	return len(requestBody), nil
}

// UpdateRecord 更新记录
func (c *Client) UpdateRecord(indexName string, requestBody string) (int, error) {
	// 参数判断
	if len(indexName) == 0 {
		return 0, fmt.Errorf("parameter indexName error")
	}
	if len(requestBody) == 0 {
		return 0, fmt.Errorf("parameter requestBody error")
	}
	// http请求
	result := new(ResponseUpdateRecord)
	err := HTTPRequest("POST", c.addr+indexName+"/_update_by_query", requestBody, OKCode, result)
	if err != nil {
		return 0, err
	}
	return result.Total, nil
}

// QueryRecord 查询记录
func (c *Client) QueryRecord(indexName string, requestBody string) (*RetQueryRecord, error) {
	// 参数判断
	if len(indexName) == 0 {
		return nil, fmt.Errorf("parameter indexName error")
	}
	if len(requestBody) == 0 {
		return nil, fmt.Errorf("parameter requestBody error")
	}

	// http请求
	result := new(ResponseQueryRecord)
	err := HTTPRequest("GET", c.addr+indexName+"/_search", requestBody, OKCode, result)
	if err != nil {
		return nil, err
	}

	// 处理请求返回信息然后返回给上层一定格式的数据
	retQueryRecord := new(RetQueryRecord)
	retQueryRecord.TotalValue = result.Hits.Total.Value
	for _, v := range result.Hits.Hits {
		perRecordInfo := new(PerRecordInfo)
		perRecordInfo.ID = v.ID
		str, _ := json.Marshal(v.Source)
		perRecordInfo.Source = string(str)
		retQueryRecord.RecordInfo = append(retQueryRecord.RecordInfo, perRecordInfo)
	}
	return retQueryRecord, nil
}

// DeleteRecord 删除指定记录
func (c *Client) DeleteRecord(indexName string, id string) error {
	// 参数判断
	if len(indexName) == 0 {
		return fmt.Errorf("parameter indexName error")
	}
	if len(id) == 0 {
		return fmt.Errorf("parameter id error")
	}

	result := new(ResponseDeleteRecord)
	err := HTTPRequest("DELETE", c.addr+indexName+"/_doc/"+id, "", OKCode, result)
	if err != nil {
		return fmt.Errorf("http.NewRequest err:%v", err.Error())
	}

	// 删除成功的标志是该字段为deleted
	if result.Result != "deleted" {
		return fmt.Errorf("DeleteRecord err:%v", result.Result)
	}
	return nil
}
