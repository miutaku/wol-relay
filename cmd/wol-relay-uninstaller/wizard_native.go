//go:build nativegui

package main

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func runWizard() {
	a := fyneapp.NewWithID("com.github.miutaku.wol-relay.uninstaller")
	w := a.NewWindow("wol-relay アンインストール")
	w.Resize(fyne.NewSize(560, 360))

	title := widget.NewLabelWithStyle("wol-relay アンインストール", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	message := widget.NewLabel("このウィザードは wol-relay をこのユーザーの環境から削除します。")
	removeConfig := widget.NewCheck("設定ファイルも削除する", nil)
	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	uninstallButton := widget.NewButton("アンインストール", nil)
	closeButton := widget.NewButton("閉じる", func() {
		a.Quit()
	})
	closeButton.Disable()

	uninstallButton.OnTapped = func() {
		uninstallButton.Disable()
		removeConfig.Disable()
		status.SetText("アンインストール中...")
		go func() {
			err := uninstall(removeConfig.Checked)
			fyne.Do(func() {
				if err != nil {
					status.SetText("アンインストールに失敗しました: " + err.Error())
					uninstallButton.Enable()
					removeConfig.Enable()
					return
				}
				message.SetText("アンインストールが完了しました。")
				status.SetText("")
				closeButton.Enable()
			})
		}()
	}

	w.SetContent(container.NewBorder(
		container.NewVBox(title, message, removeConfig),
		container.NewHBox(uninstallButton, closeButton),
		nil,
		nil,
		container.NewVBox(status),
	))
	w.ShowAndRun()
}
