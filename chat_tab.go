package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strings"
)

// 创建聊天Tab
func createChatTab(w fyne.Window) fyne.CanvasObject {
	messages := container.NewVBox()
	msgScroll := container.NewVScroll(messages)
	userInput := widget.NewEntry()
	userInput.SetPlaceHolder("输入消息...")
	history := []map[string]interface{}{}

	sendButton := widget.NewButton("发送", func() {
		userMessage := userInput.Text
		systemMesage := ""
		if userMessage != "" {
			history = append(history, map[string]interface{}{"role": "user", "message": userMessage})
			messages.Add(widget.NewLabelWithStyle("你", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}))
			userMsgLabel := widget.NewLabelWithStyle(userMessage, fyne.TextAlignTrailing, fyne.TextStyle{Bold: false})
			messages.Add(userMsgLabel)
			userInput.SetText("")
			messages.Add(widget.NewLabelWithStyle("AI("+config.CodeGPT.Model+"):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			var lastAi *widget.RichText
			thinkAi := map[*widget.Button]*container.Scroll{}
			err := sendToOllamaByChatByPrompt(userMessage, config.CodeGPT.APIKey, config.CodeGPT.BaseURL,
				config.CodeGPT.Model, config.ChatPromptTemplate, func(s string, b bool) {
					if lastAi == nil {
						fmt.Println(s)
						var btnIcon *widget.Button
						if strings.HasPrefix(s, "<think>") {
							btnIcon = widget.NewButtonWithIcon("", theme.Icon(theme.IconNameArrowDropDown), func() {
								if thinkAi[btnIcon] != nil {
									if thinkAi[btnIcon].Visible() {
										thinkAi[btnIcon].Hide()
										thinkAi[btnIcon].Content.Hide()
										btnIcon.SetIcon(theme.Icon(theme.IconNameArrowDropUp))
									} else {
										thinkAi[btnIcon].Show()
										thinkAi[btnIcon].Content.Show()
										btnIcon.SetIcon(theme.Icon(theme.IconNameArrowDropDown))
									}
									messages.Refresh()
								}
							})
							box := container.NewHBox(btnIcon)
							messages.Add(container.NewBorder(nil, nil, box, nil, widget.NewLabel("思考中 ...")))
							s = ""
						}
						systemMesage += s
						lastAi = widget.NewRichTextFromMarkdown(systemMesage)
						lastAi.Wrapping = fyne.TextWrapWord
						scroll := container.NewHScroll(lastAi)
						messages.Add(scroll)
						if btnIcon != nil {
							thinkAi[btnIcon] = scroll
						}
					} else {
						if strings.HasPrefix(s, "</think>") {
							lastAi.ParseMarkdown(systemMesage + "\n\n```  \n\n")
							systemMesage = ""
							lastAi = widget.NewRichTextFromMarkdown(s)
							lastAi.Wrapping = fyne.TextWrapWord
							messages.Add(container.NewHScroll(lastAi))
						}
						systemMesage += s
						lastAi.ParseMarkdown(systemMesage)
						lastAi.Refresh()
					}
					msgScroll.ScrollToBottom()
					if b {
						history = append(history, map[string]interface{}{"role": "assistant", "message": systemMesage})
						systemMesage = ""
						lastAi = nil
					}
				}, history[1:]...)
			if err != nil {
				messages.Add(widget.NewLabelWithStyle("OoO:\n"+err.Error()+"\n", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			}
		}
	})

	userInput.OnSubmitted = func(s string) {
		if s != "" {
			sendButton.OnTapped()
		}
	}

	return container.NewBorder(
		nil,
		container.NewBorder(nil, nil, widget.NewButtonWithIcon("", theme.Icon(theme.IconNameContentAdd), func() {
			history = []map[string]interface{}{}
			messages.RemoveAll()
			messages.Add(widget.NewLabelWithStyle("欢迎使用AI代码审查工具\n\n", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		}), sendButton, userInput),
		nil,
		nil, msgScroll,
	)
}
