# Pinolo v0.6.0 P2/P3 优化方案

## 背景

基于 v0.5.0 深度架构分析报告（`docs/pinolo-deep-analysis.md`），当前项目已完成 P0/P1 正确性修复和核心检测能力提升。本文档规划 P2（中期改进）和 P3（长期方向）共 8 个优化项的具体实现方案。

---

## P2-1: 新增 `= → <=` 比较操作变异方向

### 现状

`FixMCmpOpU` 当前覆盖：
- `= → >=` ✅
- `< → <=` ✅  
- `> → >=` ✅

**缺失**：`= → <=` — 数学上 `{x: x=5} ⊂ {x: x≤5}`，是有效的 upper 变异。

### 方案

新增变异常量 `FixMCmpOpULE`，专门处理 `= → <=`：

```go
// allmutations.go
FixMCmpOpULE = "FixMCmpOpULE"  // = → <= (upper)
```

**实现文件**：新建 `fixmcmpopule.go`

```go
// addFixMCmpOpULE: = → <=
func (v *MutateVisitor) addFixMCmpOpULE(in ast.Node, flag int) {
    var myOp *opcode.Op
    switch in.(type) {
    case *ast.BinaryOperationExpr:
        myOp = &in.(*ast.BinaryOperationExpr).Op
    case *ast.CompareSubqueryExpr:
        myOp = &in.(*ast.CompareSubqueryExpr).Op
    default:
        return
    }
    if *myOp == opcode.EQ {
        v.addCandidate(FixMCmpOpULE, 1, in, flag)
    }
}
```

**数学验证**：
- `{x | x = 5} ⊂ {x | x ≤ 5}` → original ⊂ mutated → upper ✅

### 影响范围

| 文件 | 修改内容 |
|------|---------|
| `allmutations.go` | 新增 `FixMCmpOpULE` 常量 |
| 新建 `fixmcmpopule.go` | add + do 函数 |
| `mutatevisitor.go` | `miningBinaryOperationExpr` 和 `miningCompareSubqueryExpr` 中调用 |
| `stage2.go` | `ImpoMutate` switch 新增 case |
| PG 同理 | `pg_mutatevisitor.go` + `pg_mutate_functions.go` + `pg_stage2.go` |

### 预期收益

为 `=` 操作符增加一个变异方向，使 `WHERE x = 5` 产生 2 个 upper 变异（`>=` 和 `<=`），提高 bug 触发概率。

---

## P2-2: 实现论文 Algorithm 1 递归变异器

### 现状

v0.5.0 实现了 k=2 组合变异（通过 re-parse 策略），但存在局限：
- 每对组合需要重新 parse + CalCandidates，开销大
- 无法保证 k>2 的递归组合
- 与论文 Algorithm 1 的结构归纳法有本质差距

### 方案：直接 AST 双变异（无需 re-parse）

核心思路：在同一个 AST 上同时应用两个变异，避免 re-parse。

```go
// stage2.go — MutateAll 新增 Phase 2（替代现有 k=2 逻辑）

// 收集所有可变异节点列表
type candidateEntry struct {
    mutationName string
    candidate    *Candidate
    isUpper      bool
}
entries := collectAllCandidates(v)

// Phase 2: 同 AST 双变异
for i := 0; i < len(entries); i++ {
    for j := i + 1; j < len(entries); j++ {
        // 仅组合同方向变异
        if entries[i].isUpper != entries[j].isUpper { continue }
        // 跳过同类变异（避免冲突）
        if entries[i].mutationName == entries[j].mutationName { continue }
        
        // 应用第一个变异
        sql1, err1 := ImpoMutate(root, entries[i].candidate, seed)
        if err1 != nil { continue }
        
        // 重新 parse sql1，在其上应用第二个变异
        v2, err := CalCandidates(sql1)
        if err != nil { continue }
        
        // 在 v2 中找到与 entries[j] 相同类型的变异并应用
        for _, c2 := range v2.Candidates[entries[j].mutationName] {
            sql2, err2 := ImpoMutate(v2.Root, c2, seed)
            if err2 != nil { continue }
            
            result.MutateUnits = append(result.MutateUnits, &MutateUnit{
                Name: entries[i].mutationName + "+" + entries[j].mutationName,
                Sql: sql2,
                IsUpper: entries[i].isUpper,
            })
        }
    }
}
```

