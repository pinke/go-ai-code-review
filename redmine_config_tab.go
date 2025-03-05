package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 创建Redmine配置Tab
func createRedmineConfigTab(w fyne.Window) fyne.CanvasObject {
	redmineURL := widget.NewEntry()
	redmineURL.SetText(config.Redmine.URL)
	redmineURL.OnChanged = func(value string) {
		config.Redmine.URL = value
		saveConfig()
	}

	redmineAPIKey := widget.NewEntry()
	redmineAPIKey.SetText(config.Redmine.APIKey)
	redmineAPIKey.OnChanged = func(value string) {
		config.Redmine.APIKey = value
		saveConfig()
	}

	redmineProjectID := widget.NewSelect([]string{config.Redmine.ProjectID}, func(value string) {
		config.Redmine.ProjectID = value
		saveConfig()
	})
	redmineProjectID.SetSelected(config.Redmine.ProjectID)

	redmineAssignedToUser := widget.NewSelect([]string{}, func(value string) {
		config.Redmine.AssignedToUserID = value
		saveConfig()
	})
	redmineAssignedToUser.SetSelected(config.Redmine.AssignedToUserID)

	refreshUserButton := widget.NewButton("刷新", func() {
		if config.Redmine.ProjectID == "" {
			dialog.ShowError(fmt.Errorf("请先选择默认项目"), w)
			return
		}
		projectIDStr := strings.Split(config.Redmine.ProjectID, "|")[0]
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil {
			dialog.ShowError(fmt.Errorf("无效的项目ID: %v", err), w)
			return
		}
		users, err := getRedmineProjectUsers(projectID)
		if err != nil {
			dialog.ShowError(fmt.Errorf("获取项目用户失败: %v", err), w)
			return
		}
		redmineAssignedToUser.Options = users
		redmineAssignedToUser.Refresh()
	})

	refreshProjectButton := widget.NewButton("刷新", func() {
		client := &http.Client{}
		req, err := http.NewRequest("GET", config.Redmine.URL+"/projects.json", nil)
		if err != nil {
			dialog.ShowError(fmt.Errorf("创建请求失败: %v", err), w)
			return
		}
		req.Header.Set("X-Redmine-API-Key", config.Redmine.APIKey)
		resp, err := client.Do(req)
		if err != nil {
			dialog.ShowError(fmt.Errorf("连接失败: %v", err), w)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			dialog.ShowError(fmt.Errorf("读取响应失败: %v", err), w)
			return
		}

		rsp := map[string]interface{}{}
		if err := json.Unmarshal(body, &rsp); err != nil {
			dialog.ShowError(fmt.Errorf("解析JSON失败: %v", err), w)
			return
		}

		options := []string{}
		for _, item := range rsp["projects"].([]interface{}) {
			project := item.(map[string]interface{})
			options = append(options, fmt.Sprintf("%.0f|%s", project["id"], project["name"]))
		}
		redmineProjectID.Options = options
		redmineProjectID.Refresh()
	})

	redmineProjectID.OnChanged = func(value string) {
		config.Redmine.ProjectID = value
		saveConfig()
		if value != "" {
			refreshUserButton.OnTapped()
		}
	}

	return container.NewVBox(
		widget.NewLabel("Redmine配置"),
		widget.NewForm(
			widget.NewFormItem("Redmine URL:", redmineURL),
			widget.NewFormItem("Redmine API密钥:", redmineAPIKey),
			widget.NewFormItem("",
				widget.NewButton("测试连接", func() {
					client := &http.Client{}
					req, err := http.NewRequest("GET", config.Redmine.URL+"/projects.json", nil)
					if err != nil {
						dialog.ShowError(fmt.Errorf("创建请求失败: %v", err), w)
						return
					}
					req.Header.Set("X-Redmine-API-Key", config.Redmine.APIKey)
					resp, err := client.Do(req)
					if err != nil {
						dialog.ShowError(fmt.Errorf("连接失败: %v", err), w)
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						dialog.ShowError(fmt.Errorf("连接失败，状态码: %d", resp.StatusCode), w)
						return
					}
					responseBody, _ := io.ReadAll(resp.Body)
					if strings.Contains(string(responseBody), "error") {
						dialog.ShowError(fmt.Errorf("连接失败，响应内容: %s", string(responseBody)), w)
						return
					}
					r := map[string]interface{}{}
					_ = json.Unmarshal(responseBody, &r)

					options := []string{}
					for _, value := range r["projects"].([]interface{}) {
						item := value.(map[string]interface{})
						options = append(options, fmt.Sprintf("%.0f|%s", item["id"], item["name"]))
					}
					redmineProjectID.SetOptions(options)

					dialog.ShowInformation("连接成功", "成功连接到Redmine服务器", w)
				})),
			widget.NewFormItem("", widget.NewLabel("测试提交报告")),
			widget.NewFormItem("测试Redmine项目ID:", container.NewBorder(nil, nil, nil, refreshProjectButton, redmineProjectID)),
			widget.NewFormItem("测试分配项目用户:", container.NewBorder(nil, nil, nil, refreshUserButton, redmineAssignedToUser)),
			widget.NewFormItem("",
				widget.NewButton("测试报告", func() {
					projectId := strings.Split(config.Redmine.ProjectID, "|")[0]
					assignToUserId, _ := strconv.Atoi(strings.Split(config.Redmine.AssignedToUserID, "|")[0])
					err, responseBody := submitToRedmine("自动化测试报告"+time.Now().Format("2006-01-02"),
						projectId, assignToUserId,
						fmt.Sprintf(`测试时间: %s
测试结果: 成功
测试内容: 
1. 连接测试 - 成功
2. 项目列表获取 - 成功`, time.Now().Format("2006-01-02 15:04:05")), "", "", w)
					if err != nil {
						dialog.ShowError(fmt.Errorf("提交失败: %v", err), w)
						return
					}
					dlg := dialog.NewInformation("提交成功", "测试报告已成功提交到Redmine,可到redmine查看,点确定后自动删除.", w)

					resp := string(responseBody)
					issue := map[string]interface{}{}
					_ = json.Unmarshal(responseBody, &issue)
					issueId := issue["issue"].(map[string]interface{})["id"].(float64)
					dlg.SetOnClosed(func() {
						if resp != "" {
							fmt.Println("issueId:", issueId)
							client := &http.Client{}
							urlStr := fmt.Sprintf("%s/issues/%d.json", config.Redmine.URL, int64(issueId))
							req, err := http.NewRequest("DELETE", urlStr, nil)
							if err != nil {
								dialog.ShowError(fmt.Errorf("创建请求失败: %v", err), w)
								return
							}
							req.Header.Set("X-Redmine-API-Key", config.Redmine.APIKey)
							req.Header.Set("Content-Type", "application/json")
							resp, err := client.Do(req)
							if err != nil {
								dialog.ShowError(fmt.Errorf("删除失败: %v", err), w)
								return
							}
							defer resp.Body.Close()
							bd, _ := io.ReadAll(resp.Body)
							if bd != nil {
								fmt.Println("delete response:", resp.StatusCode, string(bd))
							}
							if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
								dialog.ShowError(fmt.Errorf("删除失败，状态码: %d", resp.StatusCode), w)
								return
							}
						}
					})
					dlg.Show()
				}),
			),
		),
	)
}

