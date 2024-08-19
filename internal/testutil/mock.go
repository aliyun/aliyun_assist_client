package testutil

import (
	"fmt"

	"github.com/jarcoal/httpmock"
)

// MockMetaServer works only after mock requester.GetHTTPTransport like:
// gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
// 	transport, _ := http.DefaultTransport.(*http.Transport)
// 	return transport
// })
func MockMetaServer(region_id string) {
	httpmock.RegisterResponder("GET", "http://100.100.100.200/latest/meta-data/region-id",
		httpmock.NewStringResponder(200, region_id))
	httpmock.RegisterResponder("GET", fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/connection_detect", region_id),
		httpmock.NewStringResponder(200, "ok"))
}
