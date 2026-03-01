package apisix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	uiSettings "github.com/0xJacky/Nginx-UI/settings"
	"github.com/gin-gonic/gin"
	"github.com/uozi-tech/cosy"
	cSettings "github.com/uozi-tech/cosy/settings"
)

const (
	headerOverrideBaseURL = "X-APISIX-BASE-URL"
	headerOverrideAPIKey  = "X-APISIX-API-KEY"
)

var (
	openAPILoadOnce sync.Once
	openAPIDoc      map[string]any
	openAPILoadErr  error
)

func ProxyAdminAPI(c *gin.Context) {
	method := strings.ToUpper(strings.TrimSpace(c.Request.Method))
	if _, ok := allowedMethods[method]; !ok {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"message": "invalid method, allowed values: GET, POST, PUT, PATCH, DELETE",
		})
		return
	}

	resourcePath := strings.Trim(c.Param("path"), "/")
	if resourcePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "APISIX resource path is required"})
		return
	}

	requestBody, err := c.GetRawData()
	if err != nil {
		cosy.ErrHandler(c, err)
		return
	}

	openAPIPath := "/apisix/admin/" + resourcePath
	if err = validateRequestByOpenAPI(method, openAPIPath, requestBody, c.GetHeader("Content-Type")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	baseURL, err := resolveBaseURL(
		c.GetHeader(headerOverrideBaseURL),
		uiSettings.APISIXSettings.BaseURL,
		firstNonEmptyINI("apisix", "BaseURL", "BaseUrl", "base_url"),
		firstNonEmptyEnv(envNginxUIAPIAddr, envAPISIXAdminAPIAddr),
	)
	if err != nil {
		cosy.ErrHandler(c, err)
		return
	}

	apiKey := strings.TrimSpace(c.GetHeader(headerOverrideAPIKey))
	if apiKey == "" {
		apiKey = strings.TrimSpace(uiSettings.APISIXSettings.APIKey)
	}
	if apiKey == "" {
		apiKey = firstNonEmptyINI("apisix", "APIKey", "ApiKey", "api_key")
	}
	if apiKey == "" {
		apiKey = firstNonEmptyEnv(envNginxUIAdminKey, envAPISIXAdminKey)
	}

	client, err := NewClient(baseURL, apiKey)
	if err != nil {
		cosy.ErrHandler(c, err)
		return
	}

	query := flattenQuery(c.Request.URL.Query())
	headers := collectForwardHeaders(c.Request.Header)
	upstreamPath := "/apisix/admin/" + resourcePath

	var forwardBody any
	if len(bytes.TrimSpace(requestBody)) > 0 {
		if strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
			forwardBody = json.RawMessage(requestBody)
		} else {
			forwardBody = requestBody
		}
	}

	resp, err := client.Do(c.Request.Context(), method, upstreamPath, Request{
		Query:   query,
		Headers: headers,
		Body:    forwardBody,
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

func flattenQuery(values url.Values) map[string]string {
	if len(values) == 0 {
		return nil
	}

	out := make(map[string]string, len(values))
	for key, items := range values {
		if len(items) == 0 {
			continue
		}
		out[key] = items[0]
	}
	return out
}

func collectForwardHeaders(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	allowed := map[string]struct{}{
		"accept":        {},
		"content-type":  {},
		"if-match":      {},
		"if-none-match": {},
	}

	ignored := map[string]struct{}{
		strings.ToLower(HeaderAPIKey):         {},
		strings.ToLower(headerOverrideBaseURL): {},
		strings.ToLower(headerOverrideAPIKey):  {},
	}

	out := make(map[string]string)
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		lowerKey := strings.ToLower(key)
		if _, skip := ignored[lowerKey]; skip {
			continue
		}
		if _, ok := allowed[lowerKey]; !ok && !strings.HasPrefix(lowerKey, "x-") {
			continue
		}
		out[key] = values[0]
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func validateRequestByOpenAPI(method, path string, body []byte, contentType string) error {
	doc, err := loadOpenAPISpec()
	if err != nil {
		return err
	}

	op, ok := findOperation(doc, path, method)
	if !ok {
		return fmt.Errorf("path %s with method %s is not defined in openapi.json", path, method)
	}

	trimmedBody := bytes.TrimSpace(body)
	reqBody := getRequestBody(doc, op)
	if reqBody == nil {
		if len(trimmedBody) > 0 {
			return fmt.Errorf("request body is not allowed for %s %s by openapi.json", method, path)
		}
		return nil
	}

	required, _ := reqBody["required"].(bool)
	if len(trimmedBody) == 0 {
		if required {
			return fmt.Errorf("request body is required for %s %s by openapi.json", method, path)
		}
		return nil
	}

	content := toMap(reqBody["content"])
	if len(content) == 0 {
		return nil
	}

	mediaType := normalizeContentType(contentType)
	mediaObj := toMap(content[mediaType])
	if len(mediaObj) == 0 {
		mediaObj = toMap(content["application/json"])
	}
	if len(mediaObj) == 0 {
		for key, raw := range content {
			if strings.HasPrefix(key, "application/") || key == "*/*" {
				mediaObj = toMap(raw)
				break
			}
		}
	}
	if len(mediaObj) == 0 {
		return fmt.Errorf("content type %q is not allowed by openapi.json for %s %s", mediaType, method, path)
	}

	schema := schemaMapFromRaw(mediaObj["schema"])
	if schema == nil {
		return nil
	}

	var payload any
	if err = json.Unmarshal(trimmedBody, &payload); err != nil {
		return fmt.Errorf("request body is not valid JSON: %w", err)
	}

	return validateJSONSchema(doc, schema, payload, "$")
}

func normalizeContentType(contentType string) string {
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if contentType == "" {
		return "application/json"
	}
	return contentType
}

func loadOpenAPISpec() (map[string]any, error) {
	openAPILoadOnce.Do(func() {
		paths := []string{
			filepath.Join(".", "openapi.json"),
			filepath.Join(".", "nginx-ui", "openapi.json"),
		}

		var raw []byte
		var err error
		for _, p := range paths {
			raw, err = os.ReadFile(p)
			if err == nil {
				break
			}
		}
		if err != nil {
			openAPILoadErr = fmt.Errorf("cannot read openapi.json: %w", err)
			return
		}

		if err = json.Unmarshal(raw, &openAPIDoc); err != nil {
			openAPILoadErr = fmt.Errorf("cannot parse openapi.json: %w", err)
			return
		}
	})

	return openAPIDoc, openAPILoadErr
}

func findOperation(doc map[string]any, requestPath, method string) (map[string]any, bool) {
	method = strings.ToLower(method)
	paths := toMap(doc["paths"])
	if len(paths) == 0 {
		return nil, false
	}

	if operations := toMap(paths[requestPath]); len(operations) > 0 {
		if op := toMap(operations[method]); len(op) > 0 {
			return op, true
		}
	}

	for pattern, rawOps := range paths {
		if !matchOpenAPIPath(pattern, requestPath) {
			continue
		}
		operations := toMap(rawOps)
		if op := toMap(operations[method]); len(op) > 0 {
			return op, true
		}
	}
	return nil, false
}

func getRequestBody(doc map[string]any, operation map[string]any) map[string]any {
	if operation == nil {
		return nil
	}

	reqBody := toMap(operation["requestBody"])
	if len(reqBody) > 0 {
		return reqBody
	}

	if ref, ok := operation["requestBody"].(map[string]any); ok {
		if refStr, ok := ref["$ref"].(string); ok {
			return resolveRef(doc, refStr)
		}
	}

	if refStr, ok := operation["requestBody"].(string); ok {
		return resolveRef(doc, refStr)
	}

	return nil
}

func matchOpenAPIPath(pattern, requestPath string) bool {
	trim := func(s string) string {
		return strings.Trim(strings.TrimSpace(s), "/")
	}

	pattern = trim(pattern)
	requestPath = trim(requestPath)
	if pattern == "" && requestPath == "" {
		return true
	}

	patternParts := strings.Split(pattern, "/")
	requestParts := strings.Split(requestPath, "/")
	if len(patternParts) != len(requestParts) {
		return false
	}

	for i := range patternParts {
		p := patternParts[i]
		r := requestParts[i]
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			if r == "" {
				return false
			}
			continue
		}
		if p != r {
			return false
		}
	}
	return true
}

func validateJSONSchema(doc map[string]any, schema map[string]any, value any, path string) error {
	resolved := resolveSchema(doc, schema, map[string]struct{}{})
	if resolved == nil {
		return nil
	}

	if enumValues, ok := resolved["enum"].([]any); ok && len(enumValues) > 0 {
		match := false
		for _, candidate := range enumValues {
			if fmt.Sprint(candidate) == fmt.Sprint(value) {
				match = true
				break
			}
		}
		if !match {
			return fmt.Errorf("%s: value %v not in enum", path, value)
		}
	}

	if oneOf, ok := resolved["oneOf"].([]any); ok && len(oneOf) > 0 {
		matches := 0
		for _, item := range oneOf {
			itemSchema := schemaMapFromRaw(item)
			if itemSchema == nil {
				continue
			}
			if validateJSONSchema(doc, itemSchema, value, path) == nil {
				matches++
			}
		}
		if matches != 1 {
			return fmt.Errorf("%s: value does not satisfy exactly one oneOf schema", path)
		}
		return nil
	}

	if anyOf, ok := resolved["anyOf"].([]any); ok && len(anyOf) > 0 {
		for _, item := range anyOf {
			itemSchema := schemaMapFromRaw(item)
			if itemSchema == nil {
				continue
			}
			if validateJSONSchema(doc, itemSchema, value, path) == nil {
				return nil
			}
		}
		return fmt.Errorf("%s: value does not satisfy any anyOf schema", path)
	}

	if allOf, ok := resolved["allOf"].([]any); ok && len(allOf) > 0 {
		for _, item := range allOf {
			itemSchema := schemaMapFromRaw(item)
			if itemSchema == nil {
				continue
			}
			if err := validateJSONSchema(doc, itemSchema, value, path); err != nil {
				return err
			}
		}
		return nil
	}

	if !checkType(resolved["type"], value) {
		return fmt.Errorf("%s: invalid type", path)
	}

	objectValue, isObject := value.(map[string]any)
	if isObject {
		if required, ok := resolved["required"].([]any); ok {
			for _, key := range required {
				name, ok := key.(string)
				if !ok {
					continue
				}
				if _, exists := objectValue[name]; !exists {
					return fmt.Errorf("%s.%s: required field is missing", path, name)
				}
			}
		}

		properties := toMap(resolved["properties"])
		additionalAllowed := true
		if rawAdditional, exists := resolved["additionalProperties"]; exists {
			if b, ok := rawAdditional.(bool); ok {
				additionalAllowed = b
			}
		}

		for key, item := range objectValue {
			if propSchema := schemaMapFromRaw(properties[key]); propSchema != nil {
				if err := validateJSONSchema(doc, propSchema, item, path+"."+key); err != nil {
					return err
				}
				continue
			}
			if !additionalAllowed {
				return fmt.Errorf("%s.%s: field is not allowed by schema", path, key)
			}
		}
	}

	if arrayValue, ok := value.([]any); ok {
		if minItems, ok := toInt(resolved["minItems"]); ok && len(arrayValue) < minItems {
			return fmt.Errorf("%s: array has fewer than minItems", path)
		}
		if maxItems, ok := toInt(resolved["maxItems"]); ok && len(arrayValue) > maxItems {
			return fmt.Errorf("%s: array has more than maxItems", path)
		}

		itemSchema := schemaMapFromRaw(resolved["items"])
		if itemSchema != nil {
			for idx, item := range arrayValue {
				if err := validateJSONSchema(doc, itemSchema, item, fmt.Sprintf("%s[%d]", path, idx)); err != nil {
					return err
				}
			}
		}
	}

	if stringValue, ok := value.(string); ok {
		if minLength, ok := toInt(resolved["minLength"]); ok && len(stringValue) < minLength {
			return fmt.Errorf("%s: string shorter than minLength", path)
		}
		if maxLength, ok := toInt(resolved["maxLength"]); ok && len(stringValue) > maxLength {
			return fmt.Errorf("%s: string longer than maxLength", path)
		}
		if rawPattern, ok := resolved["pattern"].(string); ok && rawPattern != "" {
			re, err := regexp.Compile(rawPattern)
			if err == nil && !re.MatchString(stringValue) {
				return fmt.Errorf("%s: string does not match pattern", path)
			}
		}
	}

	if numberValue, ok := toFloat(value); ok {
		if min, ok := toFloat(resolved["minimum"]); ok && numberValue < min {
			return fmt.Errorf("%s: number less than minimum", path)
		}
		if max, ok := toFloat(resolved["maximum"]); ok && numberValue > max {
			return fmt.Errorf("%s: number greater than maximum", path)
		}
	}

	return nil
}

func resolveSchema(doc map[string]any, schema map[string]any, seen map[string]struct{}) map[string]any {
	if schema == nil {
		return nil
	}

	ref, ok := schema["$ref"].(string)
	if !ok || ref == "" {
		return schema
	}

	if _, exists := seen[ref]; exists {
		return schema
	}
	seen[ref] = struct{}{}

	resolved := resolveRef(doc, ref)
	if resolved == nil {
		return schema
	}
	return resolveSchema(doc, resolved, seen)
}

func resolveRef(doc map[string]any, ref string) map[string]any {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}

	current := any(doc)
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		node, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		value, exists := node[part]
		if !exists {
			return nil
		}
		current = value
	}

	return toMap(current)
}

func schemaMapFromRaw(raw any) map[string]any {
	if raw == nil {
		return nil
	}
	if schema := toMap(raw); len(schema) > 0 {
		return schema
	}
	if ref, ok := raw.(string); ok && strings.HasPrefix(ref, "#/") {
		return map[string]any{"$ref": ref}
	}
	return nil
}

func toMap(value any) map[string]any {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func checkType(rawType any, value any) bool {
	if rawType == nil {
		return true
	}

	switch t := rawType.(type) {
	case string:
		return checkSingleType(t, value)
	case []any:
		for _, item := range t {
			typeName, ok := item.(string)
			if ok && checkSingleType(typeName, value) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func checkSingleType(typeName string, value any) bool {
	switch typeName {
	case "null":
		return value == nil
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		f, ok := value.(float64)
		return ok && f == float64(int64(f))
	case "number":
		_, ok := value.(float64)
		return ok
	default:
		return true
	}
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func firstNonEmptyINI(section string, keys ...string) string {
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

func resolveBaseURL(candidates ...string) (string, error) {
	var lastErr error

	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		normalized, err := normalizeBaseURL(trimmed)
		if err == nil {
			return normalized, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("APISIX Admin API base URL is required")
}
