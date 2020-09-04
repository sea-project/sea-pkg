package elasticsearch

import (
	"fmt"
	"testing"
	"time"
)

func TestClient_CreateIndex(t *testing.T) {
	client, _ := NewClient("http://IP:9200/")
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
		t.Logf(err.Error())
	}
}

func TestClient_QueryIndexMappingInfo(t *testing.T) {
	client, _ := NewClient("http://IP:9200/")
	indexName := "xzy090208"
	result, err := client.QueryIndexMappingInfo(indexName)
	if err != nil {
		t.Logf(err.Error())
	}
	t.Logf("result:%v", result)
}

func TestClient_DeleteIndex(t *testing.T) {
	client, _ := NewClient("http://IP:9200/")
	indexName := "xzy090208"
	err := client.DeleteIndex(indexName)
	if err != nil {
		t.Logf(err.Error())
	}
}

func TestClient_AddRecord(t *testing.T) {
	client, _ := NewClient("http://IP:9200/")
	indexName := "userinfo"
	err := client.AddRecord(indexName, "{  \"name\": \"xtt\",  \"sex\": true,  \"age\": 30}")
	if err != nil {
		t.Logf(err.Error())
	}
}

func TestClient_BatchAddRecord(t *testing.T) {
	client, _ := NewClient("http://IP:9200/")
	indexName := "userinfo"
	body := make([]string, 0)
	body = append(body, `{ "name": "xxx", "sex": true, "age": 29}`)
	body = append(body, `{ "name": "yyy", "sex": false, "age": 30}`)
	n, err := client.BatchAddRecord(indexName, body)
	if err != nil {
		t.Logf(err.Error())
	}
	t.Logf("%v", n)
}

func TestClient_BatchAddRecordTest(t *testing.T) {
	client, _ := NewClient("http://IP:9200/")
	indexName := "userinfo"

	for {
		body := make([]string, 0)
		for i := 0; i < 1000; i++ {
			body = append(body, `{ "name": "xxx", "sex": true, "age": 29}`)
		}
		starttime := time.Now().UnixNano() / 1e6
		n, err := client.BatchAddRecord(indexName, body)
		if err != nil {
			t.Logf(err.Error())
		}
		fmt.Println("耗时：", (time.Now().UnixNano()/1e6)-starttime, "增加交易：", n)
	}
}

func TestClient_UpdateRecord(t *testing.T) {
	client, _ := NewClient("http://IP:9200/")
	indexName := "userinfo"
	n, err := client.UpdateRecord(indexName, `{  "query": {     "match": {      "_id": "uX2aR3QBKSuPjy8yQPSJ"    }  },  "script": {    "source": "ctx._source.age = 36"  }}`)
	if err != nil {
		t.Logf(err.Error())
	}
	t.Logf("%v", n)
}

func TestClient_QueryRecord(t *testing.T) {
	client, _ := NewClient("http://IP:9200/")
	indexName := "userinfo"
	retQueryRecord, err := client.QueryRecord(indexName, `{  "query": {    "match_all": {}  },  "from": 1,  "size": 10}`)
	if err != nil {
		t.Logf(err.Error())
	}
	t.Logf("%v", retQueryRecord.String())
}

func TestClient_DeleteRecord(t *testing.T) {
	client, _ := NewClient("http://IP:9200/")
	indexName := "userinfo"
	id := "333CTnQBKSuPjy8yifQ8"
	err := client.DeleteRecord(indexName, id)
	if err != nil {
		t.Logf(err.Error())
	}
}
