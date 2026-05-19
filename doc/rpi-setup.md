# Raspberry Pi OSでの設定

すでにRaspberry Pi OS Imagerで設定もろもろ済ませている前提。
基本的にはOverlay RootFSを使う。

## 実行内容

```bash
$ sudo raspi-config # セットアップ時用にオーバークロックするため。再起動をかける。
$ sudo apt update && sudo apt upgrade -y && sudo apt purge nano -y && sudo apt install vim wget -y && sudo apt autoremove -y
$ sudo reboot
```

```shell
$ curl -fsSL https://tailscale.com/install.sh | sh && sudo tailscale up --auth-key=tskey-auth-xxxxxxxxxxx-yyyyyyyyyyyyyyyyyy
$ sudo tailscale up
```

```shell
$ wget https://github.com/miutaku/wol-relay/releases/download/v0.0.12/wol-relay_0.0.12_linux_armv6-headless.deb
$ sudo apt install ./wol-relay_0.0.12_linux_armv6-headless.deb -y
$ sudo vim /etc/wol-relay/wol-relay.conf
$ sudo systemctl enable --now wol-relay 
```

<ここでwake on lanがちゃんとリレーする状態かを確認>

`$ sudo raspi-config` して Overlay RootFSを有効化し、 OverClock を Default に戻し、 /boot も / も両方ro化する。


## セキュリティ対策

### 手動でやる

自分で、以下を手作業でやる方法。

```shell
# オーバーレイ無効化 + /boot RW 化
sudo raspi-config nonint disable_overlayfs
sudo raspi-config nonint disable_bootro
sudo reboot

# アプデ
sudo apt update && sudo apt upgrade -y 

# 再度オーバーレイ有効化 + /boot RO化
sudo raspi-config nonint enable_overlayfs
sudo raspi-config nonint enable_bootro
sudo reboot
```

### 自動化する

別途、 k8s などで別ホストから実行する方法。シェルスクリプトやバッチファイルでやってもいいかもしれない。
最悪、Wake on LAN叩きに行くPCに入れててもいいと思う。なんでもいい。

```shell
apiVersion: batch/v1
kind: CronJob
metadata:
  name: wol-pi-security-update
spec:
  schedule: "0 4 1 * *"              # 毎月1日 04:00
  timeZone: "Asia/Tokyo"             # k8s 1.27+ で native 対応、これ大事
  concurrencyPolicy: Forbid          # 前回のが走ってたら新規は起動しない
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 5          # 失敗ログは多めに残す
  startingDeadlineSeconds: 3600      # k8s が落ちてて起動できなかった時の救済
  jobTemplate:
    spec:
      backoffLimit: 2                # 失敗時のリトライ回数
      activeDeadlineSeconds: 1800    # 30分でタイムアウト(SSHハング対策)
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: ssh-runner
              image: alpine:3.20
              command: ["/bin/sh", "-c"]
              args:
                - |
                  apk add --no-cache openssh-client
                  ssh -i /keys/id_ed25519 \
                      -o UserKnownHostsFile=/keys/known_hosts \
                      -o StrictHostKeyChecking=yes \
                      -o ConnectTimeout=30 \
                      -o ServerAliveInterval=30 \
                      pi-updater@rpi.local \
                      /usr/local/sbin/security-update.sh
              volumeMounts:
                - name: ssh-keys
                  mountPath: /keys
                  readOnly: true
          volumes:
            - name: ssh-keys
              secret:
                secretName: rpi-ssh-key
                defaultMode: 0400
```
