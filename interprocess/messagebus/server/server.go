package server

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"net/http"
	"strings"

	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

type RegisterFunc func(sr grpc.ServiceRegistrar)

func ListenAndServeGRPC(logger logrus.FieldLogger, endpoint buses.Endpoint, serveErr chan error, registers []RegisterFunc, opt ...grpc.ServerOption) (*grpc.Server, error) {
	listener, err := Listen(logger, endpoint)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
		}).WithError(err).Error("Failed to listen on specified endpoint")
		return nil, err
	}

	grpcServer := grpc.NewServer(opt...)
	for _, register := range registers {
		register(grpcServer)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.WithFields(logrus.Fields{
				"endpoint": endpoint,
			}).WithError(err).Error("Failed to start service on listening endpoint")
			if serveErr != nil {
				serveErr <- err
			}
		}
	}()
	return grpcServer, nil
}

func ListenAndServeHTTP(logger logrus.FieldLogger, endpoint buses.Endpoint, mux http.Handler, serveErr chan error, registers []RegisterFunc, opt ...grpc.ServerOption) (*http.Server, error) {
	listener, err := Listen(logger, endpoint)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
		}).WithError(err).Error("Failed to listen on specified endpoint")
		return nil, err
	}

	grpcServer := grpc.NewServer(opt...)
	for _, register := range registers {
		register(grpcServer)
	}

	server := &http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
				grpcServer.ServeHTTP(w, r)
			} else if mux != nil {
				mux.ServeHTTP(w, r)
			}
		}), &http2.Server{}),
	}
	server.SetKeepAlivesEnabled(false)

	go func() {
		if err := server.Serve(listener); err != nil {
			logger.WithFields(logrus.Fields{
				"endpoint": endpoint,
			}).WithError(err).Error("Failed to start server on listening endpoint")
			if serveErr != nil {
				serveErr <- err
			}
		}
	}()
	return server, nil
}
