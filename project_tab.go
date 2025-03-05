package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 创建项目列表Tab
func createProjectTab(w fyne.Window) fyne.CanvasObject {
	// 左侧项目列表
	projectSelectId := -1
	selectedProject := Project{}
	selectedProjectVer := ""
	projectList := widget.NewList(
		func() int { return len(config.Projects) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(config.Projects[i].Name)
		},
	)

	// 文件列表显示
	fileList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText("")
		},
	)

	deleteButton := widget.NewButton("删除", func() {
		if projectSelectId >= 0 {
			config.Projects = append(config.Projects[:projectSelectId], config.Projects[projectSelectId+1:]...)
			projectList.Refresh()
			saveConfig()
		}
	})
	var pullButton *widget.Button
	pull := func() {
		if projectSelectId >= 0 {
			cmd := exec.Command("git", "-C", selectedProject.Path, "pull")
			output, err := cmd.Output()
			if err != nil {
				dialog.ShowError(fmt.Errorf("更新失败%s: %v", selectedProject.Name, err), w)
				return
			}
			dialog.ShowInformation("更新成功"+selectedProject.Name, string(output), w)
		}
		pullButton.Enable()
	}
	pullButton = widget.NewButton("更新", func() {
		pullButton.Disable()
		defer func() {
			pullButton.Enable()
		}()
		go pull()
	})
	deleteButton.Disable()
	pullButton.Disable()

	// 右侧git log列表
	logList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {},
	)

	// 项目选择事件
	projectList.OnUnselected = func(id widget.ListItemID) {
		if projectSelectId == id {
			projectSelectId = -1
		}
	}
	logs := []string{}
	projectList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 {
			deleteButton.Enable()
			pullButton.Enable()
		} else {
			deleteButton.Disable()
			pullButton.Disable()
		}
		projectSelectId = id
		selectedProject = config.Projects[id]
		cmd := exec.Command("git", "-C", selectedProject.Path, "log", "--pretty=format:%h - %s - %an - %cd", "--date=short")
		output, err := cmd.Output()
		if err != nil {
			dialog.ShowError(fmt.Errorf("获取git log失败: %v", err), w)
			return
		}
		logs = strings.Split(string(output), "\n")
		logList.Length = func() int { return len(logs) }
		logList.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(logs[i])
		}
		logList.Refresh()
	}

	// 日志选择事件
	logList.OnSelected = func(id widget.ListItemID) {
		idStr := strings.Split(logs[id], " ")[0]
		selectedProjectVer = idStr
		cmd := exec.Command("git", "-C", selectedProject.Path, "show", "--name-only", "--pretty=format:", idStr)
		output, err := cmd.Output()
		if err != nil {
			dialog.ShowError(fmt.Errorf("获取git日志失败: %v", err), w)
			return
		}
		files := strings.Split(string(output), "\n")
		uniqueFiles := make(map[string]bool)
		for _, file := range files {
			if file != "" {
				uniqueFiles[file] = true
			}
		}
		fileList.Length = func() int { return len(uniqueFiles) }
		fileList.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(getKeyByIndex(uniqueFiles, i))
		}
		fileList.Refresh()
	}

	// 当双击 fileList 时
	preSelectId := -1
	var preSelectTime int64 = 0
	fileList.OnUnselected = func(id widget.ListItemID) {
		fmt.Println("OnUnselected ", id, preSelectId, preSelectTime)
	}
	fileList.OnSelected = func(id widget.ListItemID) {
		fmt.Println("OnSelected ", id, preSelectId, preSelectTime)
	}

	var procFunc = func() {
		if selectedProjectVer == "" {
			dialog.ShowError(fmt.Errorf("请选择提交版本"), w)
			return
		}
		if err := AiReview(selectedProject, selectedProjectVer, w); err != nil {
			dialog.ShowError(fmt.Errorf("代码审查失败: %v", err), w)
			return
		}
	}
	reviewButton := widget.NewButton("代码审查", procFunc)

	// 创建工具栏按钮
	addButton := widget.NewButton("添加", func() {
		projectForm, projectFormName, projectFormPath, projectFormRedmineProjectId, projectFormRedmineAssignedToUserId := createProjectForm(w)
		dlg := dialog.NewCustomConfirm("添加项目", "确定", "取消", projectForm, func(b bool) {
			if b {
				name := projectFormName.Text
				path := projectFormPath.Text
				redmineProjectId := projectFormRedmineProjectId.Text
				redmineAssignedToUserId := projectFormRedmineAssignedToUserId.Text

				_, err := os.Stat(filepath.Join(path, ".git"))
				if err != nil {
					dialog.ShowError(fmt.Errorf("非Git仓库: %s", path), w)
					return
				}
				config.Projects = append(config.Projects, Project{
					Path:                    path,
					Name:                    name,
					RedmineProjectId:        redmineProjectId,
					RedmineAssignedToUserId: redmineAssignedToUserId,
				})
				projectList.Refresh()
				saveConfig()
			}
		}, w)
		dlg.Resize(fyne.NewSize(400, 300))
		dlg.Show()
	})

	// 创建工具栏
	toolbar := container.NewHBox(
		addButton,
		deleteButton,
		pullButton,
	)

	// 布局
	left := container.NewBorder(toolbar, nil, nil, nil, projectList)
	vSplit := container.NewVSplit(logList, fileList)
	vSplit.SetOffset(0.8)
	right := container.NewBorder(
		nil,
		container.NewVBox(reviewButton),
		nil, nil, vSplit,
	)
	split := container.NewHSplit(left, right)
	split.SetOffset(0.3)

	return split
}