### Soundness 证明

若 `mutated1 ⊇ original`（upper1）且 `mutated2 ⊇ original`（upper2），
在 mutated1 的基础上再做 upper2 变异：`mutated12 ⊇ mutated1 ⊇ original` → 传递性成立。

### 配置化

通过 `TaskConfig.MaxCombinationDepth` 控制组合深度：
- `0`（默认）：仅单点变异
- `1`：k=2 组合
- `2`：k=3 组合（搜索空间 O(N³)，仅适合小查询）

### 影响范围

| 文件 | 修改内容 |
|------|---------|
| `stage2.go` | 重写 `MutateAll` Phase 2 逻辑 |
| `pg_stage2.go` | PG 版本同步 |
| `task/task.go` 等 | 新增 `MaxCombinationDepth` 配置传递 |

---

## P2-3: 函数调用内表达式变异

### 现状

`visitFuncCallExpr` 和 `visitFuncCastExpr` 为空实现，导致以下高频 SQL 模式不可变异：
- `WHERE YEAR(date_col) = 2024`
- `WHERE ABS(a - b) > threshold`
- `WHERE CAST(x AS SIGNED) > 0`
- `WHERE UPPER(name) = 'FOO'`
- `WHERE DATE_FORMAT(d, '%Y') = '2024'`

### 方案

在 `visitFuncCallExpr` 中递归遍历函数参数，在 `visitFuncCastExpr` 中递归遍历 CAST 表达式：

```go
func (v *MutateVisitor) visitFuncCallExpr(in *ast.FuncCallExpr, flag int) {
    if in == nil { return }
    // 递归遍历函数参数，寻找可变异子表达式
    for _, arg := range in.Args {
        if exprNode, ok := arg.(ast.ExprNode); ok {
            v.visitExprNode(exprNode, flag)
        }
    }
}

func (v *MutateVisitor) visitFuncCastExpr(in *ast.FuncCastExpr, flag int) {
    if in == nil { return }
    // 递归遍历 CAST 的内部表达式
    if in.Expr != nil {
        v.visitExprNode(in.Expr, flag)
    }
}
```

### 关键验证：`ast_replace.go` 兼容性

`exprReplacer.Leave` 已有 `*ast.FuncCallExpr` 分支（line 115-122），能正确替换函数参数中的子表达式。`*ast.FuncCastExpr` 需要新增：

```go
case *ast.FuncCastExpr:
    cast := in.(*ast.FuncCastExpr)
    if cast.Expr == r.target {
        cast.Expr = r.replacement
        r.replaced = true
    }
```

### PG 版本

PG 的 `FuncCall` 节点通过 `visitNode` 递归已部分覆盖（`visitFuncCall` 遍历 `args`）。需确认 `visitFuncCall` 是否正确递归参数。

### 影响范围

| 文件 | 修改内容 |
|------|---------|
| `mutatevisitor.go` | 实现 `visitFuncCallExpr`、`visitFuncCastExpr` |
| `ast_replace.go` | 新增 `FuncCastExpr` 分支 |
| PG 版本 | 验证 `visitFuncCall` 已覆盖 |

### 预期收益

覆盖 ~15% 的额外 WHERE 谓词，使 `YEAR()`, `ABS()`, `CAST()`, `UPPER()` 等函数内的比较操作可被变异。

---

## P2-4: Task Runner 代码去重

### 现状

4 个 task runner 80% 代码重复：
- `RunTask`（MySQL，~625 行）
- `RunTaskPostgreSQL`（PG，~280 行）
- `RunTaskGaussDB`（GaussDB-M，~200 行）
- `RunTaskGaussDBA`（GaussDB-A，~210 行）

