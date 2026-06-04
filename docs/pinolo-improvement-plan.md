# Pinolo v0.5.0 整改方案

## 背景

基于深度架构分析报告（`docs/pinolo-deep-analysis.md`），对照论文 *Detecting Logical Bugs in DBMS with Approximate Query Synthesis* 的设计思路，发现以下需要整改的问题。

## 整改项

### P0 — 正确性修复（最高优先级）

#### P0-1: BetweenExpr NOT flag 缺失
- **问题**: `miningBetweenExpr` 未检查 `in.Not`，NOT BETWEEN 变异方向错误，产生假阳性
- **修复**: MySQL `mutatevisitor.go` 和 PG `pg_mutatevisitor.go` 添加 `if in.Not { flag ^= 1 }`
- **文件**: `mutation/stage2/mutatevisitor.go`, `mutation/stage2/pg_mutatevisitor.go`

#### P0-2: FixMInNullU 分类 — 经验证为正确，无需修改
- **原分析**: `IN(a,b,c) → IN(a,b,c,NULL)` 标记为 Upper(U=1) 被认为是错误
- **验证结论**: U=1 是正确的。在正向上下文中 cmp=0 通过；在负向上下文（NOT IN）中 Flag 翻转为 lower，结果缩小也通过。改为 U=0 反而会在负向上下文产生假阳性
- **状态**: ✅ 无需修改

#### P0-3: TaskPool 不等待 goroutine 完成
- **问题**: 主循环退出后 goroutine 可能仍在运行，结果不完整
- **修复**: 使用 `sync.WaitGroup` 确保所有已启动的 goroutine 完成
- **文件**: `task/taskpool.go`

### P1 — 检测能力提升

#### P1-1: 实现 k=2 组合变异
- **问题**: 论文 Algorithm 1 的组合变异是 O(2^N)，代码只做单点变异 O(N)
- **修复**: 在 `MutateAll` 中实现 k=2 组合变异
- **文件**: `mutation/stage2/stage2.go`, `mutation/stage2/pg_stage2.go`

#### P1-2: 添加 IS NULL / IS NOT NULL 变异
- **问题**: `visitIsNullExpr` 完全跳过，不产生任何变异
- **修复**: 添加 `FixMIsNullToFalseL`（IS NULL → FALSE, lower）和 `FixMIsNotNullToTrueU`（IS NOT NULL → TRUE, upper）
- **文件**: 新建 `mutation/stage2/fixmisnull.go`，修改 `allmutations.go`, `mutatevisitor.go`, `stage2.go`

#### P1-3: False Positive 检测器移植到 PG/GaussDB
- **问题**: `FalsePositiveDetector` 仅在 MySQL runner 中实现
- **修复**: 移植到 `postgresql_task.go`, `gaussdb_task.go`, `gaussdb_a_task.go`
- **文件**: `task/postgresql_task.go`, `task/gaussdb_task.go`, `task/gaussdb_a_task.go`

#### P1-4: Oracle 错误不终止任务
- **问题**: `oracle.Check()` 错误直接 `return nil, oracleErr` 终止整个任务
- **修复**: 改为记录错误 + continue
- **文件**: `task/task.go`, `task/postgresql_task.go`, `task/gaussdb_task.go`, `task/gaussdb_a_task.go`

### P2 — 覆盖面扩展

#### P2-1: 数值归一化支持科学计数法
- **问题**: `normalizeNumeric()` 不处理 `1e10`、`1.5E-3` 等格式
- **修复**: 扩展归一化逻辑
- **文件**: `connector/result.go`

## 实施顺序

1. P0-1 → P0-2 → P0-3 → 编译验证
2. P1-2 → P1-1 → P1-3 → P1-4 → 编译验证
3. P2-1 → 编译验证
4. 更新 release notes

## 验收标准

- `go build` 编译通过
- `go test ./mutation/stage2/...` 通过
- NOT BETWEEN 变异不再产生假阳性
- FixMInNullL 正确标记为 Lower
- IS NULL 变异可被触发
- k=2 组合变异正常工作
