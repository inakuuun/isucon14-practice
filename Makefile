# etc/systemd/system/isuride-go.service の WorkingDirectory / ExecStart に合わせる
APP_DIR  ?= /home/isucon/webapp/go
BIN_NAME ?= isuride
SERVICE  ?= isuride-go.service
GO       ?= go

.PHONY: build restart deploy status log

# Go アプリをビルドして ExecStart のパスにバイナリを配置する
build:
	cd $(APP_DIR) && $(GO) build -o $(BIN_NAME) .

# systemd のサービスを再起動し、起動状態を確認する
restart:
	sudo systemctl daemon-reload
	sudo systemctl restart $(SERVICE)
	sudo systemctl status $(SERVICE) --no-pager

# ビルドしてから再起動する
deploy: build restart

status:
	sudo systemctl status $(SERVICE) --no-pager
