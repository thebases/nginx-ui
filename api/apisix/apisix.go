package apisix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	uiSettings "github.com/0xJacky/Nginx-UI/settings"
	"github.com/go-resty/resty/v2"
	"github.com/uozi-tech/cosy/logger"
	cSettings "github.com/uozi-tech/cosy/settings"
)

const (
	HeaderAPIKey         = "x-api-key"
	headerRealIP         = "X-Real-IP"
	headerForwardedFor   = "X-Forwarded-For"
	headerForwardedProto = "X-Forwarded-Proto"
	defaultTimeout       = 15 * time.Second
)

// Request defines the APISIX Admin API request options.
type Request struct {
	Query   map[string]string
	Headers map[string]string
	Body    any
	Result  any
}

// Client is a lightweight APISIX Admin API client.
type Client struct {
	baseURL string
	apiKey  string
	client  *resty.Client
}

// NewClient creates a new APISIX Admin API client with base URL and admin key.
func NewClient(baseURL, apiKey string) (*Client, error) {
	normalizedBaseURL, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	trimmedAPIKey := strings.TrimSpace(apiKey)
	if trimmedAPIKey == "" {
		return nil, errors.New("APISIX Admin API key in config is required")
	}

	httpClient := resty.New()
	httpClient.SetBaseURL(normalizedBaseURL)
	httpClient.SetTimeout(defaultTimeout)
	httpClient.SetHeader("Accept", "application/json")
	httpClient.SetHeader("Content-Type", "application/json")
	httpClient.SetHeader(HeaderAPIKey, trimmedAPIKey)

	return &Client{
		baseURL: normalizedBaseURL,
		apiKey:  trimmedAPIKey,
		client:  httpClient,
	}, nil
}

// BaseURL returns the normalized base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// APIKey returns current APISIX admin key.
func (c *Client) APIKey() string {
	return c.apiKey
}

// SetTimeout updates request timeout for the APISIX Admin API client.
func (c *Client) SetTimeout(timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	c.client.SetTimeout(timeout)
}

// SetAPIKey updates the default APISIX admin key header.
func (c *Client) SetAPIKey(apiKey string) {
	trimmedKey := strings.TrimSpace(apiKey)
	c.apiKey = trimmedKey
	if trimmedKey == "" {
		c.client.Header.Del(HeaderAPIKey)
		return
	}
	c.client.SetHeader(HeaderAPIKey, trimmedKey)
}

// Do calls APISIX Admin API with provided options.
func (c *Client) Do(ctx context.Context, method, path string, req Request) (*resty.Response, error) {
	normalizedPath, err := normalizePathForBaseURL(c.baseURL, path)
	if err != nil {
		return nil, err
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, errors.New("APISIX Admin API key in config is required")
	}

	request := c.client.R()
	if ctx != nil {
		request.SetContext(ctx)
	}

	if len(req.Query) > 0 {
		request.SetQueryParams(req.Query)
	}
	if len(req.Headers) > 0 {
		request.SetHeaders(req.Headers)
	}
	request.Header.Del(headerRealIP)
	request.Header.Del(headerForwardedFor)
	request.Header.Del(headerForwardedProto)
	request.SetHeader(HeaderAPIKey, strings.TrimSpace(c.apiKey))
	if req.Body != nil {
		request.SetBody(req.Body)
	}
	if req.Result != nil {
		request.SetResult(req.Result)
	}

	targetURL := buildTargetURLForLog(c.baseURL, normalizedPath, req.Query)
	logger.Infof(
		"APISIX proxy outbound request method=%s url=%s headers=%s body=%s",
		method,
		targetURL,
		marshalHeadersForLog(request.Header),
		marshalBodyForLog(req.Body),
	)

	resp, err := request.Execute(method, normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("call APISIX Admin API %s %s failed: %w", method, normalizedPath, err)
	}
	logger.Infof("APISIX proxy outbound response status=%d headers=%s body=%s",
		resp.StatusCode(),
		marshalHeadersForLog(resp.Header()),
		resp.Body(),
	)

	return resp, nil
}

func (c *Client) Get(ctx context.Context, path string, req Request) (*resty.Response, error) {
	return c.Do(ctx, "GET", path, req)
}

func (c *Client) Post(ctx context.Context, path string, req Request) (*resty.Response, error) {
	return c.Do(ctx, "POST", path, req)
}

func (c *Client) Put(ctx context.Context, path string, req Request) (*resty.Response, error) {
	return c.Do(ctx, "PUT", path, req)
}

func (c *Client) Patch(ctx context.Context, path string, req Request) (*resty.Response, error) {
	return c.Do(ctx, "PATCH", path, req)
}

func (c *Client) Delete(ctx context.Context, path string, req Request) (*resty.Response, error) {
	return c.Do(ctx, "DELETE", path, req)
}

