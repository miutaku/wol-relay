//go:build nativegui

package nativegui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/miutaku/wol-relay/internal/agent"
	"github.com/miutaku/wol-relay/internal/config"
)

type Options struct {
	Agent       *agent.Agent
	ConfigPath  string
	AgentErrors <-chan error
}

func Run(ctx context.Context, opts Options) error {
	app := fyneapp.NewWithID("com.github.miutaku.wol-relay")
	window := app.NewWindow("wol-relay")
	window.Resize(fyne.NewSize(760, 560))

	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	cfg := opts.Agent.Config()
	nodeLabel := widget.NewLabel(cfg.NodeName)

	nodeName := widget.NewEntry()
	nodeName.SetPlaceHolder("例: desktop")
	nodeName.SetText(cfg.NodeName)
	listenHTTP := widget.NewEntry()
	listenHTTP.SetPlaceHolder("例: 127.0.0.1:8080 / 192.168.20.10:8080")
	listenHTTP.SetText(cfg.ListenHTTP)
	listenMagic := widget.NewEntry()
	listenMagic.SetPlaceHolder("例: :9, 0.0.0.0:9009")
	listenMagic.SetText(strings.Join(cfg.ListenMagic, ", "))
	allowedMagicSources := widget.NewEntry()
	allowedMagicSources.SetPlaceHolder("例: 192.168.10.0/24, 192.168.10.50")
	allowedMagicSources.SetText(strings.Join(cfg.AllowedMagicSources, ", "))
	defaultRelay := widget.NewEntry()
	defaultRelay.SetPlaceHolder("例: http://192.168.20.10:8080")
	defaultRelay.SetText(cfg.DefaultRelay)
	defaultTarget := widget.NewEntry()
	defaultTarget.SetPlaceHolder("例: 255.255.255.255:9 / 192.168.10.255:9")
	defaultTarget.SetText(cfg.DefaultTarget)
	sharedSecret := widget.NewPasswordEntry()
	sharedSecret.SetPlaceHolder("例: 長いランダム文字列")
	sharedSecret.SetText(cfg.Auth.SharedSecret)
	allowInsecure := widget.NewCheck("HMAC認証を無効化", nil)
	allowInsecure.SetChecked(cfg.Auth.AllowInsecure)
	notificationsEnabled := widget.NewCheck("OS通知を有効化", nil)
	notificationsEnabled.SetChecked(cfg.Notifications.Enabled)
	lightweight := widget.NewCheck("軽量モード", nil)
	lightweight.SetChecked(cfg.Lightweight)
	syncSettings := func() {
		cfg := opts.Agent.Config()
		nodeName.SetText(cfg.NodeName)
		listenHTTP.SetText(cfg.ListenHTTP)
		listenMagic.SetText(strings.Join(cfg.ListenMagic, ", "))
		allowedMagicSources.SetText(strings.Join(cfg.AllowedMagicSources, ", "))
		defaultRelay.SetText(cfg.DefaultRelay)
		defaultTarget.SetText(cfg.DefaultTarget)
		sharedSecret.SetText(cfg.Auth.SharedSecret)
		allowInsecure.SetChecked(cfg.Auth.AllowInsecure)
		notificationsEnabled.SetChecked(cfg.Notifications.Enabled)
		lightweight.SetChecked(cfg.Lightweight)
		nodeLabel.SetText(cfg.NodeName)
	}

	hostName := widget.NewEntry()
	hostName.SetPlaceHolder("例: nas")
	hostMAC := widget.NewEntry()
	hostMAC.SetPlaceHolder("例: 00:11:22:33:44:55")
	hostIP := widget.NewEntry()
	hostIP.SetPlaceHolder("例: 192.168.10.20")
	hostBroadcast := widget.NewEntry()
	hostBroadcast.SetPlaceHolder("例: 192.168.10.255:9")
	hostRelay := widget.NewEntry()
	hostRelay.SetPlaceHolder("例: http://192.168.20.10:8080")
	hostAllowedBy := widget.NewEntry()
	hostAllowedBy.SetPlaceHolder("例: desktop, laptop")
	checkEnabled := widget.NewCheck("起動確認", nil)
	checkMethod := widget.NewSelect([]string{"tcp", "icmp"}, nil)
	checkMethod.SetSelected("tcp")
	checkPort := widget.NewEntry()
	checkPort.SetPlaceHolder("例: 22 / 3389")
	checkPort.SetText("22")
	checkTimeout := widget.NewEntry()
	checkTimeout.SetPlaceHolder("例: 2m")
	checkTimeout.SetText("2m")
	checkInterval := widget.NewEntry()
	checkInterval.SetPlaceHolder("例: 3s")
	checkInterval.SetText("3s")
	hostFormTitle := widget.NewLabelWithStyle("ホスト追加", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hostSaveButton := widget.NewButton("追加", nil)
	cancelHostEditButton := widget.NewButton("編集をキャンセル", nil)
	cancelHostEditButton.Hide()
	editingHost := ""
	clearHostForm := func() {
		hostName.SetText("")
		hostMAC.SetText("")
		hostIP.SetText("")
		hostBroadcast.SetText("")
		hostRelay.SetText("")
		hostAllowedBy.SetText("")
		checkEnabled.SetChecked(false)
		checkMethod.SetSelected("tcp")
		checkPort.SetText("22")
		checkTimeout.SetText("2m")
		checkInterval.SetText("3s")
		editingHost = ""
		hostFormTitle.SetText("ホスト追加")
		hostSaveButton.SetText("追加")
		cancelHostEditButton.Hide()
	}
	loadHostForm := func(host config.HostConfig) {
		editingHost = hostKey(host)
		hostName.SetText(host.Name)
		hostMAC.SetText(host.MAC)
		hostIP.SetText(host.IP)
		hostBroadcast.SetText(host.Broadcast)
		hostRelay.SetText(host.Relay)
		hostAllowedBy.SetText(strings.Join(host.AllowedBy, ", "))
		checkEnabled.SetChecked(host.Check.Enabled)
		method := host.Check.Method
		if method == "" {
			method = "tcp"
		}
		checkMethod.SetSelected(method)
		port := host.Check.Port
		if port == 0 {
			port = 22
		}
		checkPort.SetText(strconv.Itoa(port))
		timeout := host.Check.Timeout
		if timeout == "" {
			timeout = "2m"
		}
		checkTimeout.SetText(timeout)
		interval := host.Check.Interval
		if interval == "" {
			interval = "3s"
		}
		checkInterval.SetText(interval)
		hostFormTitle.SetText("ホスト編集")
		hostSaveButton.SetText("変更を保存")
		cancelHostEditButton.Show()
		status.SetText("ホスト設定を編集できます。")
	}

	list := container.NewVBox()
	var refresh func()
	refresh = func() {
		cfg := opts.Agent.Config()
		list.Objects = nil
		if len(cfg.Hosts) == 0 {
			list.Add(widget.NewLabel("起こしたいPCやサーバーを追加してください。"))
		}
		for _, host := range cfg.Hosts {
			host := host
			title := host.Name
			if title == "" {
				title = host.MAC
			}
			meta := strings.Join(nonEmpty(host.MAC, host.IP, broadcastLabel(host.Broadcast), relayLabel(host.Relay), allowedByLabel(host.AllowedBy), checkLabel(host.Check)), " / ")
			wakeButton := widget.NewButton("Wake", func() {
				status.SetText("送信中...")
				go func() {
					result, err := opts.Agent.Wake(context.Background(), title, agent.SourceCLI)
					fyne.Do(func() {
						if err != nil {
							status.SetText(err.Error())
							return
						}
						status.SetText(result.Message)
					})
				}()
			})
			editButton := widget.NewButton("編集", func() {
				loadHostForm(host)
			})
			deleteButton := widget.NewButton("削除", func() {
				cfg, ok := opts.Agent.DeleteHost(title)
				if !ok {
					status.SetText("削除対象が見つかりません。")
					return
				}
				if opts.ConfigPath != "" {
					if err := saveConfig(opts, cfg); err != nil {
						status.SetText(err.Error())
						return
					}
				}
				status.SetText("削除しました。")
				if editingHost == title {
					clearHostForm()
				}
				refresh()
			})
			list.Add(widget.NewCard(title, meta, container.NewHBox(wakeButton, editButton, deleteButton)))
		}
		list.Refresh()
	}

	saveSettingsButton := widget.NewButton("全体設定を保存", func() {
		cfg := opts.Agent.Config()
		cfg.NodeName = strings.TrimSpace(nodeName.Text)
		cfg.ListenHTTP = strings.TrimSpace(listenHTTP.Text)
		cfg.ListenMagic = splitCSV(listenMagic.Text)
		cfg.AllowedMagicSources = splitCSV(allowedMagicSources.Text)
		cfg.DefaultRelay = strings.TrimSpace(defaultRelay.Text)
		cfg.DefaultTarget = strings.TrimSpace(defaultTarget.Text)
		cfg.Lightweight = lightweight.Checked
		cfg.Auth.SharedSecret = strings.TrimSpace(sharedSecret.Text)
		cfg.Auth.AllowInsecure = allowInsecure.Checked
		cfg.Notifications.Enabled = notificationsEnabled.Checked
		if err := saveConfig(opts, cfg); err != nil {
			status.SetText(err.Error())
			return
		}
		syncSettings()
		status.SetText("全体設定を保存しました。待ち受けアドレスの変更は次回起動から反映されます。")
	})

	cancelHostEditButton.OnTapped = func() {
		clearHostForm()
		status.SetText("編集をキャンセルしました。")
	}
	hostSaveButton.OnTapped = func() {
		port, err := strconv.Atoi(strings.TrimSpace(checkPort.Text))
		if err != nil || port <= 0 {
			port = 22
		}
		method := checkMethod.Selected
		if method == "" {
			method = "tcp"
		}
		host := config.HostConfig{
			Name:      strings.TrimSpace(hostName.Text),
			MAC:       strings.TrimSpace(hostMAC.Text),
			IP:        strings.TrimSpace(hostIP.Text),
			Broadcast: strings.TrimSpace(hostBroadcast.Text),
			Relay:     strings.TrimSpace(hostRelay.Text),
			AllowedBy: splitCSV(hostAllowedBy.Text),
			Check: config.CheckConfig{
				Enabled:  checkEnabled.Checked,
				Method:   method,
				Port:     port,
				Timeout:  strings.TrimSpace(checkTimeout.Text),
				Interval: strings.TrimSpace(checkInterval.Text),
			},
		}
		if host.Name == "" || host.MAC == "" {
			status.SetText("名前とMACアドレスを入力してください。")
			return
		}
		var cfg config.Config
		if editingHost != "" {
			var ok bool
			cfg, ok = replaceHost(opts.Agent.Config(), editingHost, host)
			if !ok {
				status.SetText("編集対象が見つかりません。")
				return
			}
		} else {
			cfg = opts.Agent.UpsertHost(host)
		}
		if err := saveConfig(opts, cfg); err != nil {
			status.SetText(err.Error())
			return
		}
		wasEditing := editingHost != ""
		clearHostForm()
		if wasEditing {
			status.SetText("変更を保存しました。")
		} else {
			status.SetText("保存しました。")
		}
		refresh()
	}

	settingsForm := container.NewVBox(
		widget.NewLabelWithStyle("全体設定", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			sampled("Agent名", "desktop", nodeName),
			sampled("REST待ち受け", "127.0.0.1:8080 / 192.168.20.10:8080", listenHTTP),
			sampled("Magic待ち受け", ":9, 0.0.0.0:9009", listenMagic),
			sampled("Magic送信元許可", "192.168.10.0/24, 192.168.10.50", allowedMagicSources),
			sampled("標準送信先Agent URL", "http://192.168.20.10:8080", defaultRelay),
			sampled("標準ブロードキャスト", "255.255.255.255:9 / 192.168.10.255:9", defaultTarget),
			sampled("共有シークレット", "長いランダム文字列", sharedSecret),
			container.NewVBox(allowInsecure, notificationsEnabled, lightweight),
		),
		saveSettingsButton,
	)

	hostForm := container.NewVBox(
		hostFormTitle,
		container.NewGridWithColumns(2,
			sampled("名前", "nas", hostName),
			sampled("MACアドレス", "00:11:22:33:44:55", hostMAC),
			sampled("IPアドレス", "192.168.10.20", hostIP),
			sampled("ブロードキャスト", "192.168.10.255:9", hostBroadcast),
			sampled("送信先Agent URL", "http://192.168.20.10:8080", hostRelay),
			sampled("許可する送信元Agent名", "desktop, laptop", hostAllowedBy),
			checkEnabled,
			sampled("確認方法", "tcp / icmp", checkMethod),
			sampled("確認TCPポート", "22 / 3389", checkPort),
			sampled("確認タイムアウト", "2m", checkTimeout),
			sampled("確認間隔", "3s", checkInterval),
		),
		container.NewHBox(hostSaveButton, cancelHostEditButton),
	)

	header := container.NewBorder(nil, nil, widget.NewLabel("wol-relay"), nodeLabel)
	tabs := container.NewAppTabs(
		container.NewTabItem("全体設定", container.NewVScroll(settingsForm)),
		container.NewTabItem("ホスト", container.NewVScroll(container.NewVBox(hostForm, list))),
	)
	content := container.NewBorder(header, status, nil, nil, tabs)
	window.SetContent(content)
	refresh()

	if opts.AgentErrors != nil {
		go func() {
			select {
			case err := <-opts.AgentErrors:
				if err != nil {
					fyne.Do(func() {
						status.SetText("Agentの起動に失敗しました: " + err.Error())
					})
				}
			case <-ctx.Done():
			}
		}()
	}

	go func() {
		<-ctx.Done()
		fyne.Do(app.Quit)
	}()

	window.ShowAndRun()
	return nil
}

func checkLabel(check config.CheckConfig) string {
	if !check.Enabled {
		return ""
	}
	port := check.Port
	if port == 0 {
		port = 22
	}
	return fmt.Sprintf("確認: %s/%d", check.Method, port)
}

func sampled(label string, sample string, object fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(widget.NewLabel(label+"  例: "+sample), object)
}

func broadcastLabel(broadcast string) string {
	if strings.TrimSpace(broadcast) == "" {
		return ""
	}
	return "送信先: " + broadcast
}

func relayLabel(relay string) string {
	if strings.TrimSpace(relay) == "" {
		return ""
	}
	return "送信先Agent: " + relay
}

func allowedByLabel(nodes []string) string {
	if len(nodes) == 0 {
		return ""
	}
	return "許可: " + strings.Join(nodes, ", ")
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func saveConfig(opts Options, cfg config.Config) error {
	if opts.ConfigPath != "" {
		if err := config.Save(opts.ConfigPath, cfg); err != nil {
			return err
		}
		saved, err := config.Load(opts.ConfigPath)
		if err != nil {
			return err
		}
		opts.Agent.UpdateConfig(saved)
		return nil
	}
	opts.Agent.UpdateConfig(cfg)
	return nil
}

func replaceHost(cfg config.Config, target string, next config.HostConfig) (config.Config, bool) {
	needle := normalizeKey(target)
	for i, host := range cfg.Hosts {
		if strings.EqualFold(host.Name, target) || normalizeKey(host.MAC) == needle {
			cfg.Hosts[i] = next
			return cfg, true
		}
	}
	return cfg, false
}

func hostKey(host config.HostConfig) string {
	if strings.TrimSpace(host.Name) != "" {
		return host.Name
	}
	return host.MAC
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(value, ":", ""), "-", ""))
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
