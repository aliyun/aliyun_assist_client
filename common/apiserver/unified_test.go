package apiserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

func TestUnifiedDomainProvider_ServerDomain(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	logger := logrus.StandardLogger()

	t.Run("UseUnifiedDomainWithPresetIP", func(t *testing.T) {
		provider := &UnifiedDomainProvider{}

		// Mock detectPresetIP to return success
		patches := gomonkey.ApplyPrivateMethod(provider, "detectPresetIP", func(_ *UnifiedDomainProvider, logger logrus.FieldLogger) error {
			return nil
		})

		// Mock ConnectionDetect to return error (should not be called)
		connectionDetectCalled := false
		patches.ApplyFunc(ConnectionDetect, func(logger logrus.FieldLogger, domain string) error {
			connectionDetectCalled = true
			return nil
		})

		// Mock networkcategory.Set
		var networkCategorySet networkcategory.NetworkCategory
		patches.ApplyFunc(networkcategory.Set, func(category networkcategory.NetworkCategory) {
			networkCategorySet = category
		})
		defer patches.Reset()

		domain, err := provider.ServerDomain(logger)
		assert.NoError(t, err)
		assert.Equal(t, _unifiedDomain, domain)
		assert.True(t, provider.provided.Load())
		assert.True(t, provider.dialPresetIP.Load())
		assert.Equal(t, networkcategory.NetworkVPC, networkCategorySet)
		assert.False(t, connectionDetectCalled, "ConnectionDetect should not be called when detectPresetIP succeeds")
	})

	t.Run("UseUnifiedDomainAndResolve", func(t *testing.T) {
		provider := &UnifiedDomainProvider{}

		// Mock detectPresetIP to return error
		patches := gomonkey.ApplyPrivateMethod(provider, "detectPresetIP", func(_ *UnifiedDomainProvider, logger logrus.FieldLogger) error {
			return errUnknownDetectionResponse
		})

		// Mock ConnectionDetect to return success
		patches.ApplyFunc(ConnectionDetect, func(logger logrus.FieldLogger, domain string) error {
			return nil
		})

		// Mock networkcategory.Set
		var networkCategorySet networkcategory.NetworkCategory
		patches.ApplyFunc(networkcategory.Set, func(category networkcategory.NetworkCategory) {
			networkCategorySet = category
		})
		defer patches.Reset()

		domain, err := provider.ServerDomain(logger)
		assert.NoError(t, err)
		assert.Equal(t, _unifiedDomain, domain)
		assert.True(t, provider.provided.Load())
		assert.False(t, provider.dialPresetIP.Load())
		assert.Equal(t, networkcategory.NetworkVPC, networkCategorySet)
	})

	t.Run("Fallback", func(t *testing.T) {
		provider := &UnifiedDomainProvider{}

		// Mock detectPresetIP to return error
		patches := gomonkey.ApplyPrivateMethod(provider, "detectPresetIP", func(_ *UnifiedDomainProvider, logger logrus.FieldLogger) error {
			return errUnknownDetectionResponse
		})

		// Mock ConnectionDetect to return error
		patches.ApplyFunc(ConnectionDetect, func(logger logrus.FieldLogger, domain string) error {
			return errUnknownDetectionResponse
		})

		defer patches.Reset()

		domain, err := provider.ServerDomain(logger)
		assert.Empty(t, domain)
		assert.ErrorIs(t, err, requester.ErrNotProvided)
		assert.False(t, provider.provided.Load())
		assert.False(t, provider.dialPresetIP.Load())
	})
}

