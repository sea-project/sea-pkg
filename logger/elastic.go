package logger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"strconv"
	"strings"
	"sync"
	"time"
)

type elasticLogger struct {
	Addr     string `json:"addr"`
	Index    string `json:"index"`
	Level    string `json:"level"`
	LogLevel int
	Open     bool
	Es       *elasticsearch.Client
	Mu       sync.RWMutex
}

type MsgBody struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Path    string `json:"path"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type ElasticLogBody struct {
	Level     string    `json:"level"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	TimeStamp time.Time `json:"timestamp"`
}

// Init 初始化
func (e *elasticLogger) Init(jsonConfig string) error {
	if len(jsonConfig) == 0 {
		return nil
	}

	err := json.Unmarshal([]byte(jsonConfig), &e)
	if err != nil {
		return err
	}

	if e.Open == false {
		return nil
	}

	if lv, ok := LevelMap[e.Level]; ok {
		e.LogLevel = lv
	}
	err = e.connectElastic()
	if err != nil {
		return err
	}
	return nil
}

// LogWrite 写操作
func (e *elasticLogger) LogWrite(when time.Time, msgText interface{}, level int) error {

	if level > e.LogLevel {
		return nil
	}

	msg, ok := msgText.(string)
	if !ok {
		return nil
	}

	if e.Es == nil {
		err := e.connectElastic()
		if err != nil {
			return err
		}
	}

	body := new(MsgBody)
	err := json.Unmarshal([]byte(msg), &body)
	if err != nil {
		return err
	}

	esBody := new(ElasticLogBody)
	esBody.Name = body.Name
	esBody.Level = body.Level
	esBody.Content = body.Content
	esBody.Path = body.Path

	// 必须转换为时间格式，否则es不支持
	timeTemplate := "2006-01-02 15:04:05"
	stamp, err := time.ParseInLocation(timeTemplate, body.Time, time.Local)
	if err != nil {
		return err
	}
	esBody.TimeStamp = stamp.UTC()
	esByte, _ := json.Marshal(esBody)
	go e.saveMessage(string(esByte))
	return nil
}

// Destroy 销毁
func (e *elasticLogger) Destroy() {
	e.Es = nil
}

// connectElastic 链接elasticsearch
func (e *elasticLogger) connectElastic() (err error) {
	cfg := elasticsearch.Config{Addresses: []string{e.Addr}}
	e.Es, err = elasticsearch.NewClient(cfg)
	if err != nil {
		return errors.New(fmt.Sprintf("Get elastic client error %v", err))
	}
	return nil
}

// saveMessage 存储日志到服务器
func (e *elasticLogger) saveMessage(msg string) error {
	dateTime := strconv.FormatInt(time.Now().UnixNano(), 10)
	req := esapi.IndexRequest{
		Index:      e.Index,
		DocumentID: dateTime,
		Body:       strings.NewReader(msg),
		Refresh:    "true",
	}
	res, err := req.Do(context.Background(), e.Es)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

func init() {
	Register(AdapterElastic, &elasticLogger{LogLevel: LevelTrace})
}
