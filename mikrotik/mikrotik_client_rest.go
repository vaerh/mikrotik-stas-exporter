package mikrotik

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type RestClient struct {
	ctx       context.Context
	HostURL   string
	Username  string
	Password  string
	Transport TransportType
	*http.Client
}

type errorResponse struct {
	Detail  string `json:"detail"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

var (
	restMethodName = map[crudMethod]string{
		crudRead: "GET",
		crudPost: "POST",
	}
)

func (c *RestClient) GetTransport() TransportType {
	return c.Transport
}

func (c *RestClient) SendRequest(method crudMethod, url *URL) ([]MikrotikItem, error) {
	var data io.Reader

	// https://mikrotik + /rest + /interface/vlan + ? + .id=*39
	// Escaping spaces!
	requestUrl := c.HostURL + "/rest" + strings.Replace(url.GetRestURL(), " ", "%20", -1)
	LogMessage(c.ctx, DEBUG, restMethodName[method]+" request URL:  "+requestUrl)

	req, err := http.NewRequest(restMethodName[method], requestUrl, data)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.Username, c.Password)

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = res.Body.Close() }()

	body, _ := io.ReadAll(res.Body)

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		var errRes errorResponse

		LogMessage(c.ctx, DEBUG, fmt.Sprintf("error response body:\n%s", body))

		if err = json.Unmarshal(body, &errRes); err != nil {
			return nil, fmt.Errorf("json.Unmarshal - %v", err)
		} else {
			return nil, fmt.Errorf("%v '%v' returned response code: %v, message: '%v', details: '%v'",
				restMethodName[method], requestUrl, res.StatusCode, errRes.Message, errRes.Detail)
		}
	}

	LogMessage(c.ctx, TRACE, "response body: "+string(body))

	if len(body) > 2 {
		var result []MikrotikItem
		var rp any

		if body[0] == '[' && body[1] == '{' {
			rp = &result
		} else {
			result = make([]MikrotikItem, 1)
			rp = &result[0]
		}

		if err = json.Unmarshal(body, &rp); err != nil {
			if e, ok := err.(*json.SyntaxError); ok {
				LogMessage(c.ctx, DEBUG, fmt.Sprintf("json.Unmarshal(response body): syntax error at byte offset %d", e.Offset))

				if err = json.Unmarshal(EscapeChars(body), &rp); err != nil {
					return nil, fmt.Errorf("json.Unmarshal(response body): %v", err)
				}
			} else {
				return nil, err
			}
		}

		return result, nil
	}

	return nil, nil
}

func (c *RestClient) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}
