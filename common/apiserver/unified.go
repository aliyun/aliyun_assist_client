package apiserver

import (
	"context"
	"net"
	"net/http"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"go.uber.org/atomic"

	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	_unifiedDomain = "service.axt.aliyuncs.com"
	_presetIP      = "100.100.100.10"
)

type UnifiedDomainProvider struct {
	provided     atomic.Bool
	dialPresetIP atomic.Bool
}

func (*UnifiedDomainProvider) Name() string {
	return "UnifiedDomainProvider"
}

func (p *UnifiedDomainProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
	if err := p.detectPresetIP(logger); err == nil {
		p.provided.Store(true)
		p.dialPresetIP.Store(true)
		networkcategory.Set(networkcategory.NetworkVPC)
		return _unifiedDomain, nil
	} else {
		logger.WithFields(logrus.Fields{
			"domain": _unifiedDomain,
		}).WithError(err).Warning("Failed on detection of unified API server connection with preset IP")
	}

	if err := ConnectionDetect(logger, _unifiedDomain); err == nil {
		p.provided.Store(true)
		networkcategory.Set(networkcategory.NetworkVPC)
		return _unifiedDomain, nil
	} else {
		logger.WithFields(logrus.Fields{
			"domain": _unifiedDomain,
		}).WithError(err).Warning("Failed on detection of unified API server connection with DNS lookup")
	}

	return "", requester.ErrNotProvided
}

func (p *UnifiedDomainProvider) detectPresetIP(logger logrus.FieldLogger) error {
	const url = "https://" + _unifiedDomain + "/luban/api/connection_detect"

	transportToPresetIP := _wrapTransportDialContext(
		requester.GetHTTPTransport(logger, requester.WithProxy,
			requester.WithDefaultDialContextFunc, requester.WithRootCAs))
	content, err := httpGetWithTransport(logger, url, nil, transportToPresetIP)
	if err != nil {
		return err
	}

	if content == "ok" {
		return nil
	}

	return errUnknownDetectionResponse
}

func (p *UnifiedDomainProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	if !p.provided.Load() {
		return nil, requester.ErrNotProvided
	}

	return generalHTTPHeadersProvider.ExtraHTTPHeaders(logger)
}

func (p *UnifiedDomainProvider) DialContextFunc(logger logrus.FieldLogger, baseDialContext func(ctx context.Context, network, address string) (net.Conn, error)) (func(ctx context.Context, network, address string) (net.Conn, error), error) {
	if !p.provided.Load() {
		return nil, requester.ErrNotProvided
	}
	if p.dialPresetIP.Load() {
		return _newWrapDialContext(baseDialContext).DialContext, nil
	}

	return baseDialContext, nil
}

// _wrapTransportDialContext would replaces the DialContext function in t with
// the especially wrapped one, but only when t is not nil.
//
// Nil transport means working with [net/http.DefaultHTTPTransport], which
// SHOULD NOT hold the wrapped DialContext function
func _wrapTransportDialContext(t *http.Transport) *http.Transport {
	if t != nil {
		t.DialContext = _newWrapDialContext(t.DialContext).DialContext
	}

	return t
}

type _wrapDialContext struct {
	base func(ctx context.Context, network, address string) (net.Conn, error)
}

func _newWrapDialContext(base func(ctx context.Context, network, address string) (net.Conn, error)) *_wrapDialContext {
	if base == nil {
		base = (&net.Dialer{}).DialContext
	}

	return &_wrapDialContext{
		base: base,
	}
}

func (w *_wrapDialContext) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if host == _unifiedDomain {
		address = net.JoinHostPort(_presetIP, port)
	}

	return w.base(ctx, network, address)
}
