# CLAUDE.md

## 專案定位

3dp-controller：監控多台 3D 印表機（Moonraker/Klipper API），提供 REST API + 網頁 dashboard，並可選擇性地把狀態回報給上層 controller/hub。

- 後端：Go + [Gin](https://gin-gonic.com/)
- 前端：`frontend/`，React + TypeScript + Vite，用 TanStack Query 輪詢後端
- API 文件：swaggo 從 handler 註解產生 Swagger/OpenAPI，前端再用 openapi-generator 產生 axios client

## 目錄結構與職責

- `cmd/3dp-controller/main.go` — 進入點：載入設定、建立每台印表機的 `Monitor`、（選用）建立 controller connector、啟動 web server、處理終端輸入與 SIGINT graceful shutdown。
- `internal/config` — 解析 `config.yaml`（`RawConfig` → `Config`），包含 duration/URL/template 的轉換與 `ControllerFailMode`（`allow_print`/`no_print`）。
- `internal/moonraker` — 核心：
  - `moonraker_api.go`：Moonraker REST API 的無狀態 client（pause/resume/cancel、查詢印表機物件、job 列表等）。
  - `moonraker_monitor.go`：`Monitor`，每台印表機一個，2s 輪詢印表機物件並跑狀態機（`PrinterState`），5s 刷新 job/檔案 metadata。負責「未登記列印」的自動警告/暫停/取消策略。
- `internal/controller` — 選用整合：每 2s 把各 `Monitor` 狀態整理成 `Report` POST 給上層 hub（`internal/controller/api`），並把回應中的控制指令套用回 `Monitor`（`SetRegisteredJobId`/`SetAllowNoRegPrint`）。
- `internal/web` — Gin server：
  - `server.go`：建 Gin engine，掛 CORS/zap middleware，dev 模式開 `/swagger/*any`，`NoRoute` fallback 到 `frontend/dist`（SPA）。**port 目前 hardcode `:8080`**，`config.yaml` 的 `server.bind`/`server.port` 尚未被使用。
  - `api.go`：`/api/v1` 路由，含 swaggo 註解（改 API 要同步改這裡的 `@Summary`/`@Router` 等）。
  - `auth.go`：目前是空 stub，未實作 auth。
- `internal/util` — 共用工具，如 `IsErrNetworkProblem` 判斷是否為網路層錯誤（印表機離線 vs 真正錯誤）。
- `frontend/` — 見下方「前端」。
- `docs/` — `swag init` 產生物（`docs.go`/`swagger.json`/`swagger.yaml`），**gitignore，不要 commit**。

## Data flow

Printer(s) → `internal/moonraker.Monitor`（輪詢 + 狀態機）→ 同時被 `internal/web`（序列化成 API DTO 供前端讀取/操作）與 `internal/controller.Connector`（可選，上報上層 hub 並接收控制指令回寫）使用。前端每 2.5s 打 `GET /api/v1/printers` 顯示 dashboard。

## 重要慣例 / 地雷

- **`docs/` 與 `config.yaml` 都是 gitignore**：本地開發前必須先手動跑一次 `swag init`；`config.yaml` 需自行建立（可能含機敏資訊，例如印表機帳密，切勿提交）。
- **`frontend/src/api/**` 是自動產生的**（由 `docs/swagger.yaml` 經 openapi-generator 產生 TypeScript axios client），**不要手動編輯**。改 API 的流程：改 `internal/web/api.go` 的 swag 註解 → 重新 `swag init` → 在 `frontend/` 跑 `npm run api-gen`。
- swag 的 handler 註解在 `internal/web`（`internal/` 目錄），因此 `swag init` 必須加 `--parseInternal`；main 檔案也已移到 `cmd/3dp-controller/main.go`，需加 `-g cmd/3dp-controller/main.go`。完整指令：
  ```
  swag init -g cmd/3dp-controller/main.go --parseDependency --parseInternal
  ```
- module 名稱為 `3dp-controller`（對應 repo 名稱），所有內部 import 皆為 `3dp-controller/internal/...`。
- 目前沒有 Makefile、CI（無 `.github/workflows`）、Go linter 設定；前端有基本的 ESLint（`.eslintrc.cjs`）。

## 常用指令

```bash
# 後端
swag init -g cmd/3dp-controller/main.go --parseDependency --parseInternal
go build ./cmd/3dp-controller
dev=1 go run ./cmd/3dp-controller   # dev 模式：開發用 logger + /swagger UI
go vet ./...

# 前端（在 frontend/ 目錄下）
npm install
npm run api-gen   # 需先在後端產生 ../docs/swagger.yaml
npm run dev
npm run build
npm run lint
```
