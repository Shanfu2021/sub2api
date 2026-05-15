# 企业租户与企业管理员能力实现计划

状态：Draft

关联规格：[企业租户与企业管理员能力规格说明](../spec/enterprise-tenant-management.md)

更新时间：2026-05-15

## 1. 实施原则

- 不复用全局 `admin` 作为企业管理员
- 先补后端数据模型与权限边界，再补前端页面
- 所有余额与倍率约束必须走服务端校验
- 第一版聚焦企业拼车主链路，不做企业自助付费
- 尽量复用现有用户余额、备注、优惠倍率、分组授权能力

## 2. 交付范围

本次实现需交付：

- 企业租户数据模型
- 企业成员归属关系
- 企业管理员作用域权限
- 企业额度池与台账
- 企业邀请码
- 企业成员倍率约束
- 平台管理员企业管理后台
- 企业管理员管理后台
- 企业成员简洁门户裁剪
- 测试、文档、迁移说明

## 3. 总体分阶段

### Phase 0：设计落地与命名收敛

目标：

- 固化实体命名、表名、接口路径、角色命名
- 明确哪些能力复用现有实现，哪些需要新增

任务：

- 补齐本规格文档
- 明确 `tenant`、`membership`、`ledger`、`invite_code` 命名
- 明确第一版倍率范围为 `balance`
- 明确企业成员默认禁止自助充值与兑换

产出：

- `docs/spec/enterprise-tenant-management.md`
- `docs/implement-plan/enterprise-tenant-management.md`

### Phase 1：数据库与领域模型

目标：

- 建立企业租户核心数据结构

任务：

- 新增 Ent schema
  - `EnterpriseTenant`
  - `EnterpriseMembership`
  - `EnterpriseInviteCode`
  - `EnterpriseWalletLedger`
  - `EnterpriseTenantGroup`
- 生成对应 migration
- 为关键字段添加唯一索引、状态索引、查询索引
- 增加 repo 层 CRUD 与查询方法

建议触达文件：

- `backend/ent/schema/`
- `backend/migrations/`
- `backend/internal/repository/`

验收：

- 能完成建表、查表、约束校验
- 一个用户无法重复挂靠多个企业

### Phase 2：企业作用域鉴权与服务骨架

目标：

- 建立企业管理员与企业成员的权限边界

任务：

- 新增企业上下文解析逻辑
- 新增企业管理员权限校验器
- 新增企业服务层
  - `tenant_service`
  - `tenant_membership_service`
  - `tenant_invite_service`
  - `tenant_wallet_service`
  - `tenant_pricing_service`
- 平台管理员旁路权限保留
- 企业成员访问企业管理接口时统一拒绝

建议触达文件：

- `backend/internal/service/`
- `backend/internal/server/middleware/`
- `backend/internal/server/routes/`

验收：

- 企业管理员无法访问其他企业成员
- 平台管理员仍可全局访问

### Phase 3：平台管理员后台企业管理

目标：

- 让平台管理员能真正管理企业

任务：

- 新增后台企业列表页
- 新增企业详情页
- 支持创建/编辑企业
- 支持设置企业状态、备注、倍率下限、作用范围
- 支持配置企业允许分组
- 支持给企业增减总额度
- 支持绑定/解绑企业管理员
- 支持查看企业成员与额度台账

建议触达文件：

- `backend/internal/handler/admin/`
- `frontend/src/api/admin/`
- `frontend/src/views/admin/`
- `frontend/src/router/index.ts`

验收：

- 平台管理员能从 UI 完成企业全生命周期管理

### Phase 4：企业邀请码与成员挂靠

目标：

- 让企业成员能通过邀请码自动归属企业

任务：

- 新增企业邀请码管理接口
- 新增企业邀请码创建/禁用/过期/次数限制逻辑
- 扩展注册流程支持企业邀请码
- 扩展补绑定流程支持企业邀请码
- 记录 `joined_via` 与 `joined_source`

建议触达文件：

- `backend/internal/handler/auth_*`
- `backend/internal/service/auth_*`
- `backend/internal/service/tenant_invite_*`
- `frontend/src/views/auth/`
- `frontend/src/api/auth.ts`

验收：

- 新用户通过企业邀请码注册后自动归属企业
- 已有普通用户补绑定企业邀请码后自动归属企业
- 已归属其他企业的用户不能再次绑定

### Phase 5：企业额度池与成员余额分发

目标：

- 打通平台给企业额度，企业给成员余额的完整链路

任务：

- 实现企业总额度与已用额度统计
- 实现企业管理员给成员加余额
- 实现企业管理员给成员减余额
- 与现有用户余额日志联动
- 写入企业额度台账
- 处理并发下的额度扣减一致性

建议触达文件：

- `backend/internal/service/tenant_wallet_service.go`
- `backend/internal/repository/`
- `backend/internal/handler/enterprise/`
- `frontend/src/api/enterprise/`
- `frontend/src/views/enterprise/`

验收：

- 企业管理员不能超额分发
- 回收余额不能导致成员余额为负
- 企业台账与用户余额变化一致

### Phase 6：企业倍率与分组约束

目标：

- 建立企业成员的价格与分组边界

任务：

- 实现企业倍率下限校验
- 将企业成员倍率写入现有 `pricing_discount_factor`
- 第一版只允许 `pricing_scope = balance`
- 企业管理员修改成员倍率时校验：
  - 只能改本企业成员
  - 只能设置不优于企业下限的倍率
- 企业成员分组绑定时校验企业允许分组

建议触达文件：

