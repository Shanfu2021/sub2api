# 企业租户与企业管理员能力规格说明

状态：Draft

更新时间：2026-05-15

## 1. 背景

当前系统已经具备以下基础能力：

- 平台级管理员后台
- 普通用户门户
- `pro.ise.it.com` 简洁门户
- 用户余额、API Key、分组、倍率、优惠码、邀请码等机制
- 用户备注、用户自定义属性、用户分组授权等管理能力

但面向企业拼车场景时，仍缺少一套真正可用的“企业作用域管理”：

- 企业本身不是平台管理员，但需要管理自己的人
- 企业管理员需要能分发额度、管理成员、写备注、筛选成员
- 企业成员不应看到 ToC 的充值、兑换、公告、拉新等内容
- 平台管理员需要从全局视角管理多个企业，并做额度与倍率约束

本规格定义一套“企业租户 + 企业管理员 + 企业成员”的作用域模型，用于支持企业拼车运营。

## 2. 目标

- 支持一个平台管理多个企业租户
- 支持一个企业拥有多个企业管理员
- 支持企业管理员管理本企业成员，但不能越权访问其他企业数据
- 支持平台管理员给企业下发总额度，企业管理员再向成员分发额度
- 支持企业邀请码邀请成员并自动归属企业
- 支持企业成员默认进入简洁门户
- 支持企业倍率约束：企业管理员不能给成员设置优于平台给企业的倍率
- 支持企业成员备注、归属筛选、审计追踪

## 3. 非目标

本期不包含以下内容：

- 企业自助支付、自动开通企业租户
- 企业自定义域名
- 企业多级代理体系
- 企业发票、对账单、结算单
- 企业订阅结转或复杂订阅结算
- 一个用户同时属于多个企业

## 4. 术语

- 平台管理员：现有全局 `admin`，拥有全站权限
- 企业租户：平台下的一个企业实体
- 企业管理员：属于某个企业、拥有企业内管理权限的用户
- 企业成员：属于某个企业的普通使用者
- 企业额度池：平台分配给企业、供企业管理员继续分发的额度余额
- 企业倍率下限：平台为企业设定的最优倍率边界
- 企业邀请码：注册或补绑定时用于挂靠企业的专属邀请码

## 5. 总体方案

### 5.1 角色模型

- 平台管理员：保留现有 `users.role = admin`
- 企业管理员：不是全局 `admin`，通过企业成员关系表记录其 `manager` 身份
- 企业成员：企业关系表中记录为 `member`

### 5.2 门户模型

- `api.ise.it.com`
  - 保留现有标准门户与平台管理员后台
- `pro.ise.it.com`
  - 面向企业成员与企业管理员
  - 企业成员默认只看到简洁门户
  - 企业管理员在简洁门户基础上额外看到“企业管理”入口

### 5.3 权限边界

- 平台管理员可见所有企业、所有用户、所有额度台账
- 企业管理员只能看到自己企业的成员、邀请码、额度台账
- 企业成员只能看到自己的个人数据
- 企业管理员绝不能获得全局 `admin` 路由访问权

## 6. 数据模型

### 6.1 新增表

#### `enterprise_tenants`

用途：定义企业租户。

建议字段：

- `id`
- `name`
- `code`
- `status`：`active` / `disabled`
- `notes`
- `portal_host`：可选，默认仍使用 `pro.ise.it.com`
- `pricing_floor_factor`
- `pricing_scope`：`balance` / `subscription` / `all`
- `balance_quota_total`
- `balance_quota_used`
- `created_by`
- `updated_by`
- `created_at`
- `updated_at`

#### `enterprise_memberships`

用途：定义用户与企业的归属关系。

建议字段：

- `id`
- `tenant_id`
- `user_id`
- `member_role`：`manager` / `member`
- `member_note`
- `joined_via`：`invite_code` / `admin_create` / `manual_bind`
- `joined_source`
- `created_by`
- `created_at`
- `updated_at`

约束：

- 一个 `user_id` 同时只能存在一条有效企业归属

#### `enterprise_invite_codes`

用途：管理企业邀请码。

建议字段：

- `id`
- `tenant_id`
- `code`
- `status`：`active` / `disabled`
- `max_uses`
- `used_count`
- `expires_at`
- `notes`
- `created_by`
- `created_at`
- `updated_at`

