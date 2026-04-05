# Dev Container + VS Code Remote Tunnel 設計計画

作成日: 2026-04-06

## 概要

本プロジェクトをDev Container化し、VS Code Remote Tunnelを介して別端末から開発できるようにする。
Claude Codeグローバル設定・AWS/SAM認証を引き継ぎつつ、セキュリティリスクを最小化する汎用設計を採用する。

---

## アーキテクチャ

```
[別端末 (ブラウザ or VS Code)]
        │  GitHub認証済みトンネル
        ▼
[VS Code Remote Tunnel]
        │
        ▼
[開発マシン (Windows 11 + Docker Desktop)]
        │  Dev Container attach
        ▼
[Dev Container (Ubuntu)]
    ├── Node.js / Angular CLI
    ├── Go / AWS SAM CLI
    ├── Claude Code CLI
    ├── ~/.claude/  ← ホストからマウント（Claude Codeグローバル設定）
    └── ~/.aws/     ← ホストからマウント（AWS認証情報）
```

**接続フロー:**
1. 開発マシンでVS Code Remote Tunnelを起動（`code tunnel`）
2. GitHub認証でトンネルを確立
3. 別端末からVS Code（またはブラウザ）でトンネルに接続
4. VS Code上でDev Containerにアタッチ（`Dev Containers: Attach to Running Container` or `Reopen in Container`）

---

## ファイル構成

```
.devcontainer/
├── devcontainer.json     # Dev Container本体設定
└── Dockerfile            # カスタムイメージ定義
```

---

## devcontainer.json 設計

```jsonc
{
  "name": "money-management",
  "build": {
    "dockerfile": "Dockerfile",
    "context": ".."
  },

  // ホストからマウントするシークレット類
  "mounts": [
    // Claude Code グローバル設定（書き込みも必要: メモリ保存のため）
    "source=${localEnv:USERPROFILE}/.claude,target=/root/.claude,type=bind,consistency=cached",
    // AWS認証情報（読み取り専用で十分）
    "source=${localEnv:USERPROFILE}/.aws,target=/root/.aws,type=bind,readonly"
  ],

  // コンテナ起動後の設定
  "postCreateCommand": "npm install -g @angular/cli && cd frontend && npm install",

  // VS Code拡張機能（コンテナ内）
  "customizations": {
    "vscode": {
      "extensions": [
        "Angular.ng-template",
        "golang.go",
        "ms-azuretools.vscode-docker",
        "amazonwebservices.aws-toolkit-vscode"
      ],
      "settings": {
        "terminal.integrated.defaultProfile.linux": "bash"
      }
    }
  },

  // ポートフォワーディング（開発サーバー用）
  "forwardPorts": [4200, 3000],
  "portsAttributes": {
    "4200": { "label": "Angular Dev Server" },
    "3000": { "label": "SAM Local API" }
  },

  // 非rootユーザーでの実行（セキュリティ）
  "remoteUser": "vscode"
}
```

---

## Dockerfile 設計

```dockerfile
FROM mcr.microsoft.com/devcontainers/base:ubuntu-22.04

# Node.js (Angular用)
ARG NODE_VERSION=20
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
    && apt-get install -y nodejs

# Go (バックエンド用)
ARG GO_VERSION=1.22
RUN curl -fsSL https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    | tar -C /usr/local -xz
ENV PATH=$PATH:/usr/local/go/bin

# AWS CLI v2
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o /tmp/awscli.zip \
    && unzip /tmp/awscli.zip -d /tmp \
    && /tmp/aws/install \
    && rm -rf /tmp/aws /tmp/awscli.zip

# AWS SAM CLI
RUN pip3 install aws-sam-cli

# Claude Code CLI（npmで導入）
RUN npm install -g @anthropic-ai/claude-code

# クリーンアップ
RUN apt-get clean && rm -rf /var/lib/apt/lists/*
```

---

## VS Code Remote Tunnel の起動方法

### 初回セットアップ（開発マシンで一度だけ）

```bash
# VS Code CLIをインストール済みの場合
code tunnel --accept-server-license-terms

# GitHub認証を求められるので認証する
# トンネル名を設定（例: "my-dev-pc"）
```

### 常時起動（Windowsサービスとして登録）

```bash
code tunnel service install
code tunnel service start
```

### 別端末からの接続

- **VS Code**: `Remote Tunnels: Connect to Tunnel...` → GitHub認証 → マシン選択
- **ブラウザ**: `https://vscode.dev/tunnel/<tunnel-name>`

---

## セキュリティリスクと対策

### リスク分析

