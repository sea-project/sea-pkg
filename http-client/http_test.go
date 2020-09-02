package http_client

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func Test_HP_Get(t *testing.T) {
	hp := NewHttpPool(0, 0, 3)
	header := make(map[string]string)
	res, err, statusCode := hp.Request("https://IP", http.MethodGet, "", header)
	if err != nil {
		t.Log(err)
	}
	t.Log("res：", res, "StatusCode:", statusCode)
}

func Test_HP_Post(t *testing.T) {
	hp := NewHttpPool(0, 0, 3)
	header := make(map[string]string)
	header["content-type"] = "application/json"
	data := ``
	res, err, statusCode := hp.Request("https://IP", http.MethodPost, data, header)
	if err != nil {
		t.Log(err)
	}
	t.Log("res：", res, "StatusCode:", statusCode)
}

func Test_HP_Delete(t *testing.T) {
	hp := NewHttpPool(0, 0, 3)
	header := make(map[string]string)
	data := ``
	res, err, statusCode := hp.Request("http://IP", http.MethodDelete, data, header)
	if err != nil {
		t.Log(err)
	}
	t.Log("res：", res, "StatusCode:", statusCode)
}

func Test_BATCH_HttpConPool_Request(t *testing.T) {
	hpool := NewHttpPool(8, 8, 3)
	for range time.Tick(time.Second) {
		for i := 0; i < 1000; i++ {
			go tt(hpool)
		}
	}

	<-make(chan int)
}

func tt(hpool *HttpConPool) {
	_, err, statusCode := hpool.Request("http://IP/", http.MethodGet, "", map[string]string{})
	if err != nil {
		fmt.Println("err:", err, "StatusCode:", statusCode)
	}
	//fmt.Println(resp,err)
}
