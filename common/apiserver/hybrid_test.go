package apiserver

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	test_pri_key = `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDDbHGzrD8n+Hp8YPtU555INT2ZAXLKWE/LdW0/QXJ9KiZoRh29
qNDRyCgOEGmIRD1CxQLw70LXggFSO8GBfBRsYFH5LynDy7mQROSnDpAjNNurU11+
3rigaF3IWuBx//qxvi9Kz7oQ5j/hr6twQH1NRI+rcmMjvLHN+YU+DoHj5QIDAQAB
AoGATchLFU2YsasX7YuYXbn26Ryv0MefzeQKlpu9iPDexezR7q0Bx2x6+RSmxLpJ
luA6VeoeepFw1GA9cGKyaXxej/Y3Rmf0iqdgDeImQWhg1pUsIS+EPvDY4bD5+rNo
0OW2ZThjPIpR2hgh/rWXuM2lRBDeeVhmOsi7tCa23Yw8s10CQQDXvgX7SRZcVgYe
gv33hko0eyqwzi5OkGbY2JLlQMF3TL8YL1H51WLCWprtFbnZ9BnZNBdXyVQSUwms
zKdtjFuHAkEA5+PNDZft+6oIFQJcROCojJ9yR84NMswxapVbK86eEYYkhA7wbeFC
yMVm4DMOvkDfvkgEWUINTtvoOZLZyREYMwJAL74ehsBi0WY8Dm6Ak1FFhJ2pEd1e
xAYSrHQo9dDBv4vdUhXOt1HwfAAe/s5rBX+OZNGxRL0J/NAhePsFJioEawJAQzpP
4HkDjcqlvTGJ/o4DT4GKaDbcrLV2Pig+3lxwhzQUshSCr9h2vC4+vREQXSgBtfC7
EgWMRiiLEuX4Lcq+8QJAX6zjVz1rmzn82t8JHUrKDBSRdN7YbhDrjMTQafIYlnJC
WP1z7EfGtyGjDzYgjIXO7HXqo5afYWaMT/4iYHJnyg==
-----END RSA PRIVATE KEY-----`
	test_pub_key = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDDbHGzrD8n+Hp8YPtU555INT2Z
AXLKWE/LdW0/QXJ9KiZoRh29qNDRyCgOEGmIRD1CxQLw70LXggFSO8GBfBRsYFH5
LynDy7mQROSnDpAjNNurU11+3rigaF3IWuBx//qxvi9Kz7oQ5j/hr6twQH1NRI+r
cmMjvLHN+YU+DoHj5QIDAQAB
-----END PUBLIC KEY-----`
)

// cc03e747a6afbbcbf8be7668acfebee5

func TestRsaSignWithSha256(t *testing.T) {
//	value := RsaSign("test123", test_pri_key)
//	str_value := string(value)
//	fmt.Println(str_value)

}

func TestRsaSignWithMD5(t *testing.T) {
	value := rsaSign(logrus.StandardLogger(), "changfeng", test_pri_key)
	assert.Equal(t, value, `dpafdQKSIKsZmpFS3V8Wm94N8YBCW14Zix2c4JH2tZ+mTnL1ZIW4kuH0xx68WQM1ETKww6zuDKzvLayjv6KWIcIHBMm5SJCL//MWWyt4ocEc22jdAdoRIL/WWT+4uI6r+Bi5bBE0liWVIBOzVqhxx0dAtBDPzHPgc67ekHsVvTQ=`)
}