// 创建项目表单
func createProjectForm(w fyne.Window) (ctx fyne.CanvasObject, projectName, projectPath *widget.Entry,
	projectRedmineProjectId *widget.SelectEntry, projectRedmineAssignedToUserId *widget.Entry) {
	projectFormName := widget.NewEntry()
	projectFormPath := widget.NewEntry()

	projectFormPathCt := container.NewBorder(nil, nil, nil,
		widget.NewButton("选择路径", func() {
			dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				if uri == nil {
					return
				}
				_, err = os.Stat(filepath.Join(uri.Path(), ".git"))
				if err != nil {
					dialog.ShowError(fmt.Errorf("非Git仓库: %s", uri.Path()), w)
					return
				}
				projectFormPath.SetText(uri.Path())
			}, w)
		}),
		projectFormPath,
	)
	projectFormRedmineProjectId := widget.NewSelectEntry(getRedmineProjects())
	projectFormRedmineAssignedToUserId := widget.NewEntry()

	return container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("项目名称:", projectFormName),
			widget.NewFormItem("项目路径:", projectFormPathCt),
			widget.NewFormItem("Redmine项目ID:", projectFormRedmineProjectId),
			widget.NewFormItem("Redmine分配人ID:", projectFormRedmineAssignedToUserId),
		),
	), projectFormName, projectFormPath, projectFormRedmineProjectId, projectFormRedmineAssignedToUserId
}

