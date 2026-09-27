# RFC 0003: 移除 IssueGraph 远程 issue 数据同步 (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

IssueGraph 只读写本地 issue 数据和本地完整快照。它不连接 Dolt remote、不推拉 issue 历史，也不代替用户执行 Git 网络操作。用户自行使用 Git 管理和推送仓库。

## Problem

当前程序可通过 Dolt remote、`refs/dolt/data`、autopush、federation 等路径传输 issue 数据。可选 local-only 开关不能保证所有写读路径都关闭。产品级远程同步也与“Git 由用户掌控”的边界冲突。

## Goals

所有将 issue 图发送到或取自远程 Dolt、对象存储或专用 issue 数据端点的产品路径都明确失败且不发起网络请求。初始化不配置远程 Dolt。普通 issue 写入只改变本地数据库和 RFC 0004 定义的工作区快照。Git 的 clone/fetch/pull/push 仍由用户的 Git 客户端执行，IssueGraph 不运行这些命令。

## Non-goals

不禁用 Git，也不禁止用户提交、拉取或推送整个仓库；不在此 RFC 移除外部 tracker API（由 RFC [0008](0008-external-trackers-local-only.md) 单独处理）；不卸载 Dolt 或替换本地存储引擎；不定义快照格式（RFC [0004](0004-git-tracked-snapshot.md)）。

## Design

移除或 fail-closed 所有 Dolt remote 添加、推送、拉取、bootstrap、autopush、federation、对象存储同步及 `refs/dolt/data` 发布路径。不能保留可重新开启远程同步的配置开关。错误消息说明 IssueGraph 只保存本地 issue 图，用户可自行用 Git 管理仓库文件。实现覆盖 `modules/cli` 当前入口、`internal/doltremote`、初始化逻辑、后台 hook 和代理文案；具体路径在实施前按 0006 的布局更新。

## Acceptance

每个远程 issue 数据命令和配置路径均以非零状态退出，且测试证明不产生网络或远程驱动调用；初始化后无 Dolt remote 配置；本地 create/update/delete 后查询保持正确，快照由 RFC 0004 覆盖；IssueGraph 进程不执行 `git push`、`git pull`、`git fetch`、`git commit` 或 Dolt remote 操作；用户仍可通过标准 Git 客户端推送普通仓库提交。