func normalizeBaseURL(baseURL string) (string, error) {
	trimmedURL := strings.TrimSpace(baseURL)
	if trimmedURL == "" {
		return "", errors.New("APISIX Admin API base URL is required")
	}

	parsedURL, err := url.Parse(trimmedURL)
	if err != nil {
		return "", fmt.Errorf("invalid APISIX Admin API base URL: %w", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", errors.New("APISIX Admin API base URL must include scheme and host")
	}

	return strings.TrimRight(parsedURL.String(), "/"), nil
}

func normalizePath(path string) (string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "", errors.New("APISIX Admin API path is required")
	}

	if strings.HasPrefix(trimmedPath, "http://") || strings.HasPrefix(trimmedPath, "https://") {
		return trimmedPath, nil
	}

	if strings.HasPrefix(trimmedPath, "/") {
		return trimmedPath, nil
	}

	return "/" + trimmedPath, nil
}

func normalizePathForBaseURL(baseURL, path string) (string, error) {
	normalizedPath, err := normalizePath(path)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(normalizedPath, "http://") || strings.HasPrefix(normalizedPath, "https://") {
		return normalizedPath, nil
	}

	parsedBaseURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid APISIX Admin API base URL: %w", err)
	}

	basePath := strings.TrimRight(parsedBaseURL.EscapedPath(), "/")

	// Map APISIX Admin API canonical path to configured gateway path behind proxy.
	if normalizedPath == "/apisix/admin" || strings.HasPrefix(normalizedPath, "/apisix/admin/") {
		replacePath := configuredReplacePath()
		normalizedPath = replacePath + strings.TrimPrefix(normalizedPath, "/apisix/admin")
		if normalizedPath == "" {
			normalizedPath = "/"
		}
	}

	if basePath == "" {
		return normalizedPath, nil
	}

	// Avoid duplicating the base path when callers pass a full APISIX path.
	if normalizedPath == basePath {
		normalizedPath = "/"
	} else if strings.HasPrefix(normalizedPath, basePath+"/") {
		normalizedPath = strings.TrimPrefix(normalizedPath, basePath)
	}

	// For base URLs that include a path (for example .../admin), use a relative
	// request path so the HTTP client preserves the base path.
	relativePath := strings.TrimPrefix(normalizedPath, "/")
	if relativePath == "" {
		return ".", nil
	}

	return relativePath, nil
}

func configuredReplacePath() string {
	candidates := []string{
		uiSettings.APISIXSettings.ReplacePath,
		iniSectionValue("apisix", "ReplacePath", "replace_path", "Replace_Path"),
	}

	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		return normalizeReplacePath(trimmed)
	}

	return "/admin"
}

func configuredBaseURL() string {
	return firstConfiguredValue(
		uiSettings.APISIXSettings.BaseURL,
		iniSectionValue("apisix", "BaseURL", "BaseUrl", "base_url"),
	)
}

func configuredAPIKey() string {
	return firstConfiguredValue(
		uiSettings.APISIXSettings.APIKey,
		iniSectionValue("apisix", "APIKey", "ApiKey", "api_key"),
	)
}

func firstConfiguredValue(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeReplacePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/admin"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	trimmed = strings.TrimRight(trimmed, "/")
	if trimmed == "" {
		return "/"
	}
	return trimmed
}

func iniSectionValue(section string, keys ...string) string {
	if cSettings.Conf == nil {
		return ""
	}
	sec := cSettings.Conf.Section(section)
	for _, key := range keys {
		value := strings.TrimSpace(sec.Key(key).String())
		if value != "" {
			return value
		}
	}
	return ""
}

func buildTargetURLForLog(baseURL, normalizedPath string, query map[string]string) string {
	target := normalizedPath

	if !strings.HasPrefix(normalizedPath, "http://") && !strings.HasPrefix(normalizedPath, "https://") {
		base, err := url.Parse(strings.TrimSpace(baseURL))
		if err == nil {
			ref, refErr := url.Parse(normalizedPath)
			if refErr == nil {
				target = base.ResolveReference(ref).String()
			}
		}
	}

	if len(query) == 0 {
		return target
	}

	parsedTarget, err := url.Parse(target)
	if err != nil {
		return target
	}
	values := parsedTarget.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	parsedTarget.RawQuery = values.Encode()
	return parsedTarget.String()
}

func marshalBodyForLog(body any) string {
	if body == nil {
		return "null"
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprintf("%+v", body)
	}

	const maxLogBytes = 4096
	if len(payload) > maxLogBytes {
		return string(payload[:maxLogBytes]) + "...(truncated)"
	}
	return string(payload)
}

func marshalHeadersForLog(headers map[string][]string) string {
	if len(headers) == 0 {
		return "{}"
	}

	safeHeaders := make(map[string][]string, len(headers))
	for key, values := range headers {
		if strings.EqualFold(key, HeaderAPIKey) {
			safeHeaders[key] = []string{"***"}
			continue
		}
		safeHeaders[key] = append([]string(nil), values...)
	}

	payload, err := json.Marshal(safeHeaders)
	if err != nil {
		return fmt.Sprintf("%+v", safeHeaders)
	}

	return string(payload)
}