#### `enterprise_wallet_ledger`

用途：记录企业额度池变动。

建议字段：

- `id`
- `tenant_id`
- `operator_user_id`
- `target_user_id`
- `direction`：`platform_grant` / `platform_reclaim` / `manager_grant` / `manager_reclaim` / `adjustment`
- `amount`
- `balance_before`
- `balance_after`
- `notes`
- `related_user_balance_log_id`
- `created_at`

#### `enterprise_tenant_groups`

用途：定义企业可使用的分组集合。

建议字段：

- `id`
- `tenant_id`
- `group_id`
- `created_at`

### 6.2 复用现有字段与能力

- `users.notes`
  - 继续作为平台管理员全局备注
- `users.allowed_groups`
  - 可继续作为用户可绑定分组的最终结果
- `pricing_discount_factor`
  - 继续作为用户实际优惠倍率展示与计费字段
- `pricing_discount_scope`
  - 继续用于区分仅余额、仅订阅或全部生效
- 用户自定义属性
  - 可继续用于补充企业字段搜索，但不替代正式企业归属表

## 7. 核心业务规则

### 7.1 企业归属

- 一个用户只能归属一个企业
- 一个企业可有多个企业管理员
- 平台管理员可以手动绑定或解绑用户归属
- 企业邀请码注册成功后，用户自动归属对应企业
- 企业管理员创建成员账号时，成员自动归属该企业

### 7.2 企业额度池

- 平台管理员给企业发放额度时，增加 `balance_quota_total`
- 企业管理员给成员发放余额时，增加 `balance_quota_used`
- 企业管理员回收成员余额时，减少 `balance_quota_used`
- 企业管理员分发前必须校验企业可用额度 `balance_quota_total - balance_quota_used`
- 企业管理员不能透支企业额度池

### 7.3 企业倍率

- 平台管理员为企业设定 `pricing_floor_factor`
- 企业管理员给成员设置倍率时，必须满足：
  - `member_factor >= tenant.pricing_floor_factor`
- 第一版默认只作用于 `balance`
- 如后续扩展到 `subscription` 或 `all`，仍由企业级 `pricing_scope` 控制

示例：

- 企业倍率下限为 `0.4`
- 企业管理员可以给成员设置 `0.4`、`0.5`、`0.75`、`1.0`
- 企业管理员不能给成员设置 `0.3`

### 7.4 分组访问

- 平台管理员可以为企业配置允许使用的分组集合
- 企业成员最终可绑定的分组，应同时满足：
  - 在企业允许分组中
  - 在用户自身允许分组中
- 企业管理员不能把成员放到企业未授权的分组

### 7.5 成员充值与兑换

- 企业成员默认禁止自助充值
- 企业成员默认禁止余额卡兑换
- 企业成员默认禁止查看 ToC 充值、兑换、邀请返利、公告模块
- 企业成员余额主要来自企业管理员发放

### 7.6 审计

以下操作必须记审计：

- 创建/编辑/禁用企业
- 平台向企业增减额度
- 绑定/解绑企业管理员
- 企业管理员增减成员余额
- 企业管理员修改成员倍率
- 企业管理员创建、禁用邀请码
- 企业管理员禁用成员或修改成员备注

## 8. 权限矩阵

| 能力 | 平台管理员 | 企业管理员 | 企业成员 |
| --- | --- | --- | --- |
| 创建企业 | 是 | 否 | 否 |
| 编辑企业设置 | 是 | 否 | 否 |
| 设置企业总额度 | 是 | 否 | 否 |
| 设置企业倍率下限 | 是 | 否 | 否 |
| 配置企业允许分组 | 是 | 否 | 否 |
| 查看所有企业 | 是 | 否 | 否 |
| 查看本企业概览 | 是 | 是 | 否 |
| 创建本企业成员 | 是 | 是 | 否 |
| 绑定成员到企业 | 是 | 仅本企业 | 否 |
| 管理本企业邀请码 | 是 | 是 | 否 |
| 给成员加余额 | 是 | 仅本企业 | 否 |
| 给成员减余额 | 是 | 仅本企业 | 否 |
| 修改成员倍率 | 是 | 仅本企业且受倍率下限约束 | 否 |
| 修改成员备注 | 是 | 仅本企业 | 否 |
| 查看自己的 API 功能 | 是 | 是 | 是 |

