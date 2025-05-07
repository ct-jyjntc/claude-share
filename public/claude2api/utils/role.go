package utils

import (
	"claude2api/config"
)

// **获取角色前缀**
func GetRolePrefix(role string) string {
	if config.ConfigInstance.NoRolePrefix {
		return ""
	}
	switch role {
	case "system":
		return "System: "
	case "user":
		return "Human: "
	case "assistant":
		return "Assistant: "
	default:
		return "Unknown: "
	}
}
