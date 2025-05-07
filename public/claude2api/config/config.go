package config

import (
	"claude2api/logger"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type SessionInfo struct {
	SessionKey string `yaml:"sessionKey" json:"key"`
	OrgID      string `yaml:"orgID" json:"orgID,omitempty"`
}

type SessionRagen struct {
	Index      int
	RetryCount int
	Mutex      sync.Mutex
}

type Config struct {
	Sessions               []SessionInfo `yaml:"sessions"`
	Address                string        `yaml:"address"`
	APIKey                 string        `yaml:"apiKey"`
	Proxy                  string        `yaml:"proxy"`
	ChatDelete             bool          `yaml:"chatDelete"`
	MaxChatHistoryLength   int           `yaml:"maxChatHistoryLength"`
	RetryCount             int           `yaml:"retryCount"`
	NoRolePrefix           bool          `yaml:"noRolePrefix"`
	PromptDisableArtifacts bool          `yaml:"promptDisableArtifacts"`
	EnableMirrorApi        bool          `yaml:"enableMirrorApi"`
	MirrorApiPrefix        string        `yaml:"mirrorApiPrefix"`
	RwMutx                 sync.RWMutex  `yaml:"-"` // 不从YAML加载
}

// 解析 SESSION 格式的环境变量
func parseSessionEnv(envValue string) (int, []SessionInfo) {
	if envValue == "" {
		return 0, []SessionInfo{}
	}
	var sessions []SessionInfo
	sessionPairs := strings.Split(envValue, ",")
	retryCount := len(sessionPairs) // 重试次数等于 session 数量
	for _, pair := range sessionPairs {
		if pair == "" {
			retryCount--
			continue
		}
		parts := strings.Split(pair, ":")
		session := SessionInfo{
			SessionKey: parts[0],
		}

		if len(parts) > 1 {
			session.OrgID = parts[1]
		} else if len(parts) == 1 {
			session.OrgID = ""
		}

		sessions = append(sessions, session)
	}
	if retryCount > 5 {
		retryCount = 5 // 限制最大重试次数为 5 次
	}
	return retryCount, sessions
}

// 根据模型选择合适的 session
func (c *Config) GetSessionForModel(idx int) (SessionInfo, error) {
	if len(c.Sessions) == 0 {
		return SessionInfo{}, fmt.Errorf("no available sessions")
	}

	// 确保索引在有效范围内（轮询模式）
	validIdx := idx % len(c.Sessions)

	c.RwMutx.RLock()
	defer c.RwMutx.RUnlock()
	return c.Sessions[validIdx], nil
}

func (c *Config) SetSessionOrgID(sessionKey, orgID string) {
	c.RwMutx.Lock()
	defer c.RwMutx.Unlock()
	for i, session := range c.Sessions {
		if session.SessionKey == sessionKey {
			logger.Info(fmt.Sprintf("Setting OrgID for session %s to %s", sessionKey, orgID))
			c.Sessions[i].OrgID = orgID
			return
		}
	}
}
func (sr *SessionRagen) NextIndex() int {
	sr.Mutex.Lock()
	defer sr.Mutex.Unlock()

	index := sr.Index
	// 移动到下一个索引（轮询）
	sr.Index = (index + 1) % len(ConfigInstance.Sessions)
	return index
}

// 获取下一个会话，并处理重试逻辑
func (sr *SessionRagen) GetNextSessionWithRetry() (SessionInfo, error) {
	sr.Mutex.Lock()
	defer sr.Mutex.Unlock()

	// 检查是否已达到最大重试次数
	if sr.RetryCount >= ConfigInstance.RetryCount {
		return SessionInfo{}, fmt.Errorf("exceeded maximum retry count (%d)", ConfigInstance.RetryCount)
	}

	// 获取当前索引
	index := sr.Index
	// 移动到下一个索引（轮询）
	sr.Index = (index + 1) % len(ConfigInstance.Sessions)
	// 增加重试计数
	sr.RetryCount++

	// 如果已经尝试了所有会话一轮，记录日志
	if sr.Index == 0 {
		logger.Info("Completed one full rotation of all session keys, starting again from the beginning")
	}

	// 获取会话
	return ConfigInstance.GetSessionForModel(index)
}

// 重置重试计数器
func (sr *SessionRagen) ResetRetryCount() {
	sr.Mutex.Lock()
	defer sr.Mutex.Unlock()
	sr.RetryCount = 0
}

// 检查配置文件是否存在
func configFileExists() (bool, string) {
	execDir := filepath.Dir(os.Args[0])
	workDir, _ := os.Getwd()
	if execDir == "" && workDir == "" {
		logger.Error("Failed to get executable directory")
		return false, ""
	}

	var err error
	exeConfigPath := filepath.Join(execDir, "config.yaml")
	_, err = os.Stat(exeConfigPath)
	if !os.IsNotExist(err) {
		return true, exeConfigPath
	}

	workConfigPath := filepath.Join(workDir, "config.yaml")
	_, err = os.Stat(workConfigPath)
	if !os.IsNotExist(err) {
		return true, workConfigPath
	}

	return false, ""
}

// 从YAML文件加载配置
func loadConfigFromYAML(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// 设置读写锁（不从YAML加载）
	config.RwMutx = sync.RWMutex{}

	// 如果地址为空，使用默认值
	if config.Address == "" {
		config.Address = "0.0.0.0:8080"
	}

	return &config, nil
}

// 找到sessionKeys.json文件的路径
func findSessionKeysFile() (string, error) {
	// 获取可执行文件所在目录
	execDir := filepath.Dir(os.Args[0])
	// 获取当前工作目录
	workDir, _ := os.Getwd()

	// 尝试在可执行文件同级目录的data文件夹中查找
	jsonPath := filepath.Join(execDir, "data", "sessionKeys.json")
	_, err := os.Stat(jsonPath)
	if !os.IsNotExist(err) {
		return jsonPath, nil
	}

	// 尝试在当前工作目录的data文件夹中查找
	jsonPath = filepath.Join(workDir, "data", "sessionKeys.json")
	_, err = os.Stat(jsonPath)
	if !os.IsNotExist(err) {
		return jsonPath, nil
	}

	// 尝试在/app/data目录中查找（Docker容器中的路径）
	jsonPath = "/app/data/sessionKeys.json"
	_, err = os.Stat(jsonPath)
	if !os.IsNotExist(err) {
		return jsonPath, nil
	}

	return "", fmt.Errorf("sessionKeys.json not found in data directory")
}

// 从JSON文件加载session keys
func loadSessionKeysFromJSON() ([]SessionInfo, error) {
	// 如果路径为空，先查找文件
	if sessionKeysFilePath == "" {
		path, err := findSessionKeysFile()
		if err != nil {
			return nil, err
		}
		sessionKeysFilePath = path
	}

	// 读取JSON文件
	data, err := os.ReadFile(sessionKeysFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessionKeys.json: %v", err)
	}

	// 解析JSON
	type SessionKeyEntry struct {
		ID  int    `json:"id"`
		Key string `json:"key"`
	}

	type SessionKeysFile struct {
		SessionKeys []SessionKeyEntry `json:"sessionKeys"`
	}

	var sessionKeysFile SessionKeysFile
	err = json.Unmarshal(data, &sessionKeysFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sessionKeys.json: %v", err)
	}

	// 转换为SessionInfo格式
	var sessions []SessionInfo
	for _, entry := range sessionKeysFile.SessionKeys {
		sessions = append(sessions, SessionInfo{
			SessionKey: entry.Key,
			OrgID:      "", // 默认为空
		})
	}

	return sessions, nil
}

// 设置文件监听器来监控sessionKeys.json文件的变化
func setupFileWatcher() error {
	// 初始化监听器
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %v", err)
	}

	// 确保我们有文件路径
	if sessionKeysFilePath == "" {
		path, err := findSessionKeysFile()
		if err != nil {
			return err
		}
		sessionKeysFilePath = path
	}

	// 添加文件路径到监听器
	err = watcher.Add(sessionKeysFilePath)
	if err != nil {
		return fmt.Errorf("failed to add file to watcher: %v", err)
	}

	// 添加文件所在目录到监听器，以处理文件重命名的情况
	dirPath := filepath.Dir(sessionKeysFilePath)
	err = watcher.Add(dirPath)
	if err != nil {
		return fmt.Errorf("failed to add directory to watcher: %v", err)
	}

	logger.Info(fmt.Sprintf("File watcher set up for: %s", sessionKeysFilePath))

	// 启动监听器协程
	go watchFileChanges()

	return nil
}