func AiReview(project Project, ver string, w fyne.Window) error {
	// 获取git diff
	cmd := exec.Command("git", "-C", project.Path, "diff", ver+"~1", ver)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("获取git diff失败: %v", err)
	}
	diff := string(output)
	if config.CodeGPT.Provider != "Ollama API" { // config.CodeGPT.Provider != "OpenAI API"
		//TODO:
		//return fmt.Errorf("未实现的AI接口")
		dialog.ShowError(fmt.Errorf("TODO:未实现的AI接口: %v", err), w)
		return nil
	}

	// 代码审查按钮
	var timer *time.Timer
	var elapsed int64 = 0

	stopTimer := func() {
		if timer != nil {
			timer.Stop()
		}
		elapsed = 0
	}
	defer stopTimer()
	// 代码审查时间显示
	reviewTimeLabel := widget.NewLabel("审查用时: 0s")
	reviewTimeLabel.Alignment = fyne.TextAlignCenter

	redmineProject := widget.NewSelectEntry(getRedmineProjects())
	aiAdvice := widget.NewRichTextFromMarkdown("")
	aiAdviceEditor := widget.NewMultiLineEntry()
	aiAdviceEditor.SetText("")

	reviewTimeLabel.Show()
	reviewTimeLabel.Refresh()
	startTime := time.Now().UnixMilli()
	//aiAdvice.Text = s
	aiAdvice.Resize(fyne.NewSize(400, 400))
	redminProjectAssignedToUser := widget.NewSelectEntry([]string{project.RedmineAssignedToUserId})
	redmineProject.OnChanged = func(s string) {
		projectId := strings.Split(s, "|")[0]
		users, err2 := getRedmineProjectUsers(projectId)
		if err2 != nil {
			dialog.ShowError(fmt.Errorf("获取redmine项目人员失败: %v", err2), w)
			return
		}
		redminProjectAssignedToUser.SetOptions(users)
	}

	scrollAdvice := container.NewScroll(aiAdvice)
	scrollAdviceEditor := container.NewScroll(aiAdviceEditor)
	scrollAdvice.Resize(fyne.NewSize(400, 400))
	scrollAdviceEditor.Resize(fyne.NewSize(400, 400))
	aiAdviceEditor.OnChanged = func(s string) {
		aiAdvice.ParseMarkdown(s)
	}
	scrollBar := container.NewHSplit(scrollAdviceEditor, scrollAdvice)
	aiMesageBox := container.NewBorder(container.NewHBox(
		widget.NewButton("编辑", func() {
			//aiAdviceEditor.SetText(aiAdvice.String())
			if scrollAdviceEditor.Visible() {
				scrollAdviceEditor.Hide()
			} else {
				scrollAdviceEditor.Show()
			}
		}), widget.NewButton("预览", func() {
			if scrollAdvice.Visible() {
				scrollAdvice.Hide()
			} else {
				scrollAdvice.Show()
			}
		}),
		reviewTimeLabel),
		nil, nil, nil,
		scrollBar,
	)
	aiMesageBox.Resize(fyne.NewSize(400, 400))
	dlg := dialog.NewCustomConfirm("AI代码审查结果", "提交redmine", "取消",
		container.NewBorder(widget.NewForm(
			widget.NewFormItem("Redmine项目:", redmineProject),
			widget.NewFormItem("Redmine分配人:", redminProjectAssignedToUser)), nil, nil, nil,
			aiMesageBox,
		), func(b bool) {
			if elapsed > 0 {
				//dialog.ShowError(fmt.Errorf("请稍等,AI处理中 ..."), w)

				return // 已经提交过了
			}
			if b {
				projectId := strings.Split(redmineProject.Text, "|")[0]
				assignToUserId := mustAtoi(strings.Split(redminProjectAssignedToUser.Text, "|")[0])

				if err, _ := submitToRedmine("[AI审查]", projectId, assignToUserId, ver, diff, aiAdvice.String(), w); err != nil {
					dialog.ShowError(fmt.Errorf("同步redmine失败: %v", err), w)
					return
				} else {
					dialog.ShowInformation("提交到redmine", "提交到redmine成功!", w)
				}

			} else {
				//popup message
				//dialog.ShowError(fmt.Errorf("用户取消提交"), w)
			}
		}, w)
	//dlg.SetDismissText()
	dlg.Resize(fyne.NewSize(500, 400))
	dlg.Show()

	go func() {
		reviewTimeLabel.SetText(fmt.Sprintf("审查中 ... %ds  ", elapsed/1000))

		if err := sendToOllamaByChat(diff, config.CodeGPT.APIKey, config.CodeGPT.BaseURL, config.CodeGPT.Model,
			func(response string, done bool) {
				aiAdviceEditor.SetText(aiAdviceEditor.Text + response)
				//aiAdvice.AppendMarkdown(response)
				elapsed = time.Now().UnixMilli() - startTime
				if done {
					reviewTimeLabel.SetText(fmt.Sprintf("审查完成,用时: %ds!", elapsed/1000))
					stopTimer()
				} else {
					reviewTimeLabel.SetText(fmt.Sprintf("审查中 ... %ds  ", elapsed/1000))
					scrollAdviceEditor.ScrollToBottom()
					scrollAdvice.ScrollToBottom()
				}
				reviewTimeLabel.Refresh()
			}); err != nil {
			//return fmt.Errorf("发送到ollama失败: %v", err)
			dialog.ShowError(fmt.Errorf("发送到ollama失败: %v", err), w)
			stopTimer()
		}
	}()

	return nil
}
