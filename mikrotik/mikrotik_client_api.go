package mikrotik

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-routeros/routeros"
)

type ApiClient struct {
	ctx       context.Context
	HostURL   string
	Username  string
	Password  string
	Transport TransportType
	*routeros.Client
}

var (
	apiMethodName = map[CrudMethod]string{
		CrudRead:    "/print",
		CrudPost:    "/set",
		CrudMonitor: "/monitor",
	}
)

func (c *ApiClient) GetTransport() TransportType {
	return c.Transport
}

func (c *ApiClient) SendRequest(method CrudMethod, url *URL, data map[string]string) ([]MikrotikItem, error) {

	// https://help.mikrotik.com/docs/display/ROS/API
	// /interface/vlan/print + '?.id=*39' + '?type=vlan'
	cmd := url.GetApiCmd()
	// The first element is the Path
	cmd[0] += apiMethodName[method]

	for k, v := range data {
		cmd = append(cmd, fmt.Sprintf("=%v=%v", k, v))
	}
	LogMessage(c.ctx, DEBUG, "request CMD:  "+strings.Join(cmd, ""))

	resp, err := c.RunArgs(cmd)
	if err != nil {
		return nil, err
	}

	LogMessage(c.ctx, TRACE, "response body: "+resp.String())

	// Unmarshal
	var res []MikrotikItem

	for _, sentence := range resp.Re {
		m := MikrotikItem{}
		for k, v := range sentence.Map {
			m[k] = v
		}
		res = append(res, m)
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res, nil
}

func (c *ApiClient) WithContext(ctx context.Context) context.Context {
	if _, ok := ctx.Value(ctxKey{}).(*ApiClient); !ok {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, c)
}
