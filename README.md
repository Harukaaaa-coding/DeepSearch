# DeepSearch AI

面向研究与产品发现的本地 AI Agent 工作台。输入研究问题后，系统规划检索、读取网页或公开论文资料、提取证据、检查覆盖缺口，再生成可追溯报告与产品交付草案。

## 功能

- 实时研究计划、搜索与阅读数量、来源可信度和证据覆盖检查。
- 有预算限制的多轮网页搜索、arXiv 索引及可解析的公开 PDF 阅读。
- 带 Evidence ID、原文摘录、来源网址和时间信息的报告与证据链。
- 竞品比较、用户场景、MVP、PRD、数据字典、接口、测试方案、SOP 和工期草案。
- 人机共创：预览修改建议，人工确认后应用，手动保存 PRD。
- 本地知识库：导入文档、访谈、工单、研究笔记及结构化数据。
- 历史研究：新完成的报告和证据自动归档；重新打开历史不调用模型。升级前只有摘要的记录会明确标注。
- Markdown / DOCX 报告导出、PRD DOCX 导出，以及可选的原型图与系统架构图生成。
- 断流、模型输出截断、失败和取消的诊断记录。

## 本地运行

需要 `go.mod` 指定的 Go 1.26 工具链。Node.js 仅用于前端离线测试；前端无需 npm 安装或单独构建。

```powershell
git clone https://github.com/Harukaaaa-coding/DeepSearch.git
cd DeepSearch
go mod download
go run .
```

打开 [http://localhost:8080](http://localhost:8080)。未配置模型密钥时，报告会明确标记为保守证据底稿，不冒充完整的模型分析。

### 配置模型

在新克隆的项目中复制示例，然后只编辑本机的 `config.yaml`：

```powershell
Copy-Item config.example.yaml config.yaml
```

- `report_model` 配置报告供应商、模型名称和该供应商签发的密钥。示例使用项目已实现的 B.AI 适配；具体模型可用性以你的供应商账号为准。
- 代码还支持 `deepseek`、`openai` 报告供应商；没有有效文件配置时可读取 `DEEPSEEK_API_KEY` / `OPENAI_API_KEY` 及对应的 `*_MODEL` 环境变量。
- 图片生成目前使用 B.AI。示例中的 `image_model` 默认全部注释；启用前确认额度和费用。已配置图片模型时，生成 PRD 可能自动提交原型图及系统架构图任务。
- 不配置 `image_model` 且不设置 `BAI_API_KEY`，即可避免图片模型调用。图片超时会标记为结果未知，不自动重复提交；供应商仍可能已经计费。
- `config.yaml` 和 `.env*` 不进入 Git。不要在源码、截图、日志或 issue 中粘贴真实密钥。

本项目不会把历史资料自动全部塞入模型上下文。完整归档与供 Agent 使用的工具摘要分开保存；打开历史仅读取本机数据库。重新研究、共创修改及图片生成属于独立的模型操作，可能消耗额度。

## 数据与配置

| 路径 | 用途 | 是否上传 Git |
| --- | --- | --- |
| `config.example.yaml` | 不含密钥的配置示例 | 是 |
| `config.yaml` | 本机真实模型配置 | 否 |
| `agent.yaml` | 提示词路径、记忆与压缩预算 | 是 |
| `research_policy.yaml` | 研究轮次、来源和时间预算 | 是 |
| `prompts/` | 研究规划与报告提示词 | 是 |
| `data/deepsearch.db` | 知识库、完整研究归档、PRD 版本与诊断 | 否 |
| `data/raw/` | 用户导入的原文件 | 否 |
| `data/generated-images/`、`data/image-jobs/` | 图片及后台任务记录 | 否 |
| `.runtime/` | 本机构建程序和运行日志 | 否 |

首次运行会创建本地数据表。示例行业数据由用户主动加载，并标记为模拟资料，不应作为真实市场或客户证据。人工修改 PRD 后需要点击保存；备份个人研究时应单独备份本地 `data/`，而不是提交到公开仓库。

## 离线验证

```powershell
go test ./... -skip '^TestLive' -count=1
go vet ./...
node --check web/app.js
node --test web/research_stream_test.cjs web/research_history_test.cjs
```

`TestLive*` 是显式启用的公网或付费测试；不要在不清楚费用时设置 `DEEPSEARCH_LIVE_TEST=1`。上面的测试命令会强制排除这些测试。

## 当前边界

- 这是本地开发工作台，尚无完整的多用户认证与权限隔离；不要把服务端口直接暴露到公网。
- 来源数量和可信度分数不等于结论正确率。市场判断、用户需求和自动生成的 PRD 都需要人工核验。
- 扫描版 PDF、登录墙和复杂动态网页可能无法读取；系统应保留缺口，不编造资料。
- 历史中只保存过摘要的旧任务无法还原原始全文。新报告归档失败时会提示手动下载。
- 云端同步、PPT 导出及生产级部署不包含在当前版本中。