- `backend/internal/service/tenant_pricing_service.go`
- `backend/internal/service/admin_service.go`
- `backend/internal/service/api_key_auth_cache.go`
- `frontend/src/views/enterprise/`
- `frontend/src/components/admin/user/`

验收：

- 企业管理员不能将成员设为更优倍率
- 企业成员不能绑定企业未授权分组

### Phase 7：企业管理员前端

目标：

- 给企业管理员一套独立、足够轻量的管理界面

任务：

- 新增企业概览页
- 新增企业成员管理页
- 新增成员余额分发弹窗
- 新增成员倍率编辑弹窗
- 新增企业邀请码页
- 新增企业台账页
- 在 `pro.ise.it.com` 中按身份显示企业管理入口

建议触达文件：

- `frontend/src/router/index.ts`
- `frontend/src/components/layout/`
- `frontend/src/views/enterprise/`
- `frontend/src/api/enterprise/`
- `frontend/src/stores/auth.ts`

验收：

- 企业管理员登录 `pro.ise.it.com` 后可见企业管理模块
- 企业成员登录后不可见企业管理模块

### Phase 8：企业成员门户裁剪

目标：

- 让企业成员只看到企业交付所需功能

任务：

- 企业成员隐藏充值入口
- 企业成员隐藏兑换入口
- 企业成员隐藏公告入口
- 企业成员隐藏邀请返利入口
- 企业成员保留：
  - 仪表板
  - API Key
  - 可用分组/订阅
  - 个人资料

建议触达文件：

- `frontend/src/router/index.ts`
- `frontend/src/components/layout/AppSidebar.vue`
- `frontend/src/components/layout/AppHeader.vue`
- `frontend/src/views/user/`

验收：

- 企业成员看不到 ToC 模块
- 不影响平台普通用户与平台管理员现有视图

### Phase 9：平台用户列表增强

目标：

- 提升平台管理员与企业管理员的人群管理效率

任务：

- 平台管理员用户列表增加列：
  - 企业归属
  - 企业角色
  - 企业内备注
- 增加筛选：
  - 按企业筛选
  - 按企业管理员筛选
  - 按企业角色筛选
- 企业管理员成员列表支持搜索邮箱、用户名、企业内备注
- 默认显示 `notes` 与企业归属相关列

建议触达文件：

- `frontend/src/views/admin/UsersView.vue`
- `frontend/src/components/admin/user/`
- `backend/internal/repository/user_repo.go`
- `backend/internal/handler/admin/user_handler.go`

验收：

- 平台管理员可快速筛企业成员
- 企业管理员可通过备注管理大量成员

### Phase 10：测试、迁移、上线

目标：

- 保证功能可上线、可回滚、可排障

任务：

- 增加服务层单测
- 增加权限边界测试
- 增加注册挂靠企业邀请码测试
- 增加额度分发一致性测试
- 增加倍率约束测试
- 编写迁移与回滚说明
- 编写管理员操作文档

建议触达文件：

- `backend/internal/service/*_test.go`
- `backend/internal/handler/*_test.go`
- `docs/`

验收：

- 核心主链路具备自动化测试覆盖
- 上线步骤明确

## 4. 代码任务清单

### Task A：企业基础数据结构

- 新增 Ent schema 与 migration
- 新增 repo 与 service 骨架
- 输出可运行的表结构与基础查询

### Task B：企业作用域权限

- 实现企业管理员身份判定
- 实现企业级资源访问校验
- 将平台管理员保留为旁路超管

### Task C：平台管理员企业后台

- 企业列表、详情、管理员绑定、分组授权、额度调整

### Task D：企业邀请码

- 生成、禁用、次数限制、过期逻辑
- 注册/补绑定接入

### Task E：企业额度分发

- 企业额度池
- 成员加减余额
- 台账一致性

### Task F：企业倍率与成员约束

- 企业倍率下限
- 成员倍率修改
- `balance` 范围生效

### Task G：企业管理前端

- 企业概览
- 成员管理
- 邀请码
- 台账

### Task H：企业成员简洁门户裁剪

- 移除 ToC 入口
- 保留 API 使用主链路

### Task I：列表增强与可运维性

- 企业归属列
- 企业内备注
- 平台筛选与搜索增强

### Task J：测试与发布

- 单测
- 集成测试
- 文档
- 上线说明

## 5. 推荐实施顺序

推荐严格按以下顺序推进：

1. Task A
2. Task B
3. Task C
4. Task D
5. Task E
6. Task F
7. Task G
8. Task H
9. Task I
10. Task J

## 6. 里程碑定义

### Milestone 1

- 企业数据模型完成
- 企业权限框架完成
- 平台管理员可创建和配置企业

### Milestone 2

- 企业邀请码可用
- 企业成员可自动挂靠企业
- 企业管理员可分发余额

### Milestone 3

- 企业管理员前端可用
- 企业成员前端裁剪完成
- 企业倍率约束上线

### Milestone 4

- 测试与运维文档补齐
- 可灰度发布

## 7. 风险控制

- 企业额度台账与用户余额必须同事务或强一致校验
- 企业管理员绝不能命中全局 admin 路由
- 注册接入邀请码时要处理已有企业归属冲突
- 前端菜单裁剪必须与服务端权限校验同时存在，不能只做 UI 隐藏

## 8. 第一版完成判定

满足以下条件即可认定第一版完成：

- 平台管理员可以创建企业、设额度、设倍率、设企业管理员
- 企业管理员可以创建成员、通过邀请码拉成员、给成员分发余额、写备注、设倍率
- 企业成员只能看到简洁门户且不能充值/兑换
- 平台管理员可以按企业筛选并审计主要操作
