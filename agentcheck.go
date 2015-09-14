// api_monitor project main.go
package main

import (
	"api_monitor/metric"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/open-falcon/common/model"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

var STEP int64 = 60

type Endpoint struct {
	Endpoint string `json:"endpoint,omitempty"`
	Counter  string `json:"counter,omitempty"`
}

type JsonFloat float64

func (v JsonFloat) MarshalJSON() ([]byte, error) {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return []byte("null"), nil
	} else {
		return []byte(fmt.Sprintf("%f", f)), nil
	}
}

type RRDData struct {
	Timestamp int64     `json:"timestamp"`
	Value     JsonFloat `json:"value"`
}

func NewRRDData(ts int64, val float64) *RRDData {
	return &RRDData{Timestamp: ts, Value: JsonFloat(val)}
}

type GraphLastResp struct {
	Endpoint string   `json:"endpoint"`
	Counter  string   `json:"counter"`
	Value    *RRDData `json:"value"`
}

func GetList(url string) ([]*Endpoint, error) {
	if url == "" {
		url = "http://127.0.0.1:6031/all/endpoints"
	}
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	//	fmt.Println(string(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		defer resp.Body.Close()
	}
	var endpoints []*Endpoint
	err = json.Unmarshal(body, &endpoints)
	//	fmt.Println(len(endpoints))
	return endpoints, err
}

func CheckLast(url string, endpoints []*Endpoint) []*model.MetricValue {
	for i, _ := range endpoints {
		endpoints[i].Counter = "agent.alive"
	}
	post, err := json.Marshal(&endpoints)
	//	fmt.Println(string(post))
	if err != nil {
		return nil
	}
	if url == "" {
		url = "http://127.0.0.1:9966/graph/last"
	}
	resp, err := http.DefaultClient.Post(url, "application/json", strings.NewReader(string(post)))

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil
	}
	if body != nil {
		defer resp.Body.Close()
	}

	var last []*GraphLastResp
	err = json.Unmarshal(body, &last)

	//	fmt.Println(string(body))

	if err != nil {
		return nil
	}
	now := time.Now().Unix()
	var metrics []*model.MetricValue
	for i, _ := range last {
		//		fmt.Println(now - last[i].Value.Timestamp)
		if (now - last[i].Value.Timestamp) > (STEP * 2) {
			fmt.Printf("%v about %v s\n", last[i].Endpoint, (now - last[i].Value.Timestamp))
			m := metric.GaugeValue("agent.alive", 0)
			m.Endpoint = last[i].Endpoint
			m.Step = STEP
			if now-last[i].Value.Timestamp > 3*STEP {
				m.Timestamp = now - 2*STEP
			} else {
				m.Timestamp = STEP + last[i].Value.Timestamp
			}
			metrics = append(metrics, m)
		}
	}
	return metrics

}

func PushMetric(url string, metrics []*model.MetricValue) error {
	post, err := json.Marshal(&metrics)
	if err != nil {
		return nil
	}
	if url == "" {
		url = "http://127.0.0.1:1988/v1/push"
	}
	resp, err := http.DefaultClient.Post(url, "application/json", strings.NewReader(string(post)))

	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil
	}
	if body != nil {
		defer resp.Body.Close()
	}

	//	fmt.Println(string(body))

	if string(body) == "success" {
		return nil
	} else {
		return fmt.Errorf(string(body))
	}

}

//main

type Config struct {
	GetListURL    string `json:"geturl,omitempty"`
	GetLastURL    string `json:"lasturl,omitempty"`
	PushMetricURL string `json:"pushurl,omitempty"`
}

var CONFIGFILE string

func main() {
	flag.StringVar(&CONFIGFILE, "f", "./cfg.json", "Path of config file")
	flag.Int64Var(&STEP, "s", 60, "internal")
	flag.Parse()

	config, err := ioutil.ReadFile(CONFIGFILE)
	if err != nil {
		log.Fatal("read config file error")
		return
	}
	var conf Config
	err = json.Unmarshal(config, &conf)
	if err != nil {
		log.Fatal("render json error")
		return
	}

	endpoints, err := GetList(conf.GetListURL)
	if err != nil || len(endpoints) == 0 {
		log.Fatal("can't get endpoint list")
	}
	//	for _, endpoint := range endpoints {
	//		fmt.Println(endpoint.Endpoint)
	//	}
	m := CheckLast(conf.GetLastURL, endpoints)
	//	fmt.Println(len(m))
	//	for _, val := range m {
	//		fmt.Printf("%v - %v - %v - %v - %v\n", val.Endpoint, val.Metric, val.Timestamp, val.Type, val.Value)
	//	}
	if len(m) != 0 {
		err = PushMetric(conf.PushMetricURL, m)
	} else {
		err = nil
	}

	if err == nil {
		//		log.Printf("Check agent alive ok, upload with %v metric.\n", len(m))
	} else {
		log.Fatal("Check agent alive error, with %v", err.Error())
	}
	resArray := []*model.MetricValue{}
	res := model.MetricValue{}
	res.Endpoint, _ = os.Hostname()
	res.Timestamp = time.Now().Unix()
	res.Metric = "agent.not_alive.num"
	res.Step = 60
	res.Type = "GAUGE"
	res.Value = len(m)
	resArray = append(resArray, &res)
	output, err := json.Marshal(&resArray)
	if err != nil {
		log.Fatal("parser metric json error")
	} else {
		fmt.Printf(string(output))
	}
}