差异点仅在于：
1. Connector 类型
2. Stage1/Stage2 函数
3. 特性开关（go-randgen、GenMode、Skipped）

### 方案：泛型模板方法

```go
// task/runner.go — 通用测试循环

// Stage1Func: Stage1 预处理函数签名
type Stage1Func func(sql string, conn connector.SQLExecutor) interface {
    GetInitSql() string
    GetExecResult() *connector.Result
    GetErr() error
}

// Stage2Func: Stage2 变异函数签名  
type Stage2Func func(sql string, seed int64, conn connector.SQLExecutor) interface {
    GetMutateUnits() []MutateUnitLike
    GetErr() error
}

// RunnerOptions: 差异化配置
type RunnerOptions struct {
    Stage1Fn     Stage1Func
    Stage2Fn     Stage2Func
    HasRandGen   bool
    HasGenMode   bool
    HasSkipped   bool
}

// RunTaskGeneric: 通用任务执行器
func RunTaskGeneric(config *TaskConfig, conn connector.SQLExecutor, 
    logger *logrus.Logger, opts RunnerOptions) (*TaskResult, error) {
    // ... 统一的初始化 → Stage1 → Stage2 → Oracle → Bug Report 流程
}
```

### 具体改造

```go
// 改造后各 runner 简化为：
func RunTask(config *TaskConfig, conn *connector.Connector, logger *logrus.Logger) (*TaskResult, error) {
    return RunTaskGeneric(config, conn, logger, RunnerOptions{
        Stage1Fn:   stage1.InitAndExec,
        Stage2Fn:   stage2.MutateAllAndExec,
        HasRandGen: true,
        HasGenMode: true,
    })
}
```

### 影响范围

| 文件 | 修改内容 |
|------|---------|
| 新建 `task/runner.go` | `RunTaskGeneric` + `RunnerOptions` |
| `task/task.go` | 简化为调用 `RunTaskGeneric` |
| `task/postgresql_task.go` | 简化 |
| `task/gaussdb_task.go` | 简化 |
| `task/gaussdb_a_task.go` | 简化 |

### 风险

- 4 个 runner 的返回值类型不同（`*TaskResult` vs `*TaskResult` 但字段不同）
- Stage1/Stage2 的返回类型不同（MySQL 用 `*stage1.InitResult`，PG 用 `*stage1.PgInitResult`）
- 需要通过接口抽象统一

**建议分两步实施**：
1. 先提取公共循环体到 `runMutationLoop()` 函数
2. 再逐步用泛型替代重复代码

---

## P3-1: Error Oracle（变异后执行错误检测）

### 现状

变异后 SQL 执行出错（`Stage2UnitExecErrNum++`）被直接跳过。但如果原始 SQL 执行成功而变异 SQL 报错，这本身就是 DBMS 的逻辑 bug。

### 方案

在 task runner 的变异循环中，新增 Error Oracle 检查：

```go
// 在 mutateUnit.ExecResult.Err != nil 分支中：
if mutateUnit.ExecResult.Err != nil {
    taskResult.Stage2UnitExecErrNum += 1
    
    // Error Oracle: 原始 SQL 成功但变异 SQL 报错 → 可能是 bug
    // 仅对 upper mutation 有意义（upper 预期结果集更大，不应报错）
    if mutateUnit.IsUpper && originalResult.Err == nil {
        // 验证：重新执行变异 SQL，确认错误可复现
        reExecResult := conn.ExecSQL(mutateUnit.Sql)
        if reExecResult.Err != nil {
            // 记录为 Error Oracle violation
            bugId := taskResult.ImpoBugsNum
            taskResult.ImpoBugsNum += 1
            logger.Info("ERROR ORACLE bug! bugId=", bugId, " sqlId=", dmlSql.Id,
                " mutation=", mutateUnit.Name, " error=", reExecResult.Err)
            // 保存 bug report
        }
    }
    continue
}
```