// 监听文件变化并实时重新加载会话密钥
func watchFileChanges() {
	defer watcher.Close()

	// 防止短时间内多次触发的变量
	var debounceTimer *time.Timer
	var debounceTimeout = 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// 如果是我们关注的文件
			if event.Name == sessionKeysFilePath || filepath.Base(event.Name) == "sessionKeys.json" {
				// 如果文件被写入、创建或重命名
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
					// 使用防抖动计时器来避免多次重新加载
					if debounceTimer != nil {
						debounceTimer.Stop()
					}

					debounceTimer = time.AfterFunc(debounceTimeout, func() {
						// 重新加载会话密钥
						reloadSessionKeys()
					})
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error(fmt.Sprintf("File watcher error: %v", err))
		}
	}
}

// 重新加载会话密钥
func reloadSessionKeys() {
	logger.Info("Reloading session keys from JSON file...")

	// 加载新的会话密钥
	sessions, err := loadSessionKeysFromJSON()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to reload session keys: %v", err))
		return
	}

	// 更新配置
	ConfigInstance.RwMutx.Lock()
	oldSessionCount := len(ConfigInstance.Sessions)
	ConfigInstance.Sessions = sessions
	ConfigInstance.RwMutx.Unlock()

	// 重置会话轮询器
	Sr.Mutex.Lock()
	Sr.Index = 0
	Sr.RetryCount = 0
	Sr.Mutex.Unlock()

	logger.Info(fmt.Sprintf("Successfully reloaded %d session keys (previous count: %d)", len(sessions), oldSessionCount))

	// 打印新加载的会话密钥信息（带掩码）
	for i, session := range sessions {
		// 只显示密钥的前10个和后10个字符，中间用***替代
		sessionKeyLength := len(session.SessionKey)
		maskedKey := session.SessionKey
		if sessionKeyLength > 20 {
			maskedKey = session.SessionKey[:10] + "***" + session.SessionKey[sessionKeyLength-10:]
		}
		logger.Info(fmt.Sprintf("Session %d: %s", i+1, maskedKey))
	}
}

