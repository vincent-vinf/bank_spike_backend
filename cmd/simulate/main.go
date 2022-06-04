package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/schollz/progressbar/v3"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	UserPrefix  = "s-user-"
	PhonePrefix = "+86-"
	IdNumber    = "s-no-"
	UserNum     = 80 // 用户数量
	UserPerNum  = 5  // 每个用户请求最大数量
	SpikeId     = "6"
	BaseUrl     = "http://spike.vinf.top"
)

var (
	UrlMap = map[string]string{
		"user":  BaseUrl + "/users/",
		"spike": BaseUrl + "/spike/",
	}
	WorkStatus = []string{"公务员", "无业", "老师"}
)

type tokenInfo struct {
	username   string
	token      string
	spikeToken string
}

type spikeUserResult struct {
	status   int
	username string
	res      map[string]interface{}
}

var wg sync.WaitGroup
var muxSpikeToken sync.Mutex
var startTime time.Time
var reqTimes [UserNum * UserPerNum]int64 // 数组直接下标赋值无需锁
var tokenInfos [UserNum]*tokenInfo
var sRes [UserNum * UserPerNum]spikeUserResult
var bar *progressbar.ProgressBar

func main() {
	bar = progressbar.Default(UserNum, "login")
	startTime = time.Now()
	log.Println("simulate begin")
	// 模拟注册
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < UserNum; i++ {
		wg.Add(1)
		go func(i int) {
			// 模拟注册
			//SimulateRegister(i)
			// 模拟登录
			SimulateLogin(i)
			SimulateGet(UrlMap["user"]+"spike/access/"+SpikeId, map[string]string{"Authorization": "Bearer " + tokenInfos[i].token})
			bar.Add(1)
			wg.Done()
		}(i)
		if (i+1)%UserNum == 0 {
			wg.Wait()
		}
	}
	log.Println("\n", len(tokenInfos), "register/login successfully, ran:", time.Now().Sub(startTime))

	// 获取 spike 开始时间、定时任务
	spike, err := SimulateGet(UrlMap["user"]+"spike/"+SpikeId, map[string]string{"Authorization": "Bearer " + tokenInfos[0].token})
	if err != nil {
		log.Println("err", err)
		log.Fatal("get spike start time failed")
	}
	spikeStartTime, err := time.Parse("2006-01-02T15:04:05Z07:00", spike["StartTime"].(string))
	if err != nil {
		log.Println(err)
	}

	c := cron.New(cron.WithSeconds())
	var spec string
	if spikeStartTime.Before(time.Now()) {
		// 如果活动已开始 5 秒后自动执行
		runTime := time.Now().Add(time.Second * 1)
		spec = fmt.Sprintf("%d %d %d %d %d ?", runTime.Second(), runTime.Minute(), runTime.Hour(), runTime.Day(), runTime.Month())
	} else {
		spec = fmt.Sprintf("%d %d %d %d %d ?", spikeStartTime.Second(), spikeStartTime.Minute(), spikeStartTime.Hour(), spikeStartTime.Day(), spikeStartTime.Month())
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
	spikeEnd := time.Now().Sub(startTime)
	log.Println("\nspike end, ran:", spikeEnd.Milliseconds(), "ms")
	var max, min, aver int64
	min = math.MaxInt64
	for _, reqTime := range reqTimes {
		if reqTime < min {
			min = reqTime
		} else if reqTime > max {
			max = reqTime
		}
		aver += reqTime
	}
	aver = aver / int64(len(reqTimes))
	log.Println("req min:", min, "ms")
	log.Println("req max:", max, "ms")
	log.Println("req average:", aver, "ms")
	log.Println("total:", len(sRes))
	log.Println("qps:", float64(UserNum*UserPerNum)/float64(spikeEnd.Milliseconds())*1000)
	var cnt1, cnt2, cnt3 int
	for _, r := range sRes {
		switch r.status {
		case 1:
			cnt1++
		case 2:
			cnt2++
		case 3:
			cnt3++
		}
	}
	log.Println("success to spike:", cnt1)
	log.Println("fail to spike:", cnt2)
	log.Println("fail to request:", cnt3)
	log.Println("simulate end")
	//log.Println("sRes", sRes)
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

	tokenInfos[i] = &tokenInfo{
		username: UserPrefix + strconv.Itoa(i),
		token:    res["token"].(string),
	}
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

	tokenInfos[i] = &tokenInfo{
		username: UserPrefix + strconv.Itoa(i),
		token:    res["token"].(string),
	}
}

// SimulateSpike 模拟秒杀
func SimulateSpike() {
	startTime = time.Now()
	log.Println("spike begin")
	bar.ChangeMax(UserNum * UserPerNum)
	bar.Describe("spike")
	bar.Reset()
	for i, info := range tokenInfos {
		// 模拟用户同一时间点击 UserPerNum 次
		//cnt := rand.Intn(UserPerNum) + 1
		wg.Add(1)
		go func(info *tokenInfo, i int) { // goroutine 传参
			for j := 0; j < UserPerNum; j++ {
				//var res map[string]interface{}
				//var err error
				// 模拟获取随机秒杀链接

				//t := time.Now()
				// 保证一个用户只需 get 一次 token
				//muxSpikeToken.Lock()
				//if tokenInfos[i].spikeToken == "" {
				t1 := time.Now()
				res, err := SimulateGet(UrlMap["spike"]+SpikeId, map[string]string{"Authorization": "Bearer " + info.token})
				if err == nil && res != nil {
					tokenInfos[i].spikeToken = res["token"].(string)
				}
				//reqTimes[i*UserPerNum+j] += time.Now().Sub(t1).Milliseconds()
				//}
				//muxSpikeToken.Unlock()
				//log.Println(time.Now().Sub(t).Milliseconds())

				// get token 错误时无需 spike 请求
				if err == nil {
					if _, ok := res["error"]; !ok {
						// 模拟秒杀
						//t1 := time.Now()
						res, err = SimulatePost(
							UrlMap["spike"]+SpikeId+"/"+tokenInfos[i].spikeToken,
							nil,
							map[string]string{"Authorization": "Bearer " + info.token},
						)
					}
				}
				reqTimes[i*UserPerNum+j] = time.Now().Sub(t1).Milliseconds() // 记录 spike 请求时间，用于统计
				// 保存结果
				if err != nil {
					sRes[i*UserPerNum+j] = spikeUserResult{
						status:   3,
						username: info.username,
						res:      map[string]interface{}{"error": err.Error()},
					}
				} else {
					if res["status"] != nil && res["status"].(string) == "success" {
						sRes[i*UserPerNum+j] = spikeUserResult{
							status:   1,
							username: info.username,
							res:      res,
						}
					} else {
						sRes[i*UserPerNum+j] = spikeUserResult{
							status:   2,
							username: info.username,
							res:      res,
						}
					}
				}
				bar.Add(1)
			}

			wg.Done()
		}(info, i)
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
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.New("get response error")
	}
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
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.New("post response error")
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	resMap := make(map[string]interface{})
	_ = json.Unmarshal(body, &resMap)
	return resMap, nil
}