### 分类

| 场景 | 处理 |
|------|------|
| Upper 变异后报错 | 报告为 Error Oracle bug（结果应更大，不应报错） |
| Lower 变异后报错 | 跳过（lower 缩小结果集，报错可理解为"无结果"的极端） |
| 原始 SQL 也报错 | 跳过（stage1ExecErrNum 已计数） |

### BugReport 扩展

```go
type BugReport struct {
    // ... 现有字段
    IsErrorOracle bool   // 是否为 Error Oracle bug
    ErrorMsg      string // 错误信息（Error Oracle 专用）
}
```

### 影响范围

| 文件 | 修改内容 |
|------|---------|
| `task/task.go` 等 | 4 个 runner 新增 Error Oracle 分支 |
| `task/bugreport.go` | `BugReport` 新增字段 |
| `task/task.go` | `TaskResult` 新增 `ErrorOracleBugsNum` |

---

## P3-2: 覆盖率引导变异

### 现状

当前变异策略是**穷举所有变异点**，不区分优先级。对于复杂查询，可能产生大量低价值变异。

### 方案：基于结果差异的优先级排序

核心思路：**优先变异能产生最大结果集差异的变异点**。

```go
// stage2.go — 新增优先级排序

// MutateAllPrioritized: 按结果差异排序的变异
func MutateAllPrioritized(sql string, seed int64, conn connector.SQLExecutor) *MutateResult {
    result := MutateAll(sql, seed)
    
    // 执行所有变异并收集结果
    MutateAllAndExec(result, conn)
    
    // 按结果差异排序：差异大的优先
    sort.Slice(result.MutateUnits, func(i, j int) bool {
        diffI := resultDiff(originalResult, result.MutateUnits[i].ExecResult)
        diffJ := resultDiff(originalResult, result.MutateUnits[j].ExecResult)
        return diffI > diffJ
    })
    
    return result
}

func resultDiff(a, b *connector.Result) int {
    if a == nil || b == nil { return 0 }
    return abs(len(a.Rows) - len(b.Rows))
}
```

### 进阶：基于历史数据的权重

维护一个 mutation 成功率表（per DBMS），动态调整变异优先级：

```go
// mutation_stats.go
type MutationStats struct {
    Name         string
    TotalRuns    int
    BugsFound    int
    HitRate      float64  // bugs / total
    AvgExecTime  time.Duration
}

// 优先选择 HitRate 高的变异
```

### 影响范围

| 文件 | 修改内容 |
|------|---------|
| `stage2/stage2.go` | 新增 `MutateAllPrioritized` |
| 新建 `stage2/mutation_stats.go` | 变异统计 |
| `task/task.go` 等 | 集成优先级排序 |

---

## P3-3: 聚合函数近似支持

### 现状

论文 Theorem 3.1 明确排除聚合函数（`SUM`, `COUNT`, `AVG` 等），因为聚合不保持包含关系。Stage1 直接移除所有聚合函数和 GROUP BY。

**影响**：论文实验中 6/9 个基线发现但 Pinolo 漏掉的 bug 与聚合相关。

### 方案：条件性聚合近似

**核心洞察**：当聚合列的值域已知时，可以建立条件性近似关系。

#### 3a. COUNT + 条件过滤的近似

```
原始: SELECT COUNT(*) FROM t WHERE p(x)
变异: SELECT COUNT(*) FROM t WHERE 1（upper）

关系: COUNT(WHERE p) ≤ COUNT(WHERE 1) → 有效的 upper approximation
```

**实现**：在 Stage1 中，不删除 `COUNT(*)` + `GROUP BY`，而是保留聚合框架，仅对 WHERE/HAVING 做变异。

```go
// stage1/rmgroup.go — 条件性保留聚合

// 如果 SELECT 列表只有 COUNT(*) 且 GROUP BY 存在
// 且所有 GROUP BY 列在 SELECT 中出现
// → 保留聚合框架，不删除
if isSimpleCountGroupBy(sel) {
    return false  // 不删除聚合
}
```

