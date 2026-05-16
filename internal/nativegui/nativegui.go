//go:build nativegui

package nativegui

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/miutaku/wol-relay/internal/agent"
	"github.com/miutaku/wol-relay/internal/autostart"
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
	loginStartup := widget.NewCheck("ログイン時に自動起動", nil)
	if autostart.IsSupported() {
		enabled, err := autostart.IsEnabled(opts.ConfigPath)
		if err == nil {
			loginStartup.SetChecked(enabled)
		}
	} else {
		loginStartup.Disable()
	}
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
	openConfigButton := widget.NewButton("設定ファイルのフォルダを開く", func() {
		if opts.ConfigPath == "" {
			status.SetText("設定ファイルの場所がわかりません。")
			return
		}
		if err := openPath(filepath.Dir(opts.ConfigPath)); err != nil {
			status.SetText(err.Error())
			return
		}
		status.SetText("設定ファイルのフォルダを開きました。")
	})
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
		cfg.Auth.SharedSecret = strings.TrimSpace(sharedSecret.Text)
		cfg.Auth.AllowInsecure = allowInsecure.Checked
		cfg.Notifications.Enabled = notificationsEnabled.Checked
		if err := saveConfig(opts, cfg); err != nil {
			status.SetText(err.Error())
			return
		}
		if loginStartup.Checked && !autostart.IsSupported() {
			status.SetText("このOSではアプリ内からの自動起動登録にまだ対応していません。")
			return
		}
		if autostart.IsSupported() {
			if err := autostart.SetEnabled(loginStartup.Checked, opts.ConfigPath); err != nil {
				status.SetText(err.Error())
				return
			}
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
		helpText("このPCで動く wol-relay 全体の設定です。初めて使う場合、多くの項目は初期値のままで問題ありません。"),
		sectionCard("このPCの基本情報",
			sampled("このPCの名前", "desktop",
				"このwol-relayを識別する名前です。「living-room-pc」「laptop」など、わかりやすい英数字の名前をつけてください。ホスト設定の「許可する送信元Agent名」でこの名前を使います。",
				nodeName),
			sampled("Wake依頼を受け付けるアドレス", "127.0.0.1:8080",
				"他のPCからのWake依頼を受け取るアドレスとポート番号です。\n・同じLAN内だけで使う場合 → 「127.0.0.1:8080」のままでOK\n・別のLAN（ルーターの向こう）からも受け付けたい場合 → 「192.168.10.20:8080」のようにこのPCのIPアドレスに変更してください",
				listenHTTP),
		),
		sectionCard("Wake on LAN信号の検知",
			sampled("受信ポート", ":9",
				"他のWake on LANアプリが送った信号を受け取るUDPポート番号です。通常は「:9」のままで問題ありません。変更すると他のWoLアプリと連携できなくなる場合があります。",
				listenMagic),
			sampled("信号を受け付けるIPアドレスの範囲", "192.168.10.0/24",
				"Wake on LAN信号を受け付けるIPアドレスの範囲です。\n・空のまま → どのPCからでも受け付ける\n・特定のPC1台だけ → 「192.168.10.5」\n・特定のネットワーク全体 → 「192.168.10.0/24」\n複数指定する場合はカンマ区切りで入力してください。",
				allowedMagicSources),
		),
		sectionCard("Wake信号の標準送り先",
			sampled("標準の中継先（別LAN用）", "http://192.168.20.10:8080",
				"別のLAN（ルーターの向こう）にいるPCを起こすとき、そのLAN側で動いているwol-relayのURLです。同じLAN内のPCだけを起こす場合は空でOKです。各ホストで個別に上書きもできます。",
				defaultRelay),
			sampled("標準のブロードキャスト宛先", "255.255.255.255:9",
				"同じLAN内のPCを起こすときにWake信号を送る宛先です。\n・「255.255.255.255:9」→ LAN全体に送る（通常はこれでOK）\n・「192.168.10.255:9」→ 特定のネットワークだけに送る\n各ホストで個別に上書きもできます。",
				defaultTarget),
		),
		sectionCard("セキュリティと通知の設定",
			sampled("共有シークレット（パスワード）", "長いランダム文字列",
				"連携するwol-relay同士が「本物かどうか」を確認するためのパスワードです。通信するすべてのPCで同じ値を設定してください。長くてランダムな文字列を推奨します。変更した場合は相手側も同じ値に変える必要があります。",
				sharedSecret),
			fieldCard("HMAC認証を無効化",
				"通常はオフのままにしてください。オンにすると上記パスワードによる認証をスキップします。テスト目的以外では使わないでください。",
				allowInsecure),
			fieldCard("OS通知を有効化",
				"Wake信号の送信結果や起動確認の結果をWindowsの通知として表示します。結果をすぐ知りたい場合はオンにしてください。",
				notificationsEnabled),
			fieldCard("ログイン時に自動起動",
				"PCにログインしたとき、このアプリを自動で起動します。常にWake on LANを受け付けたい場合はオンにしてください。",
				loginStartup),
			fieldCard("軽量モード",
				"GUIを表示せず、バックグラウンドだけで動作するモードです。Raspberry PiやサーバーなどGUIのない環境向けで、GUIからは変更できません。コマンドラインまたはインストーラー経由で設定します。",
				helpText("CLI / インストールモード専用")),
		),
		sectionCard("設定ファイル",
			fieldCard("設定ファイルの直接編集",
				"GUIで変更できない詳細な設定を変えたい場合、設定ファイルをテキストエディタで直接編集できます。編集後はアプリを再起動すると反映されます。",
				container.NewVBox(helpText(opts.ConfigPath), openConfigButton)),
		),
		saveSettingsButton,
	)

	hostForm := container.NewVBox(
		hostFormTitle,
		helpText("起こしたいPCやサーバーを登録します。同じLAN内のPCなら名前とMACアドレスだけで始められます。別のLAN（ルーターの向こう側）のPCを起こす場合は、送信先のwol-relay URLも指定してください。"),
		sectionCard("起こしたいPCの情報",
			sampled("表示名", "nas",
				"このアプリ内での表示名です。「gaming-pc」「nas」など、わかりやすい名前をつけてください。Wakeボタンや一覧に表示されます。",
				hostName),
			sampled("MACアドレス", "00:11:22:33:44:55",
				"起こしたいPCのネットワークアダプターに割り当てられた固有のIDです。Wake on LANはこのIDを使って特定のPCだけを起こします。\n・Windowsの確認方法: 設定 → ネットワークとインターネット → 使用中のアダプター → ハードウェアプロパティ → 物理アドレス（MAC）\n・コロン区切り（AA:BB:CC:DD:EE:FF）またはハイフン区切り（AA-BB-CC-DD-EE-FF）で入力できます",
				hostMAC),
			sampled("IPアドレス", "192.168.10.20",
				"起こしたいPCのIPアドレスです。起動確認機能（本当に起動したか確認する機能）を使う場合に必要です。起動確認を使わない場合は空のままでOKです。",
				hostIP),
		),
		sectionCard("Wake信号の送り先",
			sampled("ブロードキャスト宛先", "192.168.10.255:9",
				"同じLAN内のPCを起こすとき、Wake信号を送るネットワークアドレスです。\n・空にする → 全体設定の「標準のブロードキャスト宛先」を使う\n・「192.168.10.255:9」→ 192.168.10.x のネットワーク内に送る\n・「255.255.255.255:9」→ LAN全体に送る",
				hostBroadcast),
			sampled("中継先のwol-relay URL", "http://192.168.20.10:8080",
				"起こしたいPCが別のLAN（ルーターの向こう側）にある場合、そのLANで動いているwol-relayのURLを入力します。同じLAN内のPCを起こすだけなら空でOKです。",
				hostRelay),
			sampled("起動を許可するPC名", "desktop, laptop",
				"このPCを起こすことを許可するwol-relayの名前（全体設定の「このPCの名前」）です。カンマ区切りで複数指定できます。空にすると、認証済みのどのwol-relayからでも起こせます。",
				hostAllowedBy),
		),
		sectionCard("起動確認",
			container.NewVBox(checkEnabled, helpText("Wake信号を送った後、PCが本当に起動したかどうかを自動で確認します。オンにするとWake後に応答を待ちます。")),
			sampled("確認方法", "tcp / icmp",
				"起動確認に使う方法を選びます。\n・「tcp」→ 指定したポートへの接続で確認（確実）\n・「icmp」→ pingで確認（Windowsはデフォルトでpingをブロックするため、ファイアウォール設定が別途必要）\n迷ったら「tcp」を選んでください。",
				checkMethod),
			sampled("確認に使うTCPポート", "3389",
				"確認方法が「tcp」のとき、接続を試みるポート番号です。起動しているか確認できれば何でもOKです。\n・リモートデスクトップ（RDP）を使っているなら → 3389\n・SSHを使っているなら → 22\n・wol-relayが動いているなら → wol-relayのポート番号（例: 8080）",
				checkPort),
			sampled("起動確認のタイムアウト", "2m",
				"この時間が経過しても応答がなければ「起動未確認」として扱います。PCの起動に時間がかかる場合は長めに設定してください。「2m」= 2分、「90s」= 90秒。",
				checkTimeout),
			sampled("確認の試行間隔", "3s",
				"起動確認を何秒おきに繰り返すかです。短くすると起動をより素早く検知できますが、負荷がわずかに増えます。通常は「3s」（3秒）のままで十分です。",
				checkInterval),
		),
		container.NewHBox(hostSaveButton, cancelHostEditButton),
	)

	header := container.NewBorder(nil, nil, widget.NewLabel("wol-relay"), nodeLabel)
	repoURL, _ := url.Parse("https://github.com/miutaku/wol-relay")
	intro := container.NewVBox(
		widget.NewLabelWithStyle("Wake on LAN をルーターを越えて安全に届けます", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		helpText("Wake on LAN（WoL）は電源オフのPCをネットワーク経由で起こす仕組みですが、通常はルーターを越えられません。wol-relay は、別のLAN（ルーターの向こう側）にいるPCへもWake on LAN信号を届けるための中継アプリです。"),
		helpText("このPCが受け取ったWake信号を、あらかじめ許可した相手のwol-relayへ転送します。転送は署名付きで行われるため、許可していない相手からの起動を防げます。"),
		widget.NewHyperlink("GitHub: https://github.com/miutaku/wol-relay", repoURL),
	)
	tabs := container.NewAppTabs(
		container.NewTabItem("全体設定", container.NewVScroll(settingsForm)),
		container.NewTabItem("ホスト", container.NewVScroll(container.NewVBox(hostForm, list))),
	)
	content := container.NewBorder(container.NewVBox(header, intro), status, nil, nil, tabs)
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

func sampled(label string, _ string, description string, object fyne.CanvasObject) fyne.CanvasObject {
	return fieldCard(label, description, object)
}

func sectionCard(title string, objects ...fyne.CanvasObject) fyne.CanvasObject {
	items := make([]fyne.CanvasObject, 0, len(objects)*2)
	for i, obj := range objects {
		if i > 0 {
			items = append(items, widget.NewSeparator())
		}
		items = append(items, obj)
	}
	return widget.NewCard(title, "", container.NewVBox(items...))
}

func fieldCard(title string, description string, object fyne.CanvasObject) fyne.CanvasObject {
	label := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	desc := helpText(description)
	return container.NewVBox(label, desc, object)
}

func helpText(value string) *widget.Label {
	label := widget.NewLabel(value)
	label.Wrapping = fyne.TextWrapWord
	return label
}

func openPath(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
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
