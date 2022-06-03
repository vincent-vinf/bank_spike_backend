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
	UserNum     = 100 // 用户数量
	UserPerNum  = 1   // 每个用户请求最大数量
	SpikeId     = "4"
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
var mux sync.Mutex
var tokenInfos []*tokenInfo
var sRes spikeSimulateResult

func main() {
	// 模拟注册
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < UserNum; i++ {
		wg.Add(1)
		go func(i int) {
			// 模拟注册
			//SimulateRegister(i)
			// 模拟登录
			SimulateLogin(i)
			wg.Done()
		}(i)
		if (i+1)%10 == 0 {
			wg.Wait()
		}
	}
	log.Println(len(tokenInfos), "register/login successfully")

	// 获取 spike 开始时间、定时任务
	spike, err := SimulateGet(UrlMap["user"]+"spike/"+SpikeId, map[string]string{"Authorization": "Bearer " + tokenInfos[0].token})
	if err != nil {
		log.Println("err", err)
		log.Fatal("get spike start time failed")
	}
	startTime, err := time.Parse("2006-01-02T15:04:05Z07:00", spike["StartTime"].(string))
	if err != nil {
		fmt.Println(err)
	}

	c := cron.New(cron.WithSeconds())
	var spec string
	if startTime.Before(time.Now()) {
		// 如果活动已开始 5 秒后自动执行
		runTime := time.Now().Add(time.Second * 1)
		spec = fmt.Sprintf("%d %d %d %d %d ?", runTime.Second(), runTime.Minute(), runTime.Hour(), runTime.Day(), runTime.Month())
	} else {
		spec = fmt.Sprintf("%d %d %d %d %d ?", startTime.Second(), startTime.Minute(), startTime.Hour(), startTime.Day(), startTime.Month())
	}
	wg.Add(1) // 等待定时任务完成
	_, err = c.AddFunc(spec, SimulateSpike)
	if err != nil {
		log.Println("err", err)
		log.Fatal("cron start failed")
	}
	c.Start()
	log.Println("cron start successfully")

	// 结果打印（用户 id，秒杀结果，时间）、超卖检验
	wg.Wait() // 等待所有用户秒杀结束
	log.Println("spike end")
	log.Println("success:", sRes.success.cnt)
	log.Println("fail:", sRes.fail.cnt)
}

func SimulateRegister(i int) {
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
		log.Println("err", err)
		log.Println("res", res)
		log.Fatal("simulate register failed")
	}

	mux.Lock()
	tokenInfos = append(tokenInfos, &tokenInfo{
		username: UserPrefix + strconv.Itoa(i),
		token:    res["token"].(string),
	})
	mux.Unlock()
}

func SimulateLogin(i int) {
	res, err := SimulatePost(
		UrlMap["user"]+"login",
		map[string]interface{}{
			"phone":  PhonePrefix + strconv.Itoa(i),
			"passwd": "123456",
		},
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if err != nil || res["error"] != nil {
		log.Println("err", err)
		log.Println("res", res)
		log.Fatal("simulate login failed")
	}

	mux.Lock()
	tokenInfos = append(tokenInfos, &tokenInfo{
		username: UserPrefix + strconv.Itoa(i),
		token:    res["token"].(string),
	})
	mux.Unlock()
}

// SimulateSpike 模拟秒杀
func SimulateSpike() {
	log.Println("spike begin")
	for _, info := range tokenInfos {
		// 模拟用户同一时间点击 UserPerNum 次
		cnt := rand.Intn(UserPerNum) + 1
		for i := 0; i < cnt; i++ {
			wg.Add(1)
			go func(info *tokenInfo) {
				// 模拟获取随机秒杀链接
				res, err := SimulateGet(UrlMap["spike"]+SpikeId, map[string]string{"Authorization": "Bearer " + info.token})
				if err != nil {
					log.Println("err", err)
					log.Fatal("simulate spike failed")
				}
				// 是否正常获取到 token
				if _, ok := res["error"]; !ok {
					// 模拟秒杀
					res, err = SimulatePost(
						UrlMap["spike"]+SpikeId+"/"+res["token"].(string),
						nil,
						map[string]string{"Authorization": "Bearer " + info.token},
					)
				}

				mux.Lock()
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
				mux.Unlock()
				wg.Done()
			}(info)
		}
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
	if res == nil {
		fmt.Println(data)
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	resMap := make(map[string]interface{})
	_ = json.Unmarshal(body, &resMap)
	return resMap, nil
}