#### 3b. SUM 的非负列近似

```
原始: SELECT SUM(x) FROM t WHERE p  (x > 0)
变异: SELECT SUM(x) FROM t WHERE 1  (upper)

关系: SUM(x | WHERE p) ≤ SUM(x | WHERE 1) 当 x ≥ 0
```

**限制**：需要知道列值域（通过 schema 信息或 CHECK 约束），实际中难以通用。

#### 3c. 最简方案：聚合查询独立测试路径

新增一个独立的 task runner 路径，专门处理含聚合的查询：

```go
// task/task_aggregate.go

func RunAggregateTask(config *TaskConfig, conn connector.SQLExecutor) {
    // 1. Stage1: 仅移除窗口函数、LEFT/RIGHT JOIN、LIMIT
    //    保留聚合函数和 GROUP BY
    // 2. Stage2: 仅对 WHERE/HAVING 做蕴含变异
    // 3. Oracle: 对 COUNT/SUM/MIN/MAX 结果使用数值比较
    //    COUNT: upper → mutated >= original
    //    SUM (positive col): upper → mutated >= original
    //    MIN: upper → mutated <= original
    //    MAX: upper → mutated >= original
}
```

### Oracle 扩展

```go
// oracle/aggregate_check.go

func CheckAggregate(originalResult, mutatedResult *connector.Result, 
    isUpper bool, aggFunc string) (bool, error) {
    // 提取聚合值
    origVal := parseNumeric(originalResult.Rows[0][0])
    mutVal := parseNumeric(mutatedResult.Rows[0][0])
    
    switch aggFunc {
    case "count", "sum":
        // upper: mutated >= original
        if isUpper { return mutVal >= origVal, nil }
        return mutVal <= origVal, nil
    case "min":
        // upper: mutated <= original (放宽条件可能引入更小值)
        if isUpper { return mutVal <= origVal, nil }
        return mutVal >= origVal, nil
    case "max":
        // upper: mutated >= original (放宽条件可能引入更大值)
        if isUpper { return mutVal >= origVal, nil }
        return mutVal <= origVal, nil
    }
}
```

### 影响范围

| 文件 | 修改内容 |
|------|---------|
| `stage1/` | 新增聚合保留判断逻辑 |
| 新建 `oracle/aggregate_check.go` | 聚合专用 Oracle |
| 新建 `task/task_aggregate.go` | 聚合查询独立测试路径 |
| `task/task.go` | 路由聚合查询到 `RunAggregateTask` |

### 风险

- 聚合近似需要列值域信息，通用性有限
- 多行 GROUP BY 结果的数值比较需要行匹配逻辑
- 建议先实现 COUNT 近似（最简单、最通用），再逐步扩展

---

## P3-4: NULL 安全变异器

### 现状

论文 Theorem 3.1 要求数据库不含 NULL 值。但实际测试数据库中经常有 NULL。

当前处理：
- Connector 将 NULL 映射为字符串 `"NULL"`
- CMP 将 `"NULL"` 作为普通字符串比较
- Stage1 移除 LEFT/RIGHT JOIN（减少 NULL 引入），但不阻止其他 NULL 来源

### 方案

#### 4a. NULL 感知的结果比较

当前 CMP 的 `"NULL"` 字符串比较在语义上是正确的（两个查询都返回 NULL 则匹配）。但存在歧义：字面值字符串 `"NULL"` 与 SQL NULL 不可区分。

```go
// connector/result.go — 使用特殊标记区分 NULL

const NullMarker = "\x00NULL\x00"  // 用不可打印字符包裹

// Connector 扫描结果时：
if data[i] == nil {
    dataS[i] = NullMarker
}
```

#### 4b. 三值逻辑感知的变异正确性分析

**分析每个变异在 NULL 存在时的行为**：