func TestUnifiedDomainProvider_detectPresetIP(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	logger := logrus.StandardLogger()

	t.Run("HttpGetReturnsOK", func(t *testing.T) {
		provider := &UnifiedDomainProvider{}

		// Mock httpGetWithTransport to return "ok"
		patches := gomonkey.ApplyFunc(httpGetWithTransport, func(logger logrus.FieldLogger, url string, headers map[string]string, transport *http.Transport) (string, error) {
			return "ok", nil
		})
		defer patches.Reset()

		err := provider.detectPresetIP(logger)
		assert.NoError(t, err)
	})

	t.Run("HttpGetReturnsNonOK", func(t *testing.T) {
		provider := &UnifiedDomainProvider{}

		// Mock httpGetWithTransport to return non-"ok"
		patches := gomonkey.ApplyFunc(httpGetWithTransport, func(logger logrus.FieldLogger, url string, headers map[string]string, transport *http.Transport) (string, error) {
			return "not-ok", nil
		})
		defer patches.Reset()

		err := provider.detectPresetIP(logger)
		assert.Error(t, err)
		assert.Equal(t, errUnknownDetectionResponse, err)
	})

	t.Run("HttpGetReturnsError", func(t *testing.T) {
		provider := &UnifiedDomainProvider{}

		// Mock httpGetWithTransport to return error
		expectedErr := errors.New("mock error")
		patches := gomonkey.ApplyFunc(httpGetWithTransport, func(logger logrus.FieldLogger, url string, headers map[string]string, transport *http.Transport) (string, error) {
			return "", expectedErr
		})
		defer patches.Reset()

		err := provider.detectPresetIP(logger)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, expectedErr, err)
	})
}

func TestUnifiedDomainProvider_ExtraHTTPHeaders(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	logger := logrus.StandardLogger()

	t.Run("NotProvided", func(t *testing.T) {
		provider := &UnifiedDomainProvider{}
		// Do not set provided flag

		headers, err := provider.ExtraHTTPHeaders(logger)
		assert.Error(t, err)
		assert.Equal(t, requester.ErrNotProvided, err)
		assert.Nil(t, headers)
	})

	t.Run("ProvidedSuccess", func(t *testing.T) {
		provider := &UnifiedDomainProvider{}
		provider.provided.Store(true)

		expectedHeaders := map[string]string{"header1": "value1"}
		patches := gomonkey.ApplyMethodFunc(&generalHTTPHeadersProvider, "ExtraHTTPHeaders", func(logger logrus.FieldLogger) (map[string]string, error) {
			return expectedHeaders, nil
		})
		defer patches.Reset()

		headers, err := provider.ExtraHTTPHeaders(logger)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeaders, headers)
	})
}

func TestUnifiedDomainProvider_DialContextFunc(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	logger := logrus.StandardLogger()

	t.Run("NotProvided", func(t *testing.T) {
		provider := &UnifiedDomainProvider{}
		// Do not set provided flag

		dialFunc, err := provider.DialContextFunc(logger, nil)
		assert.Error(t, err)
		assert.Equal(t, requester.ErrNotProvided, err)
		assert.Nil(t, dialFunc)
	})

	t.Run("ProvidedDialPresetIP", func(t *testing.T) {
		const expectedNetwork = "tcp"

		provider := &UnifiedDomainProvider{}
		provider.provided.Store(true)
		provider.dialPresetIP.Store(true)

		baseDialContextCalled := false
		baseDialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
			assert.Equal(t, expectedNetwork, network)
			// Preset IP should be applied on the address to dial via base function
			assert.Equal(t, _presetIP+":443", address)
			baseDialContextCalled = true
			return nil, nil
		}

		dialFunc, err := provider.DialContextFunc(logger, baseDialContext)
		assert.NoError(t, err)
		assert.NotNil(t, dialFunc)

		// Call the returned dial function
		conn, err := dialFunc(context.Background(), expectedNetwork, _unifiedDomain+":443")
		assert.NoError(t, err)
		assert.Nil(t, conn)
		assert.True(t, baseDialContextCalled)
	})

	t.Run("ProvidedNotDialPresetIP", func(t *testing.T) {
		const expectedNetwork = "tcp"
		const expectedAddress = _unifiedDomain+":443"

		provider := &UnifiedDomainProvider{}
		provider.provided.Store(true)
		// dialPresetIP is false by default

		baseDialContextCalled := false
		baseDialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
			assert.Equal(t, expectedNetwork, network)
			// Dialing address of the unified domain SHOULD be as is
			assert.Equal(t, expectedAddress, address)
			baseDialContextCalled = true
			return nil, nil
		}

		dialFunc, err := provider.DialContextFunc(logger, baseDialContext)
		assert.NoError(t, err)

		// Call the returned dial function
		conn, err := dialFunc(context.Background(), expectedNetwork, expectedAddress)
		assert.NoError(t, err)
		assert.Nil(t, conn)
		assert.True(t, baseDialContextCalled)
	})
}

