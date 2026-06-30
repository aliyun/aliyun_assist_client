package channel

type ErrorResponseHandler func(statusCode int, content string, err error)

var (
	websocketDialErrorResponseHook ErrorResponseHandler
)

func RegisterWebsocketDialErrorResponseHook(handler ErrorResponseHandler) {
	websocketDialErrorResponseHook = handler
}
