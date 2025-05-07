package service

import (
	"claude2api/config"
	"claude2api/core"
	"claude2api/logger"
	"claude2api/model"
	"claude2api/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthCheckHandler handles the health check endpoint
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// ChatCompletionsHandler handles the chat completions endpoint
func ChatCompletionsHandler(c *gin.Context) {
	useMirror, exist := c.Get("UseMirrorApi")
	if exist && useMirror.(bool) {
		MirrorChatHandler(c)
		return
	}

	// Parse and validate request
	req, err := parseAndValidateRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Process messages into prompt and extract images
	processor := utils.NewChatRequestProcessor()
	processor.ProcessMessages(req.Messages)

	// Get model or use default
	model := getModelOrDefault(req.Model)
	
	// 重置重试计数器，准备开始新的请求
	config.Sr.ResetRetryCount()
	
	// 轮询尝试所有会话，直到成功或达到最大重试次数
	for {
		// 获取下一个会话，带重试计数
		session, err := config.Sr.GetNextSessionWithRetry()
		if err != nil {
			// 如果所有重试都失败，返回错误
			logger.Error(fmt.Sprintf("Failed to get session after maximum retries: %v", err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to process request after multiple attempts",
			})
			return
		}

		// 记录当前使用的会话信息
		logger.Info(fmt.Sprintf("Using session for model %s: %s (Attempt %d/%d)", 
			model, 
			session.SessionKey,
			config.Sr.RetryCount,
			config.ConfigInstance.RetryCount))

		// 如果不是第一次尝试，重置提示内容
		if config.Sr.RetryCount > 1 {
			processor.Prompt.Reset()
			processor.Prompt.WriteString(processor.RootPrompt.String())
		}

		// 处理请求
		if handleChatRequest(c, session, model, processor, req.Stream) {
			// 成功处理，重置重试计数器
			config.Sr.ResetRetryCount()
			return
		}

		// 如果请求失败，记录日志并继续尝试下一个会话
		logger.Info(fmt.Sprintf("Session %s failed, trying next session", session.SessionKey))
	}
}

func MoudlesHandler(c *gin.Context) {
	models := []map[string]interface{}{
		{"id": "claude-3-7-sonnet-20250219"},
		{"id": "claude-3-7-sonnet-20250219-think"},
	}
	c.JSON(http.StatusOK, gin.H{
		"data": models,
	})
}

func MirrorChatHandler(c *gin.Context) {
	if !config.ConfigInstance.EnableMirrorApi {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Mirror API is not enabled",
		})
		return
	}

	// Parse and validate request
	req, err := parseAndValidateRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Process messages into prompt and extract images
	processor := utils.NewChatRequestProcessor()
	processor.ProcessMessages(req.Messages)

	// Get model or use default
	model := getModelOrDefault(req.Model)

	// Extract session info from auth header
	session, err := extractSessionFromAuthHeader(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("Invalid authorization: %v", err),
		})
		return
	}

	// Process the request with the provided session
	if !handleChatRequest(c, session, model, processor, req.Stream) {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to process request",
		})
		return
	}
}

// Helper functions

func parseAndValidateRequest(c *gin.Context) (*model.ChatCompletionRequest, error) {
	var req model.ChatCompletionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("Invalid request: %v", err),
		})
		return nil, err
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "No messages provided",
		})
		return nil, fmt.Errorf("no messages provided")
	}

	return &req, nil
}

func getModelOrDefault(model string) string {
	if model == "" {
		return "claude-3-7-sonnet-20250219"
	}
	return model
}

func extractSessionFromAuthHeader(c *gin.Context) (config.SessionInfo, error) {
	authInfo := c.Request.Header.Get("Authorization")
	authInfo = strings.TrimPrefix(authInfo, "Bearer ")

	if authInfo == "" {
		return config.SessionInfo{SessionKey: "", OrgID: ""}, fmt.Errorf("missing authorization header")
	}

	if strings.Contains(authInfo, ":") {
		parts := strings.Split(authInfo, ":")
		return config.SessionInfo{SessionKey: parts[0], OrgID: parts[1]}, nil
	}

	return config.SessionInfo{SessionKey: authInfo, OrgID: ""}, nil
}

func handleChatRequest(c *gin.Context, session config.SessionInfo, model string, processor *utils.ChatRequestProcessor, stream bool) bool {
	// Initialize the Claude client
	claudeClient := core.NewClient(session.SessionKey, config.ConfigInstance.Proxy)

	// Get org ID if not already set
	if session.OrgID == "" {
		orgId, err := claudeClient.GetOrgID()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get org ID: %v", err))
			return false
		}
		session.OrgID = orgId
		config.ConfigInstance.SetSessionOrgID(session.SessionKey, session.OrgID)
	}

	claudeClient.SetOrgID(session.OrgID)

	// Upload images if any
	if len(processor.ImgDataList) > 0 {
		err := claudeClient.UploadFile(processor.ImgDataList)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to upload file: %v", err))
			return false
		}
	}

	// Handle large context if needed
	if processor.Prompt.Len() > config.ConfigInstance.MaxChatHistoryLength {
		claudeClient.SetBigContext(processor.Prompt.String())
		processor.ResetForBigContext()
		logger.Info(fmt.Sprintf("Prompt length exceeds max limit (%d), using file context", config.ConfigInstance.MaxChatHistoryLength))
	}

	// Create conversation
	conversationID, err := claudeClient.CreateConversation(model)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create conversation: %v", err))
		return false
	}

	// Send message
	if _, err := claudeClient.SendMessage(conversationID, processor.Prompt.String(), stream, c); err != nil {
		logger.Error(fmt.Sprintf("Failed to send message: %v", err))
		go cleanupConversation(claudeClient, conversationID, 3)
		return false
	}

	// Clean up conversation if enabled
	if config.ConfigInstance.ChatDelete {
		go cleanupConversation(claudeClient, conversationID, 3)
	}

	return true
}

func cleanupConversation(client *core.Client, conversationID string, retry int) {
	for i := 0; i < retry; i++ {
		if err := client.DeleteConversation(conversationID); err != nil {
			logger.Error(fmt.Sprintf("Failed to delete conversation: %v", err))
			time.Sleep(2 * time.Second)
			continue
		}
		logger.Info(fmt.Sprintf("Successfully deleted conversation: %s", conversationID))
		return // 成功后直接返回，不执行后面的错误日志
	}
	// 只有当所有重试都失败后，才会执行到这里
	logger.Error(fmt.Sprintf("Cleanup %s conversation %s failed after %d retries", client.SessionKey, conversationID, retry))
}