// 从环境变量加载配置
func loadConfigFromEnv() *Config {
	maxChatHistoryLength, err := strconv.Atoi(os.Getenv("MAX_CHAT_HISTORY_LENGTH"))
	if err != nil {
		maxChatHistoryLength = 10000 // 默认值
	}

	// 获取SESSIONS环境变量
	sessionsEnv := os.Getenv("SESSIONS")
	var retryCount int
	var sessions []SessionInfo

	// 如果SESSIONS环境变量为空，尝试从JSON文件加载
	if sessionsEnv == "" {
		logger.Info("SESSIONS environment variable is empty, trying to load from JSON file")
		jsonSessions, err := loadSessionKeysFromJSON()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load session keys from JSON: %v", err))
			// 如果JSON加载失败，使用空会话列表
			retryCount = 0
			sessions = []SessionInfo{}
		} else {
			logger.Info(fmt.Sprintf("Successfully loaded %d session keys from JSON file", len(jsonSessions)))
			sessions = jsonSessions
			retryCount = len(sessions)
			if retryCount > 5 {
				retryCount = 5 // 限制最大重试次数为 5 次
			}
		}
	} else {
		// 否则从环境变量解析
		retryCount, sessions = parseSessionEnv(sessionsEnv)
	}

	config := &Config{
		// 设置会话信息
		Sessions: sessions,
		// 设置服务地址，默认为 "0.0.0.0:8080"
		Address: os.Getenv("ADDRESS"),

		// 设置 API 认证密钥
		APIKey: os.Getenv("APIKEY"),
		// 设置代理地址
		Proxy: os.Getenv("PROXY"),
		// 自动删除聊天
		ChatDelete: os.Getenv("CHAT_DELETE") != "false",
		// 设置最大聊天历史长度
		MaxChatHistoryLength: maxChatHistoryLength,
		// 设置重试次数
		RetryCount: retryCount,
		// 设置是否使用角色前缀
		NoRolePrefix: os.Getenv("NO_ROLE_PREFIX") == "true",
		// 设置是否使用提示词禁用artifacts
		PromptDisableArtifacts: os.Getenv("PROMPT_DISABLE_ARTIFACTS") == "true",
		// 设置是否启用镜像API
		EnableMirrorApi: os.Getenv("ENABLE_MIRROR_API") == "true",
		// 设置镜像API前缀
		MirrorApiPrefix: os.Getenv("MIRROR_API_PREFIX"),
		// 设置读写锁
		RwMutx: sync.RWMutex{},
	}

	// 如果地址为空，使用默认值
	if config.Address == "" {
		config.Address = "0.0.0.0:8080"
	}
	return config
}

