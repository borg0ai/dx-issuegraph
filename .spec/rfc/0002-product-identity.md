# RFC 0002: 产品与命令统一为 IssueGraph (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

将产品对外名称统一为 IssueGraph，主命令统一为 `issuegraph`。不提供 `bd` 兼容别名；现有数据库 issue id 保持不变。

## Problem

当前用户界面、二进制、文档、安装器和代理集成仍公开 Beads / `bd`。只改 README 会留下不可用的命令和旧品牌入口。

## Goals

所有受支持的产品入口、安装产物、帮助文本、代理生成内容和 npm 包均使用 IssueGraph / `issuegraph`。新建 workspace 的默认数据目录为 `.issuegraph`，默认 issue 前缀为 `ig`；既有 `.beads` workspace 与 issue 前缀继续可读写，不强制迁移或重写 id。`ISSUEGRAPH_DIR` 成为新环境变量名，`BEADS_DIR` 在迁移期继续工作。Go module import path 可以暂留旧路径，直到独立版本化决策。

## Non-goals

不迁移仓库目录和包布局（RFC [0006](0006-issuegraph-monorepo.md)）；不改变存储或远程同步（RFC [0003](0003-drop-remote-sync.md)）；不搬动既有 `.beads` 数据库或重写已有 issue id；不提供 `bd` 可执行文件、软链接或别名。

## Design

把用户可见名称和命令名作为同一兼容性边界修改：构建和安装生成 `issuegraph`，入口解析、帮助、版本、错误提示、shell completion、文档、npm bin 和代理集成统一调用 `issuegraph`。新 workspace 默认使用 `.issuegraph` 与 `ISSUEGRAPH_DIR`；检测到既有 `.beads` 数据目录或仅设置 `BEADS_DIR` 时继续原地读写，不自动搬库。插件和技能的目录迁移由 0006 负责，本 RFC 只改其产品文案与调用命令。不保留旧 CLI 命令或安装别名。

## Acceptance

`issuegraph version`、`issuegraph --help`、`issuegraph init` 可工作；不存在 `bd` 命令或安装别名；npm 包公开 `issuegraph` bin；新 workspace 使用 `.issuegraph`、`ISSUEGRAPH_DIR` 和 `ig` 前缀；既有 `.beads` workspace、`BEADS_DIR` 和 issue id 可继续读写；受支持文档与集成不再把 Beads / `bd` 作为当前产品名称或命令。
