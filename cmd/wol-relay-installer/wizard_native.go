//go:build nativegui

package main

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func runWizard() {
	a := fyneapp.NewWithID("com.github.miutaku.wol-relay.installer")
	w := a.NewWindow("wol-relay セットアップ")
	w.Resize(fyne.NewSize(560, 360))

	title := widget.NewLabelWithStyle("wol-relay セットアップ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	message := widget.NewLabel("このウィザードは wol-relay をこのユーザーの環境へインストールします。")
	status := widget.NewLabel("")
	installButton := widget.NewButton("インストール", nil)
	closeButton := widget.NewButton("閉じる", func() {
		a.Quit()
	})
	closeButton.Disable()

	installButton.OnTapped = func() {
		installButton.Disable()
		status.SetText("インストール中...")
		go func() {
			err := run()
			fyne.Do(func() {
				if err != nil {
					status.SetText("インストールに失敗しました: " + err.Error())
					installButton.Enable()
					return
				}
				message.SetText("インストールが完了しました。デスクトップの wol-relay.exe から起動できます。")
				status.SetText("")
				closeButton.Enable()
			})
		}()
	}

	w.SetContent(container.NewBorder(
		container.NewVBox(title, message),
		container.NewHBox(installButton, closeButton),
		nil,
		nil,
		container.NewVBox(status),
	))
	w.ShowAndRun()
}
