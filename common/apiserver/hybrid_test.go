package apiserver

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
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

func TestGetClientIPWithTTL(t *testing.T) {
	t.Run("first call success", func(t *testing.T) {
		defer gomonkey.ApplyFunc(osutil.ExternalIP, func() (net.IP, error) {
			return net.ParseIP("203.0.113.42"), nil
		}).Reset()

		ip, err := getClientIPWithTTL(log.GetLogger(), time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ip != "203.0.113.42" {
			t.Errorf("expected IP 203.0.113.42, got %s", ip)
		}
	})

	t.Run("cache hit within TTL", func(t *testing.T) {
		clientIPMu.Lock()
		clientIPCache = "203.0.113.42"
		clientIPExpiresAt = time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC) // expire at 12:00:30
		clientIPMu.Unlock()

		called := false
		defer gomonkey.ApplyFunc(osutil.ExternalIP, func() (net.IP, error) {
			called = true
			return net.ParseIP("198.51.100.1"), nil
		}).Reset()

		ip, err := getClientIPWithTTL(log.GetLogger(), time.Date(2024, 1, 1, 12, 0, 10, 0, time.UTC)) // 12:00:10 < 12:00:30
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if called {
			t.Error("ExternalIP was called, but cache should have been used")
		}
		if ip != "203.0.113.42" {
			t.Errorf("expected cached IP, got %s", ip)
		}
	})

	t.Run("cache miss after TTL", func(t *testing.T) {
		clientIPMu.Lock()
		clientIPCache = "192.0.2.1"
		clientIPExpiresAt = time.Date(2024, 1, 1, 12, 0, 30, 0, time.UTC) // expire at 12:00:30
		clientIPMu.Unlock()

		called := false
		newIp := "198.51.100.1"
		defer gomonkey.ApplyFunc(osutil.ExternalIP, func() (net.IP, error) {
			called = true
			return net.ParseIP(newIp), nil
		}).Reset()

		ip, err := getClientIPWithTTL(log.GetLogger(), time.Date(2024, 1, 1, 12, 6, 0, 0, time.UTC)) // 12:06:00 > 12:00:00, should expire and return new ip
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("ExternalIP was not called, but cache should have expired")
		}
		if ip != newIp {
			t.Errorf("expected new IP, got %s", ip)
		}
	})

	t.Run("ExternalIP returns error", func(t *testing.T) {
		clientIPCache = ""
		defer gomonkey.ApplyFunc(osutil.ExternalIP, func() (net.IP, error) {
			return nil, errors.New("network unreachable")
		}).Reset()

		_, err := getClientIPWithTTL(log.GetLogger(), time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC))
		assert.Equal(t, "network unreachable", err.Error())
	})

	t.Run("recover from previous error", func(t *testing.T) {
		patch := gomonkey.ApplyFunc(osutil.ExternalIP, func() (net.IP, error) {
			return nil, errors.New("temp error")
		})

		ip, err := getClientIPWithTTL(log.GetLogger(), time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC))
		assert.Equal(t, "temp error", err.Error())
		assert.Equal(t, "", ip)
		patch.Reset()

		newIp := "192.0.2.100"
		defer gomonkey.ApplyFunc(osutil.ExternalIP, func() (net.IP, error) {
			return net.ParseIP(newIp), nil
		}).Reset()

		ip, err = getClientIPWithTTL(log.GetLogger(), time.Date(2024, 1, 1, 13, 1, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error on retry: %v", err)
		}
		if ip != newIp {
			t.Errorf("expected recovered IP, got %s", ip)
		}
	})
}

func TestGetClientIPWithTTL_Concurrent(t *testing.T) {
	const goroutines = 100
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)

	defer gomonkey.ApplyFunc(osutil.ExternalIP, func() (net.IP, error) {
		time.Sleep(2 * time.Second) //mock slow ExternalIP
		return net.ParseIP("10.0.0.1"), nil
	}).Reset()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := getClientIPWithTTL(log.GetLogger(), time.Now())
			if err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)
	if len(errCh) > 0 {
		t.Errorf("concurrent calls failed: %v", <-errCh)
	}
}
