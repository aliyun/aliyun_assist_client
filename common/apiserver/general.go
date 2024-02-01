package apiserver

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"crypto/x509"
	"crypto/tls"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/kirinlabs/HttpRequest"
	"go.uber.org/atomic"

	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	IntranetDomain = ".axt.aliyun.com"
)

var (
	errUnknownDetectionResponse = errors.New("Unknown connection detection response")

	wellKnownRegionIds = []string{
		"cn-hangzhou",
		"cn-qingdao",
		"cn-beijing",
		"cn-zhangjiakou",
		"cn-huhehaote",
		"cn-shanghai",
		"cn-shenzhen",
		"cn-hongkong",
		"eu-west-1",
	}
)

type GeneralProvider struct {
	regionId atomic.String

	instanceIdHeaders map[string]string
	initInstanceIdHeadersOnce sync.Once
}

func (*GeneralProvider) Name() string {
	return "GeneralProvider"
}

func (*GeneralProvider) CACertificate(logger logrus.FieldLogger, refresh bool) ([]byte, error) {
	currentVersionDir, err := pathutil.GetCurrentPath()
	if err != nil {
		return nil, err
	}

	certPath := filepath.Join(currentVersionDir, "config", "GlobalSignRootCA.crt")
	certFile, err := os.Open(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open bundled CA certificate file: %w", err)
	}

	pemCerts, err := io.ReadAll(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundled CA certificate file %s: %w", certPath, err)
	}

	return pemCerts, nil
}

func (p *GeneralProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
	// 0. Retrieve domain from env
	if domain := os.Getenv("ALIYUN_ASSIST_SERVER_HOST"); domain != "" {
		logger.Info("Get host from env ALIYUN_ASSIST_SERVER_HOST: ", domain)
		return domain, nil
	}

	// 1. Read region id cached in file if exists
	regionId := getRegionIdInFile()
	if regionId != "" {
		domain := regionId + IntranetDomain
		if err := connectionDetect(logger, domain); err == nil {
			p.regionId.Store(regionId)
			networkcategory.Set(networkcategory.NetworkVPC)
			return domain, nil
		} else {
			logger.WithFields(logrus.Fields{
				"domain": domain,
			}).WithError(err).Error("Failed on detection of API server connection")
		}
	}

	// 2. Retrieve region id from meta server in VPC network
	regionId, _ = httpGetWithoutExtraHeader(logger, "http://100.100.100.200/latest/meta-data/region-id")
	if regionId != "" {
		domain := regionId + IntranetDomain
		if err := connectionDetect(logger, domain); err == nil {
			p.regionId.Store(regionId)
			go saveRegionIdToFile(logger, regionId)
			networkcategory.Set(networkcategory.NetworkVPC)
			return domain, nil
		} else {
			logger.WithFields(logrus.Fields{
				"domain": domain,
			}).WithError(err).Error("Failed on detection of API server connection")
		}
	}

	// 3. Poll well-known API servers for region id
	for _, regionId := range wellKnownRegionIds {
		regionId, err := httpGetWithoutExtraHeader(logger, "https://" + regionId + IntranetDomain + "/luban/api/classic/region-id")
		if err != nil {
			continue
		}

		domain := regionId + IntranetDomain
		if err := connectionDetect(logger, domain); err == nil {
			p.regionId.Store(regionId)
			go saveRegionIdToFile(logger, regionId)
			networkcategory.Set(networkcategory.NetworkClassic)
			return domain, nil
		} else {
			logger.WithFields(logrus.Fields{
				"domain": domain,
			}).WithError(err).Error("Failed on detection of API server connection")
		}
	}

	return "", requester.ErrNotProvided
}

func (p *GeneralProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	if p.regionId.Load() == "" {
		return nil, requester.ErrNotProvided
	}

	p.initInstanceIdHeadersOnce.Do(func() {
		instanceId := func () string {
			instanceId, err := httpGetWithoutExtraHeader(logger, "http://100.100.100.200/latest/meta-data/instance-id")
			if err != nil {
				return "unknown"
			}

			return instanceId
		}()
		p.instanceIdHeaders = map[string]string{
			"X-Client-Instance-ID": instanceId,
		}
	})

	return p.instanceIdHeaders, nil
}

