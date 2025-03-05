package main

import (
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2/dialog"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// getRedmineProjectUsers 根据项目ID获取项目人员
func getRedmineProjectUsers(projectID interface{}) ([]string, error) {
	var id int
	switch v := projectID.(type) {
	case int:
		id = v
	case string:
		parsedID, err := strconv.Atoi(strings.Split(v, "|")[0])
		if err != nil {
			return nil, fmt.Errorf("无效的项目ID: %v", err)
		}
		id = parsedID
	default:
		return nil, fmt.Errorf("不支持的项目ID类型")
	}

	client := &http.Client{}
	url := fmt.Sprintf("%s/projects/%d/memberships.json", config.Redmine.URL, id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Redmine-API-Key", config.Redmine.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var projectRsps struct {
		Memberships []struct {
			User struct {
				Name string  `json:"name"`
				Id   float64 `json:"id"`
			} `json:"user"`
		} `json:"memberships"`
	}
	println(string(body))
	err = json.Unmarshal(body, &projectRsps)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	var projectUsers []string
	for _, membership := range projectRsps.Memberships {
		projectUsers = append(projectUsers, fmt.Sprintf("%.0f|%s",
			membership.User.Id, membership.User.Name,
		))
	}

	return projectUsers, nil
}

// getRedmineProjects
func getRedmineProjects() []string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", config.Redmine.URL+"/projects.json", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("X-Redmine-API-Key", config.Redmine.APIKey)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	rsp := map[string]interface{}{}
	if err := json.Unmarshal(body, &rsp); err != nil {
		return nil
	}
	projects := rsp["projects"].([]interface{})
	var projectList []string
	for _, project := range projects {
		projectMap := project.(map[string]interface{})
		projectName := projectMap["name"].(string)
		projectId := projectMap["id"].(float64)

		projectList = append(projectList, fmt.Sprintf("%.0f|%s", projectId, projectName))

	}
	return projectList
}

// 辅助函数：从map中按索引获取key
func getKeyByIndex(m map[string]bool, index int) string {
	i := 0
	for k := range m {
		if i == index {
			return k
		}
		i++
	}
	return ""
}

// 辅助函数：将字符串转换为整数
func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		dialog.ShowError(fmt.Errorf("无效的用户ID: %v", err), nil)
		return 0
	}
	return i
}
