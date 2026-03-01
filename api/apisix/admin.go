package apisix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	uiSettings "github.com/0xJacky/Nginx-UI/settings"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/uozi-tech/cosy"
)

const (
	envAPISIXAdminAPIAddr = "APISIX_ADMIN_API_ADDR"
	envAPISIXAdminKey     = "APISIX_ADMIN_KEY"
	envNginxUIAPIAddr     = "NGINX_UI_APISIX_ADMIN_API_ADDR"
	envNginxUIAdminKey    = "NGINX_UI_APISIX_ADMIN_KEY"

	pageSizeMin = 10
	pageSizeMax = 500

	apiRoutes         = "/apisix/admin/routes"
	apiStreamRoutes   = "/apisix/admin/stream_routes"
	apiUpstreams      = "/apisix/admin/upstreams"
	apiProtos         = "/apisix/admin/protos"
	apiServices       = "/apisix/admin/services"
	apiGlobalRules    = "/apisix/admin/global_rules"
	apiPlugins        = "/apisix/admin/plugins"
	apiPluginsList    = "/apisix/admin/plugins/list"
	apiPluginMetadata = "/apisix/admin/plugin_metadata"
	apiSecrets        = "/apisix/admin/secrets"
	apiConsumers      = "/apisix/admin/consumers"
	apiConsumerGroups = "/apisix/admin/consumer_groups"
	apiCredentials    = "/apisix/admin/consumers/%s/credentials"
	apiSSLs           = "/apisix/admin/ssls"
	apiPluginConfigs  = "/apisix/admin/plugin_configs"
)

var allowedMethods = map[string]struct{}{
	http.MethodGet:    {},
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},
}