| リスク | 深刻度 | 対策 |
|--------|--------|------|
| トンネル経由の不正アクセス | 高 | GitHub認証必須（VS Code Remote Tunnelの仕様）|
| AWS認証情報の漏洩 | 高 | read-only マウント、最小権限IAMロール使用 |
| Claude設定の漏洩 | 中 | マウント対象を`~/.claude`に限定 |
| コンテナエスケープ | 中 | 非rootユーザー実行、不要capabilityを付与しない |
| ポートの意図しない公開 | 中 | `forwardPorts`は開発用ポートのみ、localhostバインド |
| イメージの脆弱性 | 低〜中 | Microsoft公式ベースイメージ使用、定期更新 |

### 対策詳細

#### 1. AWS認証情報の保護
```jsonc
// devcontainer.json
"mounts": [
  // readonly でマウント: コンテナからの書き換えを防ぐ
  "source=${localEnv:USERPROFILE}/.aws,target=/root/.aws,type=bind,readonly"
]
```
- IAMロールはプロジェクトに必要な権限のみ付与する（最小権限の原則）
- `~/.aws/credentials` に長期クレデンシャルを置かず、SSO or 一時クレデンシャルを推奨

#### 2. コンテナの権限制限
```jsonc
// devcontainer.json
"remoteUser": "vscode",    // 非rootユーザー
"runArgs": [
  "--cap-drop=ALL",        // 全capabilityをドロップ
  "--cap-add=NET_BIND_SERVICE",  // 必要なものだけ追加
  "--security-opt=no-new-privileges:true"  // 権限昇格を禁止
]
```

#### 3. ネットワーク分離
```jsonc
// devcontainer.json
// hostネットワークを使わない（デフォルトでOK）
// ポートはlocalhostのみにバインド
"portsAttributes": {
  "4200": {
    "label": "Angular Dev Server",
    "onAutoForward": "silent"  // 自動公開しない
  }
}
```

#### 4. VS Code Remote Tunnelの認証強化
- GitHubアカウントに**2FA（二要素認証）**を必ず設定する
- トンネルを使わない期間はサービスを停止する
  ```bash
  code tunnel service stop
  ```
- 不審なアクセスがあった場合は即座にトンネルを削除
  ```bash
  code tunnel kill
  ```

#### 5. シークレットを環境変数でコンテナに渡す場合
```jsonc
// devcontainer.json
"containerEnv": {
  // 値はホスト環境変数から取得（直書きしない）
  "MY_SECRET": "${localEnv:MY_SECRET}"
}
```

---

## 他プロジェクトへの汎用展開

### 共通テンプレート（`~/.devcontainer-template/`）

```
~/.devcontainer-template/
├── devcontainer.base.json   # マウント設定など共通部分
└── Dockerfile.base          # 共通ツール（AWS CLI, Claude Code等）
```

### プロジェクトごとの差分のみ記述

```jsonc
// 各プロジェクトの .devcontainer/devcontainer.json
{
  "name": "プロジェクト名",
  "build": { "dockerfile": "Dockerfile" },

  // 共通部分（コピー or 参照）
  "mounts": [
    "source=${localEnv:USERPROFILE}/.claude,target=/root/.claude,type=bind",
    "source=${localEnv:USERPROFILE}/.aws,target=/root/.aws,type=bind,readonly"
  ],

  // プロジェクト固有の設定
  "postCreateCommand": "プロジェクト固有のセットアップ",
  "forwardPorts": [/* プロジェクトで使うポート */]
}
```

### 汎用Dockerfileの継承

```dockerfile
# 各プロジェクトの Dockerfile
FROM ghcr.io/<your-org>/devcontainer-base:latest

# プロジェクト固有ツールのみ追加
RUN apt-get install -y <project-specific-tool>
```

---

## 実装タスク

- [ ] `.devcontainer/Dockerfile` 作成
- [ ] `.devcontainer/devcontainer.json` 作成
- [ ] `.gitignore` に Dev Container キャッシュを追加
- [ ] VS Code Remote Tunnel をWindowsサービスとして設定
- [ ] 動作確認（別端末からの接続テスト）
- [ ] セキュリティチェック（マウントのパーミッション確認）

---

## 注意事項

- `~/.claude/` には認証情報・メモリが含まれる。マウントは信頼できる端末間のみ行う
- AWS認証情報は `readonly` マウント必須。書き換えが必要な場合（SSO更新等）はホスト側で実行する
- Dev Container内でDockerを使う場合（Docker-in-Docker）は追加のセキュリティリスクがある。本プロジェクトでは不要なので無効化する
- Windows環境では `${localEnv:USERPROFILE}` を使う。macOS/Linuxでは `${localEnv:HOME}`
