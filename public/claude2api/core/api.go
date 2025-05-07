package core

import (
	"bufio"
	"claude2api/logger"
	"claude2api/model"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/imroc/req/v3"
)

type Client struct {
	SessionKey   string
	orgID        string
	client       *req.Client
	defaultAttrs map[string]interface{}
}

type ResponseEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		THINKING string `json:"thinking"`
	} `json:"delta"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewClient(sessionKey string, proxy string) *Client {
	client := req.C().ImpersonateChrome().SetTimeout(time.Minute * 5)
	client.Transport.SetResponseHeaderTimeout(time.Second * 10)
	if proxy != "" {
		client.SetProxyURL(proxy)
	}
	// Set common headers
	headers := map[string]string{
		"accept":                    "text/event-stream, text/event-stream",
		"accept-language":           "zh-CN,zh;q=0.9",
		"anthropic-client-platform": "web_claude_ai",
		"content-type":              "application/json",
		"origin":                    "https://claude.ai",
		"priority":                  "u=1, i",
	}
	for key, value := range headers {
		client.SetCommonHeader(key, value)
	}
	// Set cookies
	client.SetCommonCookies(&http.Cookie{
		Name:  "sessionKey",
		Value: sessionKey,
	})
	// Create default client with session key
	c := &Client{
		SessionKey: sessionKey,
		client:     client,
		defaultAttrs: map[string]interface{}{
			"personalized_styles": []map[string]interface{}{
				{
					"type":       "default",
					"key":        "Default",
					"name":       "Normal",
					"nameKey":    "normal_style_name",
					"prompt":     "Normal",
					"summary":    "Default responses from Claude",
					"summaryKey": "normal_style_summary",
					"isDefault":  true,
				},
			},
			"tools": []map[string]interface{}{
				{
					"type": "web_search_v0",
					"name": "web_search",
				},
			},
			"parent_message_uuid": "00000000-0000-4000-8000-000000000000",
			"attachments":         []interface{}{},
			"files":               []interface{}{},
			"sync_sources":        []interface{}{},
			"rendering_mode":      "messages",
			"timezone":            "America/New_York",
		},
	}
	return c
}

// SetOrgID sets the organization ID for the client
func (c *Client) SetOrgID(orgID string) {
	c.orgID = orgID
}
func (c *Client) GetOrgID() (string, error) {
	url := "https://claude.ai/api/organizations"
	resp, err := c.client.R().
		SetHeader("referer", "https://claude.ai/new").
		Get(url)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	type OrgResponse []struct {
		ID            int    `json:"id"`
		UUID          string `json:"uuid"`
		Name          string `json:"name"`
		RateLimitTier string `json:"rate_limit_tier"`
	}

	var orgs OrgResponse
	if err := json.Unmarshal(resp.Bytes(), &orgs); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if len(orgs) == 0 {
		return "", errors.New("no organizations found")
	}
	if len(orgs) == 1 {
		return orgs[0].UUID, nil
	}
	for _, org := range orgs {
		if org.RateLimitTier == "default_claude_ai" {
			return org.UUID, nil
		}
	}
	return "", errors.New("no default organization found")

}

// CreateConversation creates a new conversation and returns its UUID
func (c *Client) CreateConversation(model string) (string, error) {
	if c.orgID == "" {
		return "", errors.New("organization ID not set")
	}
	url := fmt.Sprintf("https://claude.ai/api/organizations/%s/chat_conversations", c.orgID)
	// 如果以-think结尾
	requestBody := map[string]interface{}{
		"model":                            model,
		"uuid":                             uuid.New().String(),
		"name":                             "",
		"include_conversation_preferences": true,
	}
	if len(model) > 6 && model[len(model)-6:] == "-think" {
		requestBody["paprika_mode"] = "extended"
		requestBody["model"] = model[:len(model)-6]
	}
	resp, err := c.client.R().
		SetHeader("referer", "https://claude.ai/new").
		SetBody(requestBody).
		Post(url)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var result map[string]interface{}
	// logger.Info(fmt.Sprintf("create conversation response: %s", resp.String()))
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	uuid, ok := result["uuid"].(string)
	if !ok {
		return "", errors.New("conversation UUID not found in response")
	}
	return uuid, nil
}

// SendMessage sends a message to a conversation and returns the status and response
func (c *Client) SendMessage(conversationID string, message string, stream bool, gc *gin.Context) (int, error) {
	if c.orgID == "" {
		return 500, errors.New("organization ID not set")
	}
	url := fmt.Sprintf("https://claude.ai/api/organizations/%s/chat_conversations/%s/completion",
		c.orgID, conversationID)
	// Create request body with default attributes
	requestBody := c.defaultAttrs
	requestBody["prompt"] = message
	// Set up streaming response
	resp, err := c.client.R().DisableAutoReadResponse().
		SetHeader("referer", fmt.Sprintf("https://claude.ai/chat/%s", conversationID)).
		SetHeader("accept", "text/event-stream, text/event-stream").
		SetHeader("anthropic-client-platform", "web_claude_ai").
		SetHeader("cache-control", "no-cache").
		SetBody(requestBody).
		Post(url)
	if err != nil {
		return 500, fmt.Errorf("request failed: %w", err)
	}
	logger.Info(fmt.Sprintf("Claude response status code: %d", resp.StatusCode))
	if resp.StatusCode == http.StatusTooManyRequests {
		return http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return 200, c.HandleResponse(resp.Body, stream, gc)
}

// HandleResponse converts Claude's SSE format to OpenAI format and writes to the response writer
func (c *Client) HandleResponse(body io.ReadCloser, stream bool, gc *gin.Context) error {
	defer body.Close()
	// Set headers for streaming
	if stream {
		gc.Writer.Header().Set("Content-Type", "text/event-stream")
		gc.Writer.Header().Set("Cache-Control", "no-cache")
		gc.Writer.Header().Set("Connection", "keep-alive")
		// 发送200状态码
		gc.Writer.WriteHeader(http.StatusOK)
		gc.Writer.Flush()
	}
	scanner := bufio.NewScanner(body)
	clientDone := gc.Request.Context().Done()
	// Keep track of the full response for the final message
	thinkingShown := false
	res_all_text := ""
	for scanner.Scan() {
		select {
		case <-clientDone:
			// 客户端已断开连接，清理资源并退出
			logger.Info("Client closed connection")
			return nil
		default:
			// 继续处理响应
		}
		line := scanner.Text()
		// logger.Info(fmt.Sprintf("Claude SSE line: %s", line))
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		var event ResponseEvent
		if err := json.Unmarshal([]byte(data), &event); err == nil {
			if event.Type == "error" && event.Error.Message != "" {
				model.ReturnOpenAIResponse(event.Error.Message, stream, gc)
				return nil
			}
			if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
				res_text := event.Delta.Text
				if thinkingShown {
					res_text = "</think>\n" + res_text
					thinkingShown = false
				}
				res_all_text += res_text
				if !stream {
					continue
				}
				model.ReturnOpenAIResponse(res_text, stream, gc)
				continue
			}
			if event.Delta.Type == "thinking_delta" {
				res_text := event.Delta.THINKING
				if !thinkingShown {
					res_text = "<think>" + res_text
					thinkingShown = true
				}
				res_all_text += res_text
				if !stream {
					continue
				}
				model.ReturnOpenAIResponse(res_text, stream, gc)
				continue
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	if !stream {
		model.ReturnOpenAIResponse(res_all_text, stream, gc)
	} else {
		// 发送结束标志
		gc.Writer.Write([]byte("data: [DONE]\n\n"))
		gc.Writer.Flush()
	}

	return nil
}

// DeleteConversation deletes a conversation by ID
func (c *Client) DeleteConversation(conversationID string) error {
	if c.orgID == "" {
		return errors.New("organization ID not set")
	}
	url := fmt.Sprintf("https://claude.ai/api/organizations/%s/chat_conversations/%s",
		c.orgID, conversationID)
	requestBody := map[string]string{
		"uuid": conversationID,
	}
	resp, err := c.client.R().
		SetHeader("referer", fmt.Sprintf("https://claude.ai/chat/%s", conversationID)).
		SetBody(requestBody).
		Delete(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// UploadFile uploads files to Claude and adds them to the client's default attributes
// fileData should be in the format: data:image/jpeg;base64,/9j/4AA...
func (c *Client) UploadFile(fileData []string) error {
	if c.orgID == "" {
		return errors.New("organization ID not set")
	}
	if len(fileData) == 0 {
		return errors.New("empty file data")
	}

	// Initialize files array in default attributes if it doesn't exist
	if _, ok := c.defaultAttrs["files"]; !ok {
		c.defaultAttrs["files"] = []interface{}{}
	}

	// Process each file
	for _, fd := range fileData {
		if fd == "" {
			continue // Skip empty entries
		}

		// Parse the base64 data
		parts := strings.SplitN(fd, ",", 2)
		if len(parts) != 2 {
			return errors.New("invalid file data format")
		}

		// Get the content type from the data URI
		metaParts := strings.SplitN(parts[0], ":", 2)
		if len(metaParts) != 2 {
			return errors.New("invalid content type in file data")
		}

		metaInfo := strings.SplitN(metaParts[1], ";", 2)
		if len(metaInfo) != 2 || metaInfo[1] != "base64" {
			return errors.New("invalid encoding in file data")
		}

		contentType := metaInfo[0]

		// Decode the base64 data
		fileBytes, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return fmt.Errorf("failed to decode base64 data: %w", err)
		}

		// Determine filename based on content type
		var filename string
		switch contentType {
		case "image/jpeg":
			filename = "image.jpg"
		case "image/png":
			filename = "image.png"
		case "application/pdf":
			filename = "document.pdf"
		default:
			filename = "file"
		}

		// Create the upload URL
		url := fmt.Sprintf("https://claude.ai/api/%s/upload", c.orgID)

		// Create a multipart form request
		resp, err := c.client.R().
			SetHeader("referer", "https://claude.ai/new").
			SetHeader("anthropic-client-platform", "web_claude_ai").
			SetFileBytes("file", filename, fileBytes).
			SetContentType("multipart/form-data").
			Post(url)

		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, resp.String())
		}

		// Parse the response
		var result struct {
			FileUUID string `json:"file_uuid"`
		}

		if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if result.FileUUID == "" {
			return errors.New("file UUID not found in response")
		}

		// Add file to default attributes
		c.defaultAttrs["files"] = append(c.defaultAttrs["files"].([]interface{}), result.FileUUID)
	}

	return nil
}

func (c *Client) SetBigContext(context string) {
	c.defaultAttrs["attachments"] = []map[string]interface{}{
		{
			"file_name":         "context.txt",
			"file_type":         "text/plain",
			"file_size":         len(context),
			"extracted_content": context,
		},
	}

}
