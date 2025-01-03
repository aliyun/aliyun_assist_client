package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"config_ecs_instance_connect/agent/util"
	"config_ecs_instance_connect/agent/log"
)

type ResponseData struct {
	InstanceId string    `json:"instanceId"`
	RequestId  string    `json:"requestId"`
	KeyInfos   []KeyInfo `json:"keyInfos"`
}

type KeyInfo struct {
	UserName    string `json:"userName"`
	PublicKey   string `json:"publicKey"`
	ExpiredTime int64  `json:"expiredTime"`
}

func main() {
	
	log.Info("config_ecs_instance_connect core start")

	sleep_internals_seconds := 3
	for {
		host := util.GetServerHost()
		if host != "" {
			log.Info(fmt.Sprint("GET_HOST_OK ", host))
			break
		} else {
			log.Info("GET_HOST_ERROR")
		}
		time.Sleep(time.Duration(sleep_internals_seconds) * time.Second)
		sleep_internals_seconds = sleep_internals_seconds * 2
		if sleep_internals_seconds > 180 {
			log.Info("GetServerHost failed,exit")
			return
		}
	}

	userName := ""
	if len(os.Args) >= 2 {
		userName = os.Args[1]
	}
	log.Info(fmt.Sprint("user name:", userName))

	url := "https://" + util.GetServerHost()
	url += "/luban/api/v1/task/list_ssh_public_key"
	if userName != "" {
		url += "?username=" + userName
	}
	response, err := util.HttpGet(url)
	if err != nil {
		log.Info(fmt.Sprint("list_ssh_public_key failed,exit", err))
		return
	}

	var responseData ResponseData
	err = json.Unmarshal([]byte(response), &responseData)

	if err == nil {
		for _, v := range responseData.KeyInfos {
			if v.UserName=="" {
				v.UserName = "root"
			}
			time_stamp := time.Now().UnixNano() / 1e6
			if int64(time_stamp) > v.ExpiredTime {
				log.Info(fmt.Sprint("skip due to ExpiredTime", time_stamp, v.ExpiredTime))
				continue
			}
			if userName != v.UserName {
				log.Info(fmt.Sprint("skip due to UserName:", userName, v.UserName))
				continue
			}
			log.Info(fmt.Sprintf("recv a key: %s******", v.PublicKey[:20]))
			fmt.Println(v.PublicKey)
		}
	} else {
		log.Info(fmt.Sprint("json Unmarshal failed,exit", err))
		return
	}

}
