package cryptdata

import (
	"crypto/rsa"
	
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

type rsaKeyPair struct {
	Id               string
	CreatedTimestamp int64
	ExpiredTimestamp int64
	PrivateKey       *rsa.PrivateKey
	PublicKey        string

}

type KeyInfo struct {
	Id               string `json:"id"`
	CreatedTimestamp int64  `json:"createdTimestamp"`
	ExpiredTimestamp int64  `json:"expiredTimestamp"`
	PublicKey        string `json:"publicKey"`
}

type KeyInfos []KeyInfo

func (k KeyInfos) Len() int {
	return len(k)
}
func (k KeyInfos) Less(i, j int) bool {
	return k[i].ExpiredTimestamp < k[j].ExpiredTimestamp
}
func (k KeyInfos) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

type secretParam struct {
	SecretName string
	PlainText string
	CreatedTimestamp int64
	ExpiredTimestamp int64
}

type ParamInfo struct {
	SecretName string
	CreatedTimestamp int64
	ExpiredTimestamp int64
}

var (
	clearExpiredTimer_    *timermanager.Timer
	clearExpiredInterval_ = 60
)

func Init() {
	var err error
	timerManager := timermanager.GetTimerManager()
	clearExpire := func() {
		clearExpiredKey()
		clearExpiredParam()
	}
	if clearExpiredTimer_, err = timerManager.CreateTimerInSeconds(clearExpire, clearExpiredInterval_); err != nil {
		log.GetLogger().Error("InitPluginCheckTimer: pluginListReportTimer err: ", err.Error())
	} else {
		go func() {
			_, err = clearExpiredTimer_.Run()
			if err != nil {
				log.GetLogger().Error("Timer for clearing expired keypairs and params failed: ", err.Error())
			} else {
				log.GetLogger().Info("Timer for clearing expired keypairs and params started")
			}
		}()
	}
}