func submitToRedmine(title, projectId string, assignedToUserId int, ver, diff, aiAdvice string, w fyne.Window) (error, []byte) {
	//去掉deepseek 的思考
	if strings.HasPrefix(aiAdvice, "<think>") && strings.Contains(aiAdvice, "</think>") {
		aiAdvice = aiAdvice[strings.Index(aiAdvice, "</think>")+len("</think>"):]
	}
	fmt.Println("Project id:", projectId, " assignedToUserId:", assignedToUserId)
	report := map[string]interface{}{
		"issue": map[string]interface{}{
			"project_id":     projectId,
			"subject":        title,
			"description":    ver + "\n\n====== AI 审核 ======\n\n" + aiAdvice + "\n\n======= 提交的内容 =======\n\n" + diff,
			"tracker_id":     1, // 1通常表示Bug
			"status_id":      1, // 1通常表示新建
			"priority_id":    7,
			"assigned_to_id": assignedToUserId, // 使用项目配置的分配人ID
		},
	}

	// 将报告转换为JSON
	jsonData, _ := json.Marshal(report)

	// 创建HTTP请求
	client := &http.Client{}
	req, err := http.NewRequest("POST", config.Redmine.URL+"/issues.json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err), nil
	}

	// 设置请求头
	req.Header.Set("X-Redmine-API-Key", config.Redmine.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("提交失败: %v", err), nil
	}
	defer resp.Body.Close()

	// 检查响应状态
	//StatusUnprocessableEntity
	if resp.StatusCode != http.StatusCreated {

		if resp != nil {
			responseBody, _ := io.ReadAll(resp.Body)
			dialog.ShowError(fmt.Errorf("提交失败，响应内容: %s", string(responseBody)), w)
		} else {
			dialog.ShowError(fmt.Errorf("提交失败，状态码: %d", resp.StatusCode), w)
		}
		return fmt.Errorf("提交失败，状态码: %d", resp.StatusCode), nil
	}
	//dlg := dialog.NewInformation("提交成功", "测试报告已成功提交到Redmine,可到redmine查看,点确定后自动删除.", w)
	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("提交失败，响应内容: %s", string(r)), nil
	}
	return nil, r

}
