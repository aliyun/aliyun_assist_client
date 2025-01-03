package httpbase

func Get(url string, requestOptions... RequestOption) (string, error) {
	requestOptionsPrepended := append([]RequestOption{
		WithHeaders(map[string]string{
			UserAgentHeader: UserAgentValue,
		}),
		WithTimeoutInSeconds(5),
	}, requestOptions...)
	request := buildRequest(requestOptionsPrepended...)

	response, err := request.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Close()

	if response.StatusCode() > 400 {
		return "", NewStatusCodeError(response.StatusCode())
	}
	return response.Content()
}

func Put(url string, requestOptions... RequestOption) (string, error) {
	requestOptionsPrepended := append([]RequestOption{
		WithHeaders(map[string]string{
			UserAgentHeader: UserAgentValue,
		}),
		WithTimeoutInSeconds(5),
	}, requestOptions...)
	request := buildRequest(requestOptionsPrepended...)

	response, err := request.Put(url)
	if err != nil {
		return "", err
	}
	defer response.Close()

	if response.StatusCode() > 400 {
		return "", NewStatusCodeError(response.StatusCode())
	}
	return response.Content()
}
