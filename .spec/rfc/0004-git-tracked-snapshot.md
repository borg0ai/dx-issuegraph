# RFC 0004: Git 跟踪完整本地 issue 快照 (child of 0001)

**Status:** Approved

**Parent:** [0001](0001-issuegraph-local-git.md)

## Summary

本地 Dolt 继续服务运行时查询和写入。仓库中的 Git 跟踪快照成为 issue 图唯一可携带副本；用户通过 Git 自行提交、推送和恢复文件，IssueGraph 不执行 Git 网络操作。

## Problem

`.beads/issues.jsonl` 当前只是被动导出，导入是 upsert，无法表达删除；`.gitignore` 忽略 `.beads/*`。因此普通 Git 提交不包含可完整恢复的 issue 图，而远程 Dolt 被当作跨克隆同步路径。

## Goals

每次成功写入后生成完整快照；快照表达 issue 字段、依赖、标签、评论和删除。恢复时以快照清单作为全集，使本地多余记录被删除。运行时查询仍使用本地 Dolt，不直接解析快照。IssueGraph 仅写工作区文件，不执行 Git add、commit、fetch、pull 或 push。

## Non-goals

不将 Dolt 数据目录直接放入 Git；不定义产品级远程同步；不让快照充当多写者并发协议；不迁移产品目录（RFC [0006](0006-issuegraph-monorepo.md)）。

## Design

快照写入仓库根 `.issuegraph/issues.jsonl`，同目录 `manifest.json` 列出完整 id 集合和格式版本。Git 忽略规则必须跟踪这两个文件。导出使用原子替换，避免中断时留下半份快照。新增 `issuegraph snapshot restore` 命令：先验证格式、完整性和 manifest，再在单一事务中 upsert 全集并删除快照中不存在的 id；缺少 manifest 或校验失败时不改数据库。现有 `.beads/issues.jsonl` upsert 导入不再作为完整恢复路径。快照生成可由本地写入流程或 Git hook 触发，细节必须保持不依赖远程网络；Git hooks 机制由 RFC [0007](0007-git-hooks.md) 决定。

## Acceptance

创建、更新、删除各一条后，快照和 manifest 精确表达最终集合且不被 ignore；把两文件复制到空工作目录并执行 `issuegraph snapshot restore` 后，源数据一致且已删除 id 不存在；缺少 manifest、损坏文件或校验不符时恢复失败且数据库未变；生成快照期间中断不会留下可被误认为完整的文件；测试证明快照流程没有 Git 网络或 Dolt remote 调用。
