package main

import (
	"encoding/json"
	"io"
	"os"
	"os/user"
	"path/filepath"
)

// 配置结构体
type Config struct {
	CodeGPT            CodeGPTConfig `json:"codegpt"`
	Redmine            RedmineConfig `json:"redmine"`
	Projects           []Project     `json:"projects"`
	PromptTemplate     string        `json:"prompt_template"`      // 新增提示词模板字段
	ChatPromptTemplate string        `json:"chat_prompt_template"` // 新增提示词模板字段
}

type CodeGPTConfig struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	BaseURL     string `json:"base_url"`
	Model       string `json:"model"`
	DiffUnified int    `json:"diff_unified"`
	ExcludeList string `json:"exclude_list"`
}

type RedmineConfig struct {
	URL              string `json:"url"`
	APIKey           string `json:"api_key"`
	ProjectID        string `json:"project_id"`
	AssignedToUserID string `json:"assigned_to_user_id"`
}

type Project struct {
	Path                    string `json:"path"`
	Name                    string `json:"name"`
	RedmineProjectId        string `json:"redmine_project_id"`
	RedmineAssignedToUserId string `json:"redmine_assigned_to_user_id"`
}

// 配置文件路径
const configFileName = ".codegpt_ui.json"

var config = Config{
	CodeGPT: CodeGPTConfig{
		Provider: "Ollama API",
	},
	Projects: []Project{},
	Redmine: RedmineConfig{
		URL: "http://192.168.2.100:11434",
	},
}

// 加载配置文件
func loadConfig() {
	usr, _ := user.Current()
	configFilePath := filepath.Join(usr.HomeDir, configFileName)

	file, err := os.Open(configFilePath)
	if err == nil {
		defer file.Close()

		data, _ := io.ReadAll(file)
		json.Unmarshal(data, &config)
	} else {
		// 初始化默认配置
		config.PromptTemplate =
			"你目前做为java、vue、typescript 等高级,最强代码审核人员,对以下的修改做审核,如果代码存在严重问题/BUG/异常,请输出问题以及改进建议,如果没有,请输出:代码优秀!" // 设置默认提示词模板
		saveConfig()
	}
}

// 保存配置文件
func saveConfig() {
	usr, _ := user.Current()
	configFilePath := filepath.Join(usr.HomeDir, configFileName)

	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configFilePath, data, 0644)
}