// 加载配置
func LoadConfig() *Config {
	// 检查配置文件是否存在
	exists, configPath := configFileExists()
	if exists {
		logger.Info(fmt.Sprintf("Found config file at %s", configPath))
		config, err := loadConfigFromYAML(configPath)
		if err == nil {
			logger.Info("Successfully loaded configuration from YAML file")
			return config
		}
		logger.Error(fmt.Sprintf("Failed to load config from YAML: %v, falling back to environment variables", err))
	}

	// 如果配置文件不存在或加载失败，从环境变量加载
	logger.Info("Loading configuration from environment variables")
	return loadConfigFromEnv()
}

var ConfigInstance *Config
var Sr *SessionRagen
var watcher *fsnotify.Watcher
var sessionKeysFilePath string

func init() {
	rand.Seed(time.Now().UnixNano())
	// 加载环境变量
	_ = godotenv.Load()
	Sr = &SessionRagen{
		Index:      0,
		RetryCount: 0,
		Mutex:      sync.Mutex{},
	}
	ConfigInstance = LoadConfig()

	// 设置文件监听器来实时监控sessionKeys.json文件的变化
	if len(ConfigInstance.Sessions) > 0 && os.Getenv("SESSIONS") == "" {
		// 如果使用的是JSON文件中的会话密钥，则设置监听器
		err := setupFileWatcher()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to set up file watcher: %v", err))
		} else {
			logger.Info("File watcher for sessionKeys.json has been set up successfully")
		}
	}

	logger.Info("Loaded config:")
	logger.Info(fmt.Sprintf("Max Retry count: %d", ConfigInstance.RetryCount))
	logger.Info(fmt.Sprintf("Total Session Keys: %d", len(ConfigInstance.Sessions)))
	for i, session := range ConfigInstance.Sessions {
		// 只显示密钥的前10个和后10个字符，中间用***替代
		sessionKeyLength := len(session.SessionKey)
		maskedKey := session.SessionKey
		if sessionKeyLength > 20 {
			maskedKey = session.SessionKey[:10] + "***" + session.SessionKey[sessionKeyLength-10:]
		}
		logger.Info(fmt.Sprintf("Session %d: %s, OrgID: %s", i+1, maskedKey, session.OrgID))
	}
	logger.Info(fmt.Sprintf("Address: %s", ConfigInstance.Address))
	logger.Info(fmt.Sprintf("APIKey: %s", ConfigInstance.APIKey))
	logger.Info(fmt.Sprintf("Proxy: %s", ConfigInstance.Proxy))
	logger.Info(fmt.Sprintf("ChatDelete: %t", ConfigInstance.ChatDelete))
	logger.Info(fmt.Sprintf("MaxChatHistoryLength: %d", ConfigInstance.MaxChatHistoryLength))
	logger.Info(fmt.Sprintf("NoRolePrefix: %t", ConfigInstance.NoRolePrefix))
	logger.Info(fmt.Sprintf("PromptDisableArtifacts: %t", ConfigInstance.PromptDisableArtifacts))
	logger.Info(fmt.Sprintf("EnableMirrorApi: %t", ConfigInstance.EnableMirrorApi))
	logger.Info(fmt.Sprintf("MirrorApiPrefix: %s", ConfigInstance.MirrorApiPrefix))
}
