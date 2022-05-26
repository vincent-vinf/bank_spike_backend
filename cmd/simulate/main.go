package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	UserPrefix  = "s-user-t-"
	PhonePrefix = "+86-t-"
	IdNumber    = "s-no-t-"
	UserNum     = 10 // 用户数量
	UserPerNum  = 1  // 每个用户请求最大数量
	SpikeId     = "2"
	BaseUrl     = "http://127.0.0.1:"
)

var (
	UrlMap = map[string]string{
		"user":  BaseUrl + "8080/users/",
		"spike": BaseUrl + "8081/spike/",
	}
	WorkStatus = []string{"公务员", "无业", "老师"}
)

type tokenInfo struct {
	username string
	token    string
}

type spikeUserResult struct {
	username string
	res      map[string]interface{}
}

type spikeSimulateResult struct {
	success struct {
		cnt  int
		list []spikeUserResult
	}
	fail struct {
		cnt  int
		list []spikeUserResult
	}
}

var wg sync.WaitGroup
var tokenInfos []*tokenInfo
var sRes spikeSimulateResult

func main() {
	cnt := 0

	// 模拟注册
	rand.Seed(time.Now().UnixNano())
	for i := 8; i < UserNum+8; i++ {
		go func(i int) {
			wg.Add(1)
			res, err := SimulatePost(
				UrlMap["user"]+"register",
				map[string]interface{}{
					"username":   UserPrefix + strconv.Itoa(i),
					"phone":      PhonePrefix + strconv.Itoa(i),
					"idNumber":   IdNumber + strconv.Itoa(i),
					"workStatus": WorkStatus[rand.Intn(len(WorkStatus))],
					"passwd":     "123456",
					"age":        rand.Intn(33) + 16,
				},
				map[string]string{
					"Content-Type": "application/json",
				},
			)
			if err != nil || res["error"] != nil {
				log.Fatal("simulate register failed")
			}

			cnt++
			tokenInfos = append(tokenInfos, &tokenInfo{
				username: UserPrefix + strconv.Itoa(i),
				token:    res["token"].(string),
			})
			wg.Done()
		}(i)
	}
	wg.Wait()
	log.Println(cnt, " registered successfully")

	// 测试：模拟登录
	//res, err := SimulatePost(
	//	UrlMap["user"]+"login",
	//	map[string]interface{}{
	//		"phone":  PhonePrefix + strconv.Itoa(0),
	//		"passwd": "123456",
	//	},
	//	map[string]string{
	//		"Content-Type": "application/json; charset=utf-8",
	//	},
	//)
	//if err != nil || res["error"] != nil {
	//	log.Fatal("simulate register failed")
	//}
	//
	//cnt++
	//tokenInfos = append(tokenInfos, &tokenInfo{
	//	username: UserPrefix + strconv.Itoa(0),
	//	token:    res["token"].(string),
	//})

	// 获取 spike 开始时间、定时任务
	spike, err := SimulateGet(UrlMap["user"]+"spike/"+SpikeId, map[string]string{"Authorization": "Bearer " + tokenInfos[0].token})
	if err != nil {
		log.Fatal("get spike start time failed")
	}
	startTime, err := time.Parse("2006-01-02T15:04:05Z07:00", spike["StartTime"].(string))
	if err != nil {
		fmt.Println(err)
	}

	c := cron.New(cron.WithSeconds())
	spec := fmt.Sprintf("%d %d %d %d %d ?", startTime.Second(), startTime.Minute(), startTime.Hour(), startTime.Day(), startTime.Month())
	wg.Add(1) // 等待定时任务完成
	_, err = c.AddFunc(spec, SimulateSpike)
	if err != nil {
		log.Fatal("cron start failed")
	}
	c.Start()

	// 结果打印（用户 id，秒杀结果，时间）、超卖检验
	wg.Wait() // 等待所有用户秒杀结束
	fmt.Println(sRes)
}

// SimulateSpike 模拟秒杀
func SimulateSpike() {
	for _, info := range tokenInfos {
		wg.Add(1)
		go func(info *tokenInfo) {
			// 模拟获取随机秒杀链接
			res, err := SimulateGet(UrlMap["spike"]+SpikeId, map[string]string{"Authorization": "Bearer " + info.token})
			if err != nil {
				log.Fatal("simulate spike failed")
			}
			// 模拟秒杀
			res, err = SimulatePost(
				UrlMap["spike"]+SpikeId+"/"+res["token"].(string),
				nil,
				map[string]string{"Authorization": "Bearer " + info.token},
			)
			// 保存结果
			if res["status"] != nil && res["status"].(string) == "success" {
				sRes.success.cnt++
				sRes.success.list = append(sRes.success.list, spikeUserResult{
					username: info.username,
					res:      res,
				})
			} else {
				sRes.fail.cnt++
				sRes.fail.list = append(sRes.fail.list, spikeUserResult{
					username: info.username,
					res:      res,
				})
			}
			log.Println(info.username, "finish spike")
			wg.Done()
		}(info)
	}
	wg.Done()
}

// SimulateGet get 封装
func SimulateGet(url string, headers map[string]string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	resMap := make(map[string]interface{})
	_ = json.Unmarshal(body, &resMap)
	return resMap, nil
}

// SimulatePost post 封装
func SimulatePost(url string, data map[string]interface{}, headers map[string]string) (map[string]interface{}, error) {
	bytesData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	postBody := bytes.NewReader(bytesData)
	req, err := http.NewRequest("POST", url, postBody)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	fmt.Printf("err (%v, %T)\n", err, err)
	if res == nil {
		fmt.Println(data)
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	resMap := make(map[string]interface{})
	_ = json.Unmarshal(body, &resMap)
	return resMap, nil
}
