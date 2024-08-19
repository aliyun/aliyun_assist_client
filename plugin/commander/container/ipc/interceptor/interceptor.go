package interceptor

import (
	"context"

	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/idlecheck"
	"google.golang.org/grpc"
)

// KeepProcessAlive is a interceptor: any request will extand process live time
func KeepProcessAlive(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	idlecheck.ExtendLive()
	return handler(ctx, req)
}
