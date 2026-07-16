# 3dp-controller

監控多台 3D 印表機（透過 [Moonraker](https://moonraker.readthedocs.io/)/Klipper API），提供 REST API 與網頁 dashboard，並可選擇性地將各印表機狀態回報給上層 controller/hub。

## 架構概覽

```
Printer backend(s)（Moonraker/Klipper, ...）
      │  poll (2s / 5s)，實作 internal/printer.Printer 介面
      ▼
      ├──────────────────────┐
      ▼                      ▼
internal/web            internal/controller.Connector（可選）
（REST API +              每 2s 回報狀態給上層 hub，
 前端靜態檔服務）           並接收 hub 回傳的控制指令
      │
      ▼
frontend（React + Vite）
每 2.5s 呼叫 GET /api/v1/printers 顯示印表機儀表板
```

`internal/web`、`internal/controller`、`cmd/3dp-controller` 皆只依賴 `internal/printer.Printer` 這個 backend-agnostic 介面（見 `internal/printer/printer.go`），而不直接依賴 `*moonraker.Monitor`，方便未來加入其他印表機廠牌的實作。

## 目錄結構

| 路徑 | 說明 |
| --- | --- |
| `cmd/3dp-controller` | 程式進入點（`main.go`） |
| `internal/config` | 讀取並解析 `config.yaml` |
| `internal/printer` | 印表機 backend 的共用介面（`Printer`、`Thumbnailer`、`RawReporter`）與中立 DTO（`Job`、`ErrorInfo` 等），供各 backend 實作、web/controller 依賴 |
| `internal/moonraker` | Moonraker API client + 印表機狀態輪詢/狀態機，實作 `internal/printer.Printer` |
| `internal/controller` | 選用的上層 controller/hub 回報邏輯 |
| `internal/web` | Gin REST API + 前端靜態檔（SPA）服務 |
| `internal/util` | 共用工具（如網路錯誤判斷） |
| `frontend` | React + TypeScript + Vite 前端，`src/api` 由後端 swagger 規格自動產生 |
| `docs` | `swag init` 產生的 Swagger/OpenAPI 文件（gitignore，需自行產生） |

## 開發環境需求

- Go 1.25+
- Node.js（含 npm）
- [`swag` CLI](https://github.com/swaggo/swag)：`go install github.com/swaggo/swag/cmd/swag@latest`

## 快速開始

1. 在專案根目錄建立 `config.yaml`（此檔案已加入 `.gitignore`，不會被提交）。欄位包含 `server`（目前未實際使用，port 於程式內為 hardcode `:8080`）、`no_pause_duration`、`should_pause_progress`/`should_cancel_progress`、`display_messages`、`controller`（選用的上層 hub）、`printers`（印表機清單，含各自的 `controller_fail_mode`）。

2. 產生後端 Swagger 文件（`internal/web/api.go` 的 handler 註解會被解析）：

   ```bash
   swag init -g cmd/3dp-controller/main.go
   ```

3. 啟動後端（開發模式，會開啟 `/swagger/index.html`）：

   ```bash
   dev=1 go run ./cmd/3dp-controller
   ```

4. 另開一個終端，產生前端的 API client 並啟動前端開發伺服器：

   ```bash
   cd frontend
   npm install
   npm run api-gen   # 需要上一步產生的 ../docs/swagger.yaml
   npm run dev
   ```

## 使用 Docker

```bash
docker build -t 3dp-controller .
docker run -p 8080:8080 -v $(pwd)/config.yaml:/dist/config.yaml 3dp-controller
```

Dockerfile 會依序：編譯後端並產生 Swagger 文件 → 用 Swagger 規格產生前端 API client → 建置前端 → 打包成最終的 Alpine 映像檔（後端以靜態檔方式提供前端頁面）。

## API 文件

開發模式（設定環境變數 `dev=1`）下，啟動後可至 `http://localhost:8080/swagger/index.html` 查看互動式 API 文件。
