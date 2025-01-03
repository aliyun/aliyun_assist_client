package apiserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"go.uber.org/atomic"

	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/httputil"
	"github.com/aliyun/aliyun_assist_client/common/metaserver"
	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	IntranetDomain = ".axt.aliyun.com"
)

var (
	errUnknownDetectionResponse = errors.New("Unknown connection detection response")
	wellKnownRegionIds          = []string{
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
	serverDomain             atomic.String
	extraHTTPHeadersProvider atomic.Value
}

type GeneralHTTPHeadersProvider struct {
	instanceIdHeaders         map[string]string
	initInstanceIdHeadersOnce sync.Once
}

var (
	generalHTTPHeadersProvider = GeneralHTTPHeadersProvider{}
)

func (*GeneralProvider) Name() string {
	return "GeneralProvider"
}

func (p *GeneralProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
	// 1. Read region id cached in file if exists
	regionId, _ := regionidFileProvider.RegionId(logger)
	if regionId != "" {
		domain := regionId + IntranetDomain
		if err := ConnectionDetect(logger, domain); err == nil {
			p.serverDomain.Store(domain)
			go p.cacheRegionId(logger, regionId)
			networkcategory.Set(networkcategory.NetworkVPC)
			return domain, nil
		} else {
			logger.WithFields(logrus.Fields{
				"domain": domain,
			}).WithError(err).Error("Failed on detection of API server connection")
		}
	}

	// 2. Retrieve region id from meta server in VPC network
	regionId, _ = metaserverProvider.RegionId(logger)
	if regionId != "" {
		domain := regionId + IntranetDomain
		if err := ConnectionDetect(logger, domain); err == nil {
			p.serverDomain.Store(domain)
			go p.cacheRegionId(logger, regionId)
			networkcategory.Set(networkcategory.NetworkVPC)
			return domain, nil
		} else {
			logger.WithFields(logrus.Fields{
				"domain": domain,
			}).WithError(err).Error("Failed on detection of API server connection")
		}
	}

	// 3. Poll well-known API servers for region id
	if flagging.GetApiserverTryPreset() {
		for _, regionId := range wellKnownRegionIds {
			regionId, err := HttpGetWithoutExtraHeader(logger, "https://"+regionId+IntranetDomain+"/luban/api/classic/region-id")
			if err != nil {
				continue
			}

			domain := regionId + IntranetDomain
			if err := ConnectionDetect(logger, domain); err == nil {
				p.serverDomain.Store(domain)
				go p.cacheRegionId(logger, regionId)
				networkcategory.Set(networkcategory.NetworkClassic)
				return domain, nil
			} else {
				logger.WithFields(logrus.Fields{
					"domain": domain,
				}).WithError(err).Error("Failed on detection of API server connection")
			}
		}
	}

	return "", requester.ErrNotProvided
}

func (p *GeneralProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	if p.serverDomain.Load() == "" {
		return nil, requester.ErrNotProvided
	}

	epp := p.extraHTTPHeadersProvider.Load()
	if epp == nil {
		return generalHTTPHeadersProvider.ExtraHTTPHeaders(logger)
	}
	ep, ok := epp.(requester.ExtraHTTPHeadersProvider)
	if !ok {
		return generalHTTPHeadersProvider.ExtraHTTPHeaders(logger)
	}
	return ep.ExtraHTTPHeaders(logger)
}

func (*GeneralProvider) cacheRegionId(logger logrus.FieldLogger, regionId string) {
	requester.SetRegionId(regionId)
	regionidFileProvider.SaveRegionId(logger, regionId)
}

func (gp *GeneralHTTPHeadersProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	gp.initInstanceIdHeadersOnce.Do(func() {
		instanceId := func() string {
			instanceId, err := metaserver.GetInstanceId(logger)
			if err != nil {
				return "unknown"
			}
			return instanceId
		}()
		gp.instanceIdHeaders = map[string]string{
			"X-Client-Instance-ID": instanceId,
		}
	})

	return gp.instanceIdHeaders, nil
}

func ConnectionDetect(logger logrus.FieldLogger, domain string) error {
	url := "https://" + domain + "/luban/api/connection_detect"
	content, err := HttpGetWithoutExtraHeader(logger, url)
	if err != nil {
		return err
	}

	if content == "ok" {
		return nil
	}

	return errUnknownDetectionResponse
}

func HttpGetWithoutExtraHeader(logger logrus.FieldLogger, url string) (string, error) {
	return HttpGetWithSpecifiedHeader(logger, url, nil)
}

func HttpGetWithSpecifiedHeader(logger logrus.FieldLogger, url string, headers map[string]string) (string, error) {
	logger = logger.WithField("url", url)
	transport := requester.GetHTTPTransport(logger)
	request := httputil.NewGetReq(logger, transport, 5, headers)

	response, err := request.Get(url)
	if err != nil {
		var certificateErr *tls.CertificateVerificationError
		if !errors.As(err, &certificateErr) {
			logger.WithError(err).Error("Failed to send HTTP GET request")
			return "", err
		}

		// tls.CertificateVerificationError encountered. Gonna re-accumulate root CA
		// certificate pool and retry requesting.
		logger.Info("certificate error, reload certificates and retry")
		transport = transport.Clone()
		requester.AccumulateRootCAs(logger)(func(certPool *x509.CertPool) bool {
			request = httputil.NewGetReq(logger, transport, 5, headers)
			request.SetTLSClient(&tls.Config{
				RootCAs: certPool,
			})
			if response, err = request.Get(url); err == nil {
				logger.Info("certificate updated")
				requester.RefreshHTTPCas(logger, certPool)
				return false
			}

			return true
		})
		if err != nil {
			logger.WithError(err).Error("Failed to send HTTP GET request")
			return "", err
		}
	}
	defer response.Close()

	content, _ := response.Content()
	if err == nil && response.StatusCode() > 400 {
		err = httpbase.NewStatusCodeError(response.StatusCode())
	}

	logger.WithFields(logrus.Fields{
		"url":             url,
		"responseCode":    response.StatusCode(),
		"responseContent": content,
	}).WithError(err).Infoln("HTTP GET Requested")
	return content, err
}

func HttpPostWithSpecifiedHeader(logger logrus.FieldLogger, url string, data string, contentType string, headers map[string]string) (string, error) {
	logger = logger.WithField("url", url)
	transport := requester.GetHTTPTransport(logger)
	request := httputil.NewPostReq(logger, transport, contentType, 5, headers)
	response, err := request.Post(url, data)
	if err != nil {
		var certificateErr *tls.CertificateVerificationError
		if !errors.As(err, &certificateErr) {
			logger.WithError(err).Error("Failed to send HTTP POST request")
			return "", err
		}

		// tls.CertificateVerificationError encountered. Gonna re-accumulate root CA
		// certificate pool and retry requesting.
		logger.Info("certificate error, reload certificates and retry")
		transport = transport.Clone()
		requester.AccumulateRootCAs(logger)(func(certPool *x509.CertPool) bool {
			request = httputil.NewPostReq(logger, transport, contentType, 5, headers)
			request.SetTLSClient(&tls.Config{
				RootCAs: certPool,
			})
			if response, err = request.Post(url, data); err == nil {
				logger.Info("certificate updated")
				requester.RefreshHTTPCas(logger, certPool)
				return false
			}

			return true
		})
		if err != nil {
			logger.WithError(err).Error("Failed to send HTTP POST request")
			return "", err
		}
	}
	defer response.Close()

	content, _ := response.Content()
	if err == nil && response.StatusCode() > 400 {
		err = httpbase.NewStatusCodeError(response.StatusCode())
	}

	logger.WithFields(logrus.Fields{
		"url":             url,
		"responseCode":    response.StatusCode(),
		"responseContent": content,
	}).WithError(err).Infoln("HTTP POST Requested")
	return content, err
}
