# RFC 0005: 修订 IssueGraph 章程中的本地数据边界 (child of 0001)

**Status:** Implemented

**Parent:** [0001](../0001-issuegraph-local-git.md)

## Summary

更新项目章程和架构文档：IssueGraph 管理本地 issue 图与本地 Git 跟踪快照；仓库级远程传输由用户的 Git 客户端负责，产品不提供 issue 数据远程同步。

## Problem

章程仍把 Dolt remote sync 列为核心职责，代理指引把 `bd dolt push` / `pull` 写成常规路径，并把 JSONL 定义为被动导出。直接删除产品同步会与文档承诺冲突。

## Goals

章程保留“不自造存储引擎”边界，说明 Dolt 只负责本地存储；本仓库 Git 跟踪快照是唯一可携带 issue 副本；IssueGraph 不连接 DoltHub、S3、GCS、federation 或其他 issue 数据远程。同步概念文档说明快照 manifest 与删除语义。代理指引区分产品数据流与用户自行执行的 Git 操作。

## Non-goals

本 RFC 只改文档；不改 Go、JavaScript、配置或快照实现；不删除一般 Git 使用说明，不改变团队提交/推送仓库代码的工作流；不放宽 Dolt 存储边界。

## Design

修改 `engdocs/PROJECT_CHARTER.md`、`AGENTS.md`、`AGENT_INSTRUCTIONS.md`、`engdocs/CLAUDE.md`、`docs/architecture/index.md`、`docs/core-concepts/sync-concepts.md` 及引用旧同步契约的文档。移除“Dolt 提供产品同步”和“`bd dolt push/pull` 是常规路径”的表述；把 JSONL 被动 upsert 说明替换为 RFC 0004 的完整快照和 manifest 语义。涉及仓库会话的 Git 提交/推送要求继续属于维护工作流，不得描述为 IssueGraph 产品功能。

## Acceptance

产品文档不再描述 Dolt remote 或 `bd dolt push/pull` 为产品路径；同步概念文档准确描述本地快照、manifest 删除语义和用户掌控的 Git 传输；章程继续明确 IssueGraph 不自造存储引擎；本 RFC diff 仅含文档文件。
