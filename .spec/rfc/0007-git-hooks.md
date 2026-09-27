# RFC 0007: Use repository-native Git hooks for local checks (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

使用仓库内 `.githooks/` 和 Git 原生 `core.hooksPath` 管理本地开发检查；不引入 Husky。Hooks 只运行本地检查，不承担 issue 数据远程同步。

## Problem

monorepo 需要稳定的提交前检查和版本一致性检查。pnpm/Turborepo 并不要求把 Git hook 生命周期绑定到 npm install，Husky 会增加一层项目依赖和安装副作用；现有仓库已使用 `.githooks/`。

## Goals

保留 `.githooks/` 作为唯一仓库 hook 源；由明确的开发者设置命令配置 `core.hooksPath`；hooks 可调用 pnpm/Turbo、Go CLI 或官方工具执行本地检查。未安装 hook 时，构建、测试和正常 Git 使用仍可手动执行。push hook 可以验证本地发布条件，但不得运行 IssueGraph 的远程数据同步。

## Non-goals

不改变 Git 的 clone/fetch/pull/push 行为；不让 IssueGraph 代用户提交或推送；不把 hooks 当作安全边界；不在该 RFC 中定义每个 hook 的完整检查集合。

## Design

延续当前 `.githooks/` 与 `git config core.hooksPath .githooks` 机制，并从项目设置入口提供幂等启用/诊断说明。hook 实现保持薄层，调用已有 pnpm、Turbo、Go 或质量门命令，不复制其逻辑，不新增 `.sh` 包装文件。迁移时移除 Beads 名称、`bd` 命令和可触发远程 issue 同步的 hook 段；hook 只消费当前仓库状态并运行本地检查。npm install、pnpm install 不得擅自改写用户 Git 配置。

## Acceptance

设置命令只在用户显式执行后写入当前仓库 `core.hooksPath`，重复运行幂等；hook 从 `.githooks/` 执行且调用既有项目 CLI；hooks 不调用 `bd`、Dolt remote、网络 API 或产品级 Git push/pull；pnpm install 不修改 Git 配置；开发文档说明启用、禁用和手动运行方式。
