# RFC 0001: IssueGraph 本地优先产品重组 (Umbrella)

**Status:** Approved

**Type:** Umbrella

## Summary

将 Beads 重组为 IssueGraph：命令与用户可见产品名统一为 `issuegraph`；仓库按 pnpm/Turborepo monorepo 组织；issue 数据留在本地，完整快照由用户通过 Git 管理和传输。IssueGraph 不执行任何 issue 数据远程推送或拉取。

## Problem

当前仓库仍以 Beads、`bd`、Go 根模块、`plugins/beads` 和 Dolt 远程同步为中心。产品身份、仓库结构、本地数据边界和 Git 工作流彼此独立，必须分别决策和验收。

## Goals

以下子 RFC 全部 Approved 后实施：先完成章程边界 [0005](completed/0005-charter-sync-boundary.md)、JS workspace [0006](0006-issuegraph-monorepo.md)、Go module/CLI 输出 [0009](0009-go-module-cli-build.md)、产品资产布局 [0010](0010-issuegraph-asset-layout.md)，再做产品身份 [0002](0002-product-identity.md)、Git hooks [0007](0007-git-hooks.md)、Git 快照 [0004](0004-git-tracked-snapshot.md)、移除 Dolt 远程同步 [0003](0003-drop-remote-sync.md) 和外部 tracker 离线化 [0008](0008-external-trackers-local-only.md)。0003 必须在 0005、0004 后实施；0008 在 0003 后实施。伞关闭时所有子项均为 Implemented。

## Non-goals

不自造存储引擎，不改变既有 issue hash id，不让 IssueGraph 执行 `git add`、`git commit` 或 `git push`。Git remote 是用户的 Git 工作流，不是 IssueGraph 的同步服务。Umbrella 不规定各子项的实现细节。

## Children

| RFC | Concern |
|-----|---------|
| [0011](0011-go-cli-npm-delivery.md) | `@borg0ai/issuegraph` npm installer |
| [0010](0010-issuegraph-asset-layout.md) | IssueGraph 集成资产目录迁移 |
| [0009](0009-go-module-cli-build.md) | 根 Go module 与 CLI 构建布局 |
| [0008](0008-external-trackers-local-only.md) | Disable external tracker network sync |
| [0005](completed/0005-charter-sync-boundary.md) | 修订章程与同步边界 |
| [0006](0006-issuegraph-monorepo.md) | pnpm/Turborepo 与 npm workspace 目录 |
| [0002](0002-product-identity.md) | 产品名与命令名改为 IssueGraph / `issuegraph` |
| [0004](0004-git-tracked-snapshot.md) | Git 跟踪完整本地快照 |
| [0003](0003-drop-remote-sync.md) | 移除 IssueGraph 的远程 issue 数据同步 |
| [0007](0007-git-hooks.md) | 使用仓库自带 Git hooks 管理本地检查 |
| [0009](0009-go-module-cli-build.md) | 根 Go module、CLI 源码位置与二进制构建输出 |
| [0010](0010-issuegraph-asset-layout.md) | 产品集成资产单一来源与目录迁移 |

## Acceptance

仅当所有子项均为 Implemented，`issuegraph` 成为产品名和主命令，JS workspace、Go CLI 构建、产品资产布局分别符合 0006、0009、0010，且 IssueGraph 所有运行时 issue 数据都留在本地时，Umbrella 才能关闭。IssueGraph 不连接远程 issue 存储或 tracker、不推送 issue 数据，也不调用 Git 网络命令。用户仍可用 Git 自行提交和推送仓库快照。关闭后新需求另开 RFC。
