package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"io"
	"net/http"
	"strings"
)

// 创建AI配置Tab
func createAIConfigTab(w fyne.Window) fyne.CanvasObject {
	gc := widget.NewRadioGroup([]string{"Ollama API"}, func(selected string) {
		config.CodeGPT.Provider = selected
		saveConfig()
	})
	gc.SetSelected(config.CodeGPT.Provider)

	apiKey := widget.NewEntry()
	apiKey.SetText(config.CodeGPT.APIKey)
	apiKey.OnChanged = func(value string) {
		config.CodeGPT.APIKey = value
		saveConfig()
	}

	baseURL := widget.NewEntry()
	baseURL.SetText(config.CodeGPT.BaseURL)
	baseURL.OnChanged = func(value string) {
		config.CodeGPT.BaseURL = value
		saveConfig()
	}

	modelSelect := widget.NewSelect([]string{config.CodeGPT.Model}, func(value string) {
		config.CodeGPT.Model = value
		saveConfig()
	})
	modelSelect.SetSelected(config.CodeGPT.Model)

	refreshButton := widget.NewButton("刷新", func() {
		if config.CodeGPT.Provider == "Ollama API" && config.CodeGPT.BaseURL != "" {
			models, err := fetchOllamaModels(config.CodeGPT.BaseURL)
			if err != nil {
				dialog.ShowError(fmt.Errorf("获取模型列表失败: %v", err), w)
				return
			}
			modelSelect.Options = models
			modelSelect.Refresh()
		} else {
			dialog.ShowError(fmt.Errorf("请先配置Ollama API的基础URL"), w)
		}
	})

	promptTemplate := widget.NewMultiLineEntry()
	promptTemplate.Scroll = 2
	promptTemplate.Wrapping = fyne.TextWrapWord
	promptTemplate.SetText(config.PromptTemplate)
	promptTemplate.OnChanged = func(value string) {
		config.PromptTemplate = value
		saveConfig()
	}

	chatPromptTemplate := widget.NewMultiLineEntry()
	chatPromptTemplate.Scroll = 2
	chatPromptTemplate.Wrapping = fyne.TextWrapWord
	chatPromptTemplate.SetText(config.ChatPromptTemplate)
	chatPromptTemplate.OnChanged = func(value string) {
		config.ChatPromptTemplate = value
		saveConfig()
	}

	return container.NewVBox(
		widget.NewLabel("AI配置"),
		widget.NewForm(
			widget.NewFormItem("API提供者:", gc),
			widget.NewFormItem("API密钥:", apiKey),
			widget.NewFormItem("基础URL:", baseURL),
			widget.NewFormItem("模型:", container.NewBorder(nil, nil, nil, refreshButton, modelSelect)),
			widget.NewFormItem("评审提示词:", promptTemplate),
			widget.NewFormItem("聊天提示词:", chatPromptTemplate),
		),
	)
}

// 新增：从Ollama插件获取模型列表
func fetchOllamaModels(baseURL string) ([]string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	models := []string{}
	for _, item := range response["models"].([]interface{}) {
		model := item.(map[string]interface{})
		models = append(models, model["name"].(string))
	}

	return models, nil
}

func sendToOllamaByChat(diff string, key string, url string, model string, callback func(string, bool)) error {
	return sendToOllamaByChatByPrompt(diff, key, url, model, config.PromptTemplate, callback)
}
func sendToOllamaByChatByPrompt(diff string, key string, url string, model, prompt string, callback func(string, bool), history ...map[string]interface{}) error {
	client := &http.Client{}
	reqBody := map[string]interface{}{
		"stream": true, // 启用流式响应
		"model":  model,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": prompt,
			},
			{
				"role":    "user",
				"content": diff,
			},
		},
	}
	if history != nil && len(history) > 0 {
		reqBody["messages"] = append(reqBody["messages"].([]map[string]interface{}), history[0])
	}
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	fullUrl := strings.ReplaceAll(url+"/api/chat", "//api/chat", "/api/chat")
	fmt.Println("  request:", key, "  url:", fullUrl, "  body:", string(reqBodyBytes))
	req, err := http.NewRequest("POST", fullUrl, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var fullResponse strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("  response:", line)
		// {"model":"qwen2.5:14b","created_at":"2025-03-04T06:50:50.732967Z","message":{"role":"assistant","content":"?"},"done":false}
		// {"model":"qwen2.5:14b","created_at":"2025-03-04T06:50:50.7464458Z","message":{"role":"assistant","content":""},"done_reason":"stop","done":true,

		if strings.HasPrefix(line, "{\"") {
			jsonStr := line // 去掉"data: "前缀
			var rsp map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &rsp); err != nil {
				continue
			}
			if message, ok := rsp["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					fullResponse.WriteString(content)
					callback(content, false) // 发送部分响应
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	finalResponse := fullResponse.String()
	// 去掉think
	//if strings.HasPrefix(finalResponse, "<think>") && strings.Contains(finalResponse, "</think>") {
	//	finalResponse = finalResponse[strings.Index(finalResponse, "</think>")+len("</think>"):]
	//}
	callback(finalResponse, true) // 发送最终完整响应并标记完成
	return nil
}