type requestAdminAPIRequest struct {
	BaseURL string            `json:"base_url"`
	APIKey  string            `json:"api_key"`
	Method  string            `json:"method" binding:"required"`
	Path    string            `json:"path" binding:"required"`
	Query   map[string]string `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

type apisixListItem struct {
	Value map[string]any `json:"value"`
}

type apisixListResponse struct {
	List []apisixListItem `json:"list"`
}

func RequestAdminAPI(c *gin.Context) {
	var req requestAdminAPIRequest
	if !cosy.BindAndValid(c, &req) {
		return
	}

	baseURL := strings.TrimSpace(req.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(uiSettings.APISIXSettings.BaseURL)
	}
	if baseURL == "" {
		baseURL = firstNonEmptyEnv(envNginxUIAPIAddr, envAPISIXAdminAPIAddr)
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(uiSettings.APISIXSettings.APIKey)
	}
	if apiKey == "" {
		apiKey = firstNonEmptyEnv(envNginxUIAdminKey, envAPISIXAdminKey)
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if _, ok := allowedMethods[method]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid method, allowed values: GET, POST, PUT, PATCH, DELETE",
		})
		return
	}

	client, err := NewClient(baseURL, apiKey)
	if err != nil {
		cosy.ErrHandler(c, err)
		return
	}

	resp, err := client.Do(c.Request.Context(), method, req.Path, Request{
		Query:   req.Query,
		Headers: req.Headers,
		Body:    req.Body,
	})
	if err != nil {
		cosy.ErrHandler(c, err)
		return
	}

	contentType := resp.Header().Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	c.Data(resp.StatusCode(), contentType, resp.Body())
}

func GetConsumerListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiConsumers, Request{Query: params})
}

func GetConsumerReq(ctx context.Context, client *Client, username string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiConsumers, escapePathParam(username)), Request{})
}

func PutConsumerReq(ctx context.Context, client *Client, body any) (*resty.Response, error) {
	return client.Put(ctx, apiConsumers, Request{Body: body})
}

func DeleteAllConsumers(ctx context.Context, client *Client) error {
	return deleteAllByField(ctx, client, apiConsumers, "username")
}

func GetConsumerGroupListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiConsumerGroups, Request{Query: params})
}

func GetConsumerGroupReq(ctx context.Context, client *Client, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiConsumerGroups, escapePathParam(id)), Request{})
}

func PutConsumerGroupReq(ctx context.Context, client *Client, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiConsumerGroups, escapePathParam(id)), Request{Body: body})
}

func DeleteAllConsumerGroups(ctx context.Context, client *Client) error {
	return deleteAllByField(ctx, client, apiConsumerGroups, "id")
}

func GetCredentialListReq(ctx context.Context, client *Client, username string, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf(apiCredentials, escapePathParam(username)), Request{Query: params})
}

func GetCredentialReq(ctx context.Context, client *Client, username, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf(apiCredentials, escapePathParam(username))+"/"+escapePathParam(id), Request{})
}

func PutCredentialReq(ctx context.Context, client *Client, username, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf(apiCredentials, escapePathParam(username))+"/"+escapePathParam(id), Request{Body: body})
}

func GetRouteListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiRoutes, Request{Query: params})
}

func GetRouteReq(ctx context.Context, client *Client, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiRoutes, escapePathParam(id)), Request{})
}

func PutRouteReq(ctx context.Context, client *Client, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiRoutes, escapePathParam(id)), Request{Body: body})
}

func PostRouteReq(ctx context.Context, client *Client, body any) (*resty.Response, error) {
	return client.Post(ctx, apiRoutes, Request{Body: body})
}

func DeleteAllRoutes(ctx context.Context, client *Client) error {
	return deleteAllByField(ctx, client, apiRoutes, "id")
}

func GetServiceListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiServices, Request{Query: params})
}

func GetServiceReq(ctx context.Context, client *Client, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiServices, escapePathParam(id)), Request{})
}

func PutServiceReq(ctx context.Context, client *Client, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiServices, escapePathParam(id)), Request{Body: body})
}

func PostServiceReq(ctx context.Context, client *Client, body any) (*resty.Response, error) {
	return client.Post(ctx, apiServices, Request{Body: body})
}

func DeleteAllServices(ctx context.Context, client *Client) error {
	if err := DeleteAllRoutes(ctx, client); err != nil {
		return err
	}
	if err := DeleteAllStreamRoutes(ctx, client); err != nil {
		return err
	}
	return deleteAllByField(ctx, client, apiServices, "id")
}

func GetUpstreamListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiUpstreams, Request{Query: params})
}

func GetUpstreamReq(ctx context.Context, client *Client, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiUpstreams, escapePathParam(id)), Request{})
}

func PostUpstreamReq(ctx context.Context, client *Client, body any) (*resty.Response, error) {
	return client.Post(ctx, apiUpstreams, Request{Body: body})
}

func PutUpstreamReq(ctx context.Context, client *Client, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiUpstreams, escapePathParam(id)), Request{Body: body})
}

func DeleteAllUpstreams(ctx context.Context, client *Client) error {
	return deleteAllByField(ctx, client, apiUpstreams, "id")
}

func GetStreamRouteListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiStreamRoutes, Request{Query: params})
}

func GetStreamRouteReq(ctx context.Context, client *Client, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiStreamRoutes, escapePathParam(id)), Request{})
}

func PutStreamRouteReq(ctx context.Context, client *Client, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiStreamRoutes, escapePathParam(id)), Request{Body: body})
}

func PostStreamRouteReq(ctx context.Context, client *Client, body any) (*resty.Response, error) {
	return client.Post(ctx, apiStreamRoutes, Request{Body: body})
}

func DeleteAllStreamRoutes(ctx context.Context, client *Client) error {
	return deleteAllByField(ctx, client, apiStreamRoutes, "id")
}

func GetSSLListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiSSLs, Request{Query: params})
}

func GetSSLReq(ctx context.Context, client *Client, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiSSLs, escapePathParam(id)), Request{})
}

func PutSSLReq(ctx context.Context, client *Client, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiSSLs, escapePathParam(id)), Request{Body: body})
}

func PostSSLReq(ctx context.Context, client *Client, body any) (*resty.Response, error) {
	return client.Post(ctx, apiSSLs, Request{Body: body})
}

func DeleteAllSSLs(ctx context.Context, client *Client) error {
	return deleteAllByField(ctx, client, apiSSLs, "id")
}

func GetGlobalRuleListReq(ctx context.Context, client *Client) (*resty.Response, error) {
	return client.Get(ctx, apiGlobalRules, Request{})
}

func GetGlobalRuleReq(ctx context.Context, client *Client, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiGlobalRules, escapePathParam(id)), Request{})
}

func PutGlobalRuleReq(ctx context.Context, client *Client, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiGlobalRules, escapePathParam(id)), Request{Body: body})
}

func GetPluginConfigListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiPluginConfigs, Request{Query: params})
}

func GetPluginConfigReq(ctx context.Context, client *Client, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiPluginConfigs, escapePathParam(id)), Request{})
}

func PutPluginConfigReq(ctx context.Context, client *Client, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiPluginConfigs, escapePathParam(id)), Request{Body: body})
}

func GetProtoListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiProtos, Request{Query: params})
}

func GetProtoReq(ctx context.Context, client *Client, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiProtos, escapePathParam(id)), Request{})
}

func PutProtoReq(ctx context.Context, client *Client, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiProtos, escapePathParam(id)), Request{Body: body})
}

func PostProtoReq(ctx context.Context, client *Client, body any) (*resty.Response, error) {
	return client.Post(ctx, apiProtos, Request{Body: body})
}

func GetSecretListReq(ctx context.Context, client *Client, params map[string]string) (*resty.Response, error) {
	return client.Get(ctx, apiSecrets, Request{Query: params})
}

func GetSecretReq(ctx context.Context, client *Client, manager, id string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s/%s", apiSecrets, escapePathParam(manager), escapePathParam(id)), Request{})
}

func PutSecretReq(ctx context.Context, client *Client, manager, id string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s/%s", apiSecrets, escapePathParam(manager), escapePathParam(id)), Request{Body: body})
}

func GetPluginsListReq(ctx context.Context, client *Client) (*resty.Response, error) {
	return client.Get(ctx, apiPluginsList, Request{})
}

func GetPluginsListWithSchemaReq(ctx context.Context, client *Client, subsystem string) (*resty.Response, error) {
	query := map[string]string{"all": "true"}
	if strings.TrimSpace(subsystem) != "" {
		query["subsystem"] = strings.TrimSpace(subsystem)
	}
	return client.Get(ctx, apiPlugins, Request{Query: query})
}

func GetPluginSchemaReq(ctx context.Context, client *Client, name string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiPlugins, escapePathParam(name)), Request{})
}

func PutPluginMetadataReq(ctx context.Context, client *Client, name string, body any) (*resty.Response, error) {
	return client.Put(ctx, fmt.Sprintf("%s/%s", apiPluginMetadata, escapePathParam(name)), Request{Body: body})
}

func DeletePluginMetadataReq(ctx context.Context, client *Client, name string) (*resty.Response, error) {
	return client.Delete(ctx, fmt.Sprintf("%s/%s", apiPluginMetadata, escapePathParam(name)), Request{})
}

func GetPluginMetadataReq(ctx context.Context, client *Client, pluginName string, headers map[string]string) (*resty.Response, error) {
	return client.Get(ctx, fmt.Sprintf("%s/%s", apiPluginMetadata, escapePathParam(pluginName)), Request{Headers: headers})
}

func deleteAllByField(ctx context.Context, client *Client, listPath, field string) error {
	for {
		resp, err := client.Get(ctx, listPath, Request{
			Query: map[string]string{
				"page":      "1",
				"page_size": strconv.Itoa(pageSizeMax),
			},
		})
		if err != nil {
			return err
		}

		ids, err := extractFieldValues(resp.Body(), field)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}

		for _, id := range ids {
			_, err = client.Delete(ctx, fmt.Sprintf("%s/%s", listPath, escapePathParam(id)), Request{})
			if err != nil {
				return err
			}
		}
	}
}

func extractFieldValues(body []byte, field string) ([]string, error) {
	var result apisixListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	values := make([]string, 0, len(result.List))
	for _, item := range result.List {
		raw, ok := item.Value[field]
		if !ok {
			continue
		}
		value := fmt.Sprintf("%v", raw)
		if value == "" {
			continue
		}
		values = append(values, value)
	}

	return values, nil
}

func escapePathParam(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}
