package ilinkapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wonli/apic/v2"
)

// callAPI 统一执行 API 请求并返回底层响应，适合需要读取响应头或原始字节的场景。
func (c *Client) callAPI(ctx context.Context, id *apic.ApiId, options *apic.Options) (*apic.ResponseData, error) {
	callClient := apic.NewApiClient().
		WithHTTPClient(c.apiClient.HTTPClient()).
		WithContext(ctx)
	apiResp, err := callClient.CallApi(id, options)
	if err != nil {
		return nil, err
	}
	if apiResp == nil {
		return nil, fmt.Errorf("empty response")
	}
	if apiResp.HttpStatus != 200 {
		return nil, fmt.Errorf("http status %d: %s", apiResp.HttpStatus, string(apiResp.Data))
	}
	return apiResp, nil
}

// callJSON 统一执行 API 请求，并把返回 JSON 解码到 resp。
// 各业务方法只需要关注请求参数和业务错误判断，底层 HTTP/解码逻辑集中在这里。
func (c *Client) callJSON(ctx context.Context, id *apic.ApiId, resp any, options *apic.Options) error {
	apiResp, err := c.callAPI(ctx, id, options)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(apiResp.Data, resp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