| 变异 | NULL 影响 | 是否仍然 sound |
|------|----------|---------------|
| `= → >=` | `NULL = 5` → NULL, `NULL >= 5` → NULL, 都不满足 WHERE → 行被排除 | ✅ sound |
| `>= → >` | `NULL >= 5` → NULL, `NULL > 5` → NULL | ✅ sound |
| `BETWEEN → >=a` | `NULL BETWEEN` → NULL, `NULL >= a` → NULL | ✅ sound |
| `WHERE p → WHERE 1` | 1 总是 TRUE → NULL 行变为被包含 | ✅ sound（upper 扩大） |
| `WHERE p → WHERE 0` | 0 总是 FALSE → NULL 行仍被排除 | ✅ sound |
| `IS NULL → FALSE` | 原 IS NULL 匹配 NULL 行，变异后 FALSE → 行被排除 | ✅ sound（lower 缩小） |
| `IS NOT NULL → TRUE` | 原 IS NOT NULL 排除 NULL 行，变异后 TRUE → NULL 行被包含 | ✅ sound（upper 扩大） |

**结论**：现有变异器在 NULL 存在时仍然 sound，因为 NULL 在三值逻辑中的行为是一致的（NULL 参与比较总是返回 NULL/UNKNOWN，不满足 WHERE）。

#### 4c. 新增 NULL 相关的变异器

```go
// FixMCoalesceToArgL: COALESCE(x, y) → x (lower: 移除默认值)
//   如果 x 为 NULL，COALESCE 返回 y；变异后返回 NULL → 结果缩小
//   数学: {rows where COALESCE(x,y) is used} ⊇ {rows where x is used}

// FixMIfNullToArgU: IFNULL(x, y) → y (upper: 总是使用默认值)
//   IFNULL 在 x 非 NULL 时返回 x，变异后总返回 y → 结果变化方向不确定
//   ⚠️ 需要额外条件验证

// 最安全的 NULL 变异：
// FixMWhereNotNullU: WHERE p → WHERE (p AND x IS NOT NULL)
//   添加 IS NOT NULL 守卫，结果缩小 → 但这是 lower，不是 upper
//   需要结合上下文判断
```

### 影响范围

| 文件 | 修改内容 |
|------|---------|
| `connector/result.go` | NULL 标记区分 |
| 新建 `mutation/stage2/fixmnullsafe.go` | NULL 安全变异器 |
| `mutation/oracle/oracle.go` | NULL 感知的比较逻辑 |

### 风险评估

- NULL 标记改动影响面广，需要全面回归测试
- 现有变异器已经证明在 NULL 下仍然 sound，优先级可降低
- 建议先实现 NULL 标记区分（4a），再逐步添加新变异器

---

## 实施优先级排序

| 优先级 | 项目 | 工作量 | 预期收益 | 风险 |
|--------|------|--------|---------|------|
| **1** | P2-1: `= → <=` | 低（~2h） | 增加比较操作覆盖 | 极低 |
| **2** | P2-3: 函数调用内变异 | 低（~3h） | 覆盖 ~15% 额外谓词 | 低 |
| **3** | P3-1: Error Oracle | 中（~4h） | 发现"崩溃类" bug | 低 |
| **4** | P2-4: Task Runner 去重 | 中（~6h） | 降低维护成本 | 中（重构风险） |
| **5** | P2-2: 递归变异器优化 | 高（~8h） | 搜索空间扩展 | 中（性能开销） |
| **6** | P3-2: 覆盖率引导 | 高（~8h） | 变异效率提升 | 中 |
| **7** | P3-3: 聚合函数近似 | 高（~12h） | 覆盖论文 6/9 假阴性 | 高（soundness 风险） |
| **8** | P3-4: NULL 安全变异 | 中（~6h） | 放宽论文前提 | 中（改动面广） |

## 验收标准

1. `go build` 编译通过
2. `go test ./mutation/stage2/...` 通过
3. 每个新增变异器有对应的单元测试
4. 集成测试（MySQL TPC-H）验证无回归
5. Release Notes 更新至 v0.6.0