func (p *GeneralProvider) RegionId(logger logrus.FieldLogger) (string, error) {
	regionId := p.regionId.Load()
	if regionId != "" {
		return regionId, nil
	}

	logger.Errorln("No cached region ID and server domain detection procedure is initiated for it")
	_, err := p.ServerDomain(logger)
	if err != nil {
		return "", requester.ErrNotProvided
	}
	regionId = p.regionId.Load()
	if regionId != "" {
		return regionId, nil
	} else {
		return "", requester.ErrNotProvided
	}
}

func getRegionIdInFile() string {
	currentVersionDir, _ := pathutil.GetCurrentPath()
	path := filepath.Join(filepath.Dir(currentVersionDir), "region-id")

	if regionIdFile, err := os.Open(path); err == nil {
		if raw, err2 := io.ReadAll(regionIdFile); err2 == nil {
			return strings.TrimSpace(strings.Trim(string(raw), "\r\t\n"))
		}
	}
	return ""
}

func saveRegionIdToFile(logger logrus.FieldLogger, regionId string) {
	currentVersionDir, _ := pathutil.GetCurrentPath()
	path := filepath.Join(filepath.Dir(currentVersionDir), "region-id")

	err := os.WriteFile(path, []byte(regionId), os.FileMode(0o644))
	if err != nil {
		logger.WithError(err).Warning("Failed to save detected region ID into cache file")
	} else {
		logger.Info("Saved detected region ID into cache file")
	}
}

func connectionDetect(logger logrus.FieldLogger, domain string) error {
	url := "https://" + domain + "/luban/api/connection_detect"
	content, err := httpGetWithoutExtraHeader(logger, url)
	if err != nil {
		return err
	}

	if content == "ok" {
		return nil
	}

	return errUnknownDetectionResponse
}

func httpGetWithoutExtraHeader(logger logrus.FieldLogger, url string) (string, error) {
	request := HttpRequest.Transport(requester.GetHTTPTransport(logger))

	// IMPORTANT NOTE: Although time.Duration type is used for the argument of
	// (*HttpRequest.Request).SetTimeout(d time.Duration), the actual unit is
	// not nanosecond but second, since the value would be internally multiplied
	// by base.
	//
	// See link below for implementation detail:
	// https://github.com/kirinlabs/HttpRequest/blob/432628e833bda77cc426fc1bee9825a13f6b4df1/request.go#L105
	request.SetTimeout(5)

	request.SetHeaders(map[string]string{
		requester.UserAgentHeader: requester.UserAgentValue,
	})

	logger = logger.WithField("url", url)
	response, err := request.Get(url)
	if err != nil {
		if errors.Is(err, x509.UnknownAuthorityError{}) {
			logger.Info("certificate error, reload certificates and retry")
			request.Transport(requester.PeekHTTPTransport(logger))
			certPool := requester.PeekRefreshedRootCAs(logger)
			request.SetTLSClient(&tls.Config{
				RootCAs: certPool,
			})
			if response, err = request.Get(url); err == nil {
				logger.Info("certificated updated")
				requester.RefreshHTTPCas(logger, certPool)
			} else {
				logger.WithError(err).Error("Failed to send HTTP GET request")
				return "", err
			}
		} else {
			logger.WithError(err).Error("Failed to send HTTP GET request")
			return "", err
		}
	}
	defer response.Close()

	content, _ := response.Content()
	if err == nil && response.StatusCode() > 400 {
		err = requester.NewHttpErrorCode(response.StatusCode())
	}

	logger.WithFields(logrus.Fields{
		"url": url,
		"responseCode": response.StatusCode(),
		"responseContent": content,
	}).WithError(err).Infoln("HTTP GET Requested")
	return content, err
}
