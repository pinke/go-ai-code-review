package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// 启动应用
func main() {
	// 加载配置
	loadConfig()

	// 创建应用
	a := app.New()
	w := a.NewWindow("AI代码审查")

	// 创建UI
	createUI(w)

	// 启动应用
	w.ShowAndRun()
}

// 创建UI
func createUI(w fyne.Window) {
	// 创建聊天Tab
	chatTab := createChatTab(w)
	// 创建项目Tab
	projectTab := createProjectTab(w)
	// 创建AI配置Tab
	aiConfigTab := createAIConfigTab(w)
	// 创建Redmine配置Tab
	redmineConfigTab := createRedmineConfigTab(w)

	// 右侧容器，包含所有Tab
	rightContainer := container.NewBorder(nil, nil, nil, nil, chatTab, projectTab, aiConfigTab, redmineConfigTab)

	// 左侧菜单
	menu := widget.NewList(
		func() int { return 4 }, // 列表项数量为4
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			// 菜单项标签
			labels := []string{"与AI对话", "项目列表", "AI配置", "Redmine配置"}
			o.(*widget.Label).SetText(labels[i])
		},
	)
	// 菜单项选择事件处理
	menu.OnSelected = func(id widget.ListItemID) {
		switch id {
		case 0:
			chatTab.Show()
			projectTab.Hide()
			aiConfigTab.Hide()
			redmineConfigTab.Hide()
		case 1:
			chatTab.Hide()
			projectTab.Show()
			aiConfigTab.Hide()
			redmineConfigTab.Hide()
		case 2:
			chatTab.Hide()
			projectTab.Hide()
			aiConfigTab.Show()
			redmineConfigTab.Hide()
		case 3:
			chatTab.Hide()
			projectTab.Hide()
			aiConfigTab.Hide()
			redmineConfigTab.Show()
		}
	}

	// 主窗口布局
	mainContainer := container.NewHSplit(
		menu,
		rightContainer,
	)
	// 默认显示项目列表
	w.SetContent(projectTab)

	mainContainer.SetOffset(0.2)
	w.SetContent(mainContainer)

	projectTab.Hide()
	aiConfigTab.Hide()
	redmineConfigTab.Hide()
	w.Resize(fyne.NewSize(800, 600))
}