func Test_wrapTransportDialContext(t *testing.T) {
	t.Run("NilReturnsNil", func(t *testing.T) {
		assert.Nil(t, _wrapTransportDialContext(nil))
	})

	t.Run("WrapNotNil", func(t *testing.T) {
		const expectedNetwork = "tcp"
		const expectedAddress = "localhost:8080"

		originalDialContextCalled := false
		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				originalDialContextCalled = true
				return nil, nil
			},
		}

		wrappedDialContextCalled := false
		patches := gomonkey.ApplyFunc(_newWrapDialContext, func(base func(ctx context.Context, network, address string) (net.Conn, error)) *_wrapDialContext {
			return &_wrapDialContext{
				base: func(ctx context.Context, network, address string) (net.Conn, error) {
					assert.Equal(t, expectedNetwork, network)
					assert.Equal(t, expectedAddress, address)
					wrappedDialContextCalled = true
					return nil, nil
				},
			}
		})
		defer patches.Reset()

		result := _wrapTransportDialContext(transport)
		assert.Equal(t, transport, result)

		conn, err := result.DialContext(context.Background(), expectedNetwork, expectedAddress)
		assert.Nil(t, conn)
		assert.NoError(t, err)
		assert.False(t, originalDialContextCalled)
		assert.True(t, wrappedDialContextCalled)
	})
}

func Test_newWrapDialContext(t *testing.T) {
	t.Run("WithBaseFunction", func(t *testing.T) {
		const expectedNetwork = "tcp"
		const expectedAddress = "localhost:80"

		baseCalled := false
		baseFunc := func(ctx context.Context, network, address string) (net.Conn, error) {
			assert.Equal(t, expectedNetwork, network)
			assert.Equal(t, expectedAddress, address)
			baseCalled = true
			return nil, nil
		}

		wrapDialContext := _newWrapDialContext(baseFunc)
		assert.NotNil(t, wrapDialContext)
		assert.NotNil(t, wrapDialContext.base)

		conn, err := wrapDialContext.DialContext(context.Background(), expectedNetwork, expectedAddress)
		assert.NoError(t, err)
		assert.Nil(t, conn)
		assert.True(t, baseCalled)
	})

	t.Run("WithNilBaseFunction", func(t *testing.T) {
		wrapDialContext := _newWrapDialContext(nil)
		assert.NotNil(t, wrapDialContext)
		assert.NotNil(t, wrapDialContext.base)

		// We can't easily test the actual dialer without causing real network connections
		// but we can verify the object was created successfully
	})
}

func Test_wrapDialContext_DialContext(t *testing.T) {
	t.Run("NonMatchingAddress", func(t *testing.T) {
		const expectedNetwork = "tcp"
		const expectedAddress = "localhost:80"

		baseCalled := false
		baseFunc := func(ctx context.Context, network, address string) (net.Conn, error) {
			baseCalled = true
			assert.Equal(t, expectedNetwork, network)
			assert.Equal(t, expectedAddress, address)
			return nil, nil
		}

		wrapDialContext := &_wrapDialContext{base: baseFunc}
		conn, err := wrapDialContext.DialContext(context.Background(), expectedNetwork, expectedAddress)
		assert.NoError(t, err)
		assert.Nil(t, conn)
		assert.True(t, baseCalled)
	})

	t.Run("MatchingAddress", func(t *testing.T) {
		baseCalled := false
		baseFunc := func(ctx context.Context, network, address string) (net.Conn, error) {
			baseCalled = true
			assert.Equal(t, "tcp", network)
			assert.Equal(t, _presetIP+":443", address)
			return nil, nil
		}

		wrapDialContext := &_wrapDialContext{base: baseFunc}
		conn, err := wrapDialContext.DialContext(context.Background(), "tcp", _unifiedDomain+":443")
		assert.NoError(t, err)
		assert.Nil(t, conn)
		assert.True(t, baseCalled)
	})

	t.Run("InvalidHostPort", func(t *testing.T) {
		wrapDialContext := &_wrapDialContext{base: nil}
		conn, err := wrapDialContext.DialContext(context.Background(), "tcp", "invalid-address")
		assert.Error(t, err)
		assert.Nil(t, conn)
	})
}