## 9. 后端需求

### 9.1 权限与鉴权

- 新增企业作用域鉴权中间件或服务层校验
- 企业管理员访问企业管理接口时，必须校验其 `manager` 身份与 `tenant_id`
- 企业成员访问企业管理接口一律拒绝
- 平台管理员可绕过企业作用域限制

### 9.2 服务层

需新增以下服务或子模块：

- 企业租户服务
- 企业成员关系服务
- 企业邀请码服务
- 企业额度台账服务
- 企业管理员成员余额分发服务
- 企业成员倍率约束服务

### 9.3 与现有用户/余额/倍率系统的关系

- 企业余额分发最终仍调用现有用户余额变更逻辑
- 企业倍率最终仍写入现有用户优惠/折扣字段
- 企业分组访问最终仍同步到现有用户允许分组或绑定校验逻辑

## 10. 前端需求

### 10.1 平台管理员后台

新增模块：

- 企业管理列表
- 企业详情
- 企业管理员管理
- 企业额度管理
- 企业邀请码管理
- 企业成员列表

平台管理员用户列表增强：

- 增加“企业归属”列
- 增加“企业内备注”列
- 支持按企业筛选
- 支持按企业管理员筛选

### 10.2 企业管理员门户

在 `pro.ise.it.com` 增加企业管理入口：

- 企业概览
- 成员管理
- 余额分发记录
- 邀请码管理

### 10.3 企业成员门户

企业成员默认仅保留：

- 仪表板
- API 密钥
- 订阅/分组
- 个人资料

默认隐藏：

- 充值
- 兑换
- 公告
- 邀请返利
- 文档外链弹窗入口

## 11. API 草案

### 11.1 平台管理员接口

- `GET /api/v1/admin/tenants`
- `POST /api/v1/admin/tenants`
- `GET /api/v1/admin/tenants/:id`
- `PUT /api/v1/admin/tenants/:id`
- `POST /api/v1/admin/tenants/:id/quota`
- `GET /api/v1/admin/tenants/:id/ledger`
- `POST /api/v1/admin/tenants/:id/managers`
- `DELETE /api/v1/admin/tenants/:id/managers/:userId`
- `PUT /api/v1/admin/tenants/:id/groups`
- `GET /api/v1/admin/tenants/:id/members`

### 11.2 企业管理员接口

- `GET /api/v1/enterprise/me`
- `GET /api/v1/enterprise/members`
- `POST /api/v1/enterprise/members`
- `PUT /api/v1/enterprise/members/:userId`
- `POST /api/v1/enterprise/members/:userId/balance`
- `POST /api/v1/enterprise/members/:userId/pricing`
- `GET /api/v1/enterprise/invite-codes`
- `POST /api/v1/enterprise/invite-codes`
- `PUT /api/v1/enterprise/invite-codes/:id`
- `GET /api/v1/enterprise/ledger`

### 11.3 注册与绑定接口扩展

- 注册接口接受企业邀请码
- 补绑定接口支持企业邀请码
- 服务端校验企业邀请码有效性、使用次数、归属唯一性

## 12. 验收标准

- 平台管理员可创建企业并设置额度、倍率下限、允许分组
- 企业管理员登录后只能看到本企业成员
- 企业管理员能创建成员、发余额、回收余额、写备注
- 企业管理员发放总额不能超过企业剩余额度
- 企业管理员不能把成员倍率设得优于企业倍率下限
- 企业邀请码能让新用户自动归属到对应企业
- 企业成员默认看不到 ToC 功能入口
- 平台管理员可以按企业筛选成员并查看完整台账

## 13. 已确认的产品决策

- 企业成员禁止自助充值
- 企业倍率第一版仅作用于 `balance`
- 企业管理员可直接创建成员，也支持邀请码邀请
- 一个用户只允许属于一个企业
- 一个企业允许多个企业管理员
- 企业管理员允许回收成员未使用余额，但不能让成员余额为负

## 14. 风险与注意事项

- 不能直接把企业管理员做成全局 `admin`，否则无法隔离数据
- 企业额度池与用户余额变动必须保持一致，避免账不平
- 企业倍率约束必须服务端校验，不能只靠前端限制
- 企业成员门户与普通门户共用前端时，需要严格按角色裁剪菜单与接口
