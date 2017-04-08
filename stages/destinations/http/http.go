package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"github.com/streamsets/dataextractor/api"
	"github.com/streamsets/dataextractor/container/common"
	"io/ioutil"
	"log"
	"net/http"
)

const DEBUG = false

type HttpClientDestination struct {
	ctx                   context.Context
	resourceUrl           string
	headers               []interface{}
	singleRequestPerBatch bool
	httpCompression       string
}

func (h *HttpClientDestination) Init(ctx context.Context) {
	stageContext := (ctx.Value("stageContext")).(common.StageContext)
	stageConfig := stageContext.StageConfig
	log.Println("HttpClientDestination Init method")
	for _, config := range stageConfig.Configuration {
		if config.Name == "conf.resourceUrl" {
			h.resourceUrl = config.Value.(string)
		}

		if config.Name == "conf.headers" {
			h.headers = config.Value.([]interface{})
		}

		if config.Name == "conf.singleRequestPerBatch" {
			h.singleRequestPerBatch = config.Value.(bool)
		}

		if config.Name == "conf.client.httpCompression" {
			h.httpCompression = config.Value.(string)
		}
	}
}

func (h *HttpClientDestination) Write(batch api.Batch) error {
	log.Println("HttpClientDestination write method")
	var batchJSONValue []byte
	for _, record := range batch.GetRecords() {
		jsonValue, err := json.Marshal(record.Value)
		if err != nil {
			panic(err)
		}
		if h.singleRequestPerBatch {
			batchJSONValue = append(batchJSONValue, jsonValue...)
		} else {
			h.sendToSDC(jsonValue)
		}
	}
	if h.singleRequestPerBatch {
		h.sendToSDC(batchJSONValue)
	}
	return nil
}

func (h *HttpClientDestination) sendToSDC(jsonValue []byte) {
	var buf bytes.Buffer

	if h.httpCompression == "GZIP" {
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(jsonValue); err != nil {
			panic(err)
		}
		gz.Close()
	} else {
		buf = *bytes.NewBuffer(jsonValue)
	}

	req, err := http.NewRequest("POST", h.resourceUrl, &buf)
	if h.headers != nil {
		for _, header := range h.headers {
			req.Header.Set(header.(map[string]interface{})["key"].(string),
				header.(map[string]interface{})["value"].(string))
		}
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	if h.httpCompression == "GZIP" {
		req.Header.Set("Content-Encoding", "gzip")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if DEBUG {
		log.Println("response Status:", resp.Status)
		log.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		log.Println("response Body:", string(body))
	}
}

func (h *HttpClientDestination) Destroy() {

}