<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">企业管理</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">管理企业租户、额度池、成员、邀请码与台账。</p>
          </div>
          <button class="btn btn-secondary" :disabled="loading" @click="refreshTenants">刷新</button>
        </div>

        <div class="grid gap-6 xl:grid-cols-[360px_minmax(0,1fr)]">
          <div class="space-y-3">
            <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800">
              <div class="mb-3 flex items-center justify-between gap-3">
                <div>
                  <div class="text-sm font-medium text-gray-800 dark:text-dark-100">企业目录</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">共 {{ tenantTotal }} 个企业</div>
                </div>
                <button class="btn btn-primary btn-sm" type="button" @click="resetTenantForm">新建企业</button>
              </div>
              <div class="space-y-2">
                <input v-model="search" class="input" placeholder="搜索名称 / 编码 / 备注" @keyup.enter="applyTenantFilters" />
                <div class="grid grid-cols-[1fr_auto] gap-2">
                  <select v-model="tenantStatusFilter" class="input" @change="applyTenantFilters">
                    <option value="">全部状态</option>
                    <option value="active">启用</option>
                    <option value="disabled">停用</option>
                  </select>
                  <button class="btn btn-secondary" :disabled="loading" @click="applyTenantFilters">查询</button>
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-gray-200 dark:border-dark-700">
              <div v-if="loading" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-300">加载中...</div>
              <div v-else-if="!tenants.length" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-300">
                没有找到企业
              </div>
              <div v-else class="max-h-[calc(100vh-320px)] min-h-[280px] overflow-y-auto">
                <button
                  v-for="item in tenants"
                  :key="item.id"
                  class="w-full border-b border-gray-100 px-4 py-3 text-left last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/40"
                  :class="selectedTenant?.id === item.id ? 'bg-primary-50 dark:bg-primary-900/20' : ''"
                  @click="selectTenant(item)"
                >
                  <div class="flex items-center justify-between gap-3">
                    <div class="min-w-0">
                      <div class="truncate font-medium text-gray-900 dark:text-white">{{ item.name }}</div>
                      <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ item.code }}</div>
                    </div>
                    <span class="badge" :class="item.status === 'active' ? 'badge-success' : 'badge-warning'">
                      {{ item.status }}
                    </span>
                  </div>
                  <div class="mt-2 text-xs text-gray-500 dark:text-dark-300">
                    企业余额 {{ tenantNetBalance(item).toFixed(2) }} / 可用 {{ tenantAvailableBalance(item).toFixed(2) }}
                  </div>
                </button>
              </div>
              <div class="flex items-center justify-between gap-3 border-t border-gray-100 px-3 py-3 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-300">
                <button class="btn btn-secondary btn-sm" :disabled="loading || tenantPage <= 1" @click="changeTenantPage(tenantPage - 1)">上一页</button>
                <span>第 {{ tenantPage }} / {{ tenantPages }} 页</span>
                <button class="btn btn-secondary btn-sm" :disabled="loading || tenantPage >= tenantPages" @click="changeTenantPage(tenantPage + 1)">下一页</button>
              </div>
            </div>
          </div>

          <div class="space-y-4">
            <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
              <div class="mb-3 flex items-center justify-between gap-3">
                <div>
                  <div class="text-sm font-medium text-gray-700 dark:text-dark-200">
                    {{ selectedTenant?.id ? '编辑企业' : '新建企业' }}
                  </div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">
                    {{ selectedTenant?.id ? `当前编辑：${selectedTenant.name}` : '创建后会出现在左侧企业目录' }}
                  </div>
                </div>
                <button v-if="selectedTenant" class="btn btn-secondary btn-sm" @click="resetTenantForm">切换到新建</button>
              </div>
              <div class="grid gap-3 md:grid-cols-2">
                <label class="space-y-1">
                  <span class="text-xs font-medium text-gray-600 dark:text-dark-200">企业名称</span>
                  <input v-model="tenantForm.name" class="input" placeholder="企业名称" />
                </label>
                <label class="space-y-1">
                  <span class="text-xs font-medium text-gray-600 dark:text-dark-200">企业编码</span>
                  <input v-model="tenantForm.code" class="input" placeholder="留空自动生成" :disabled="!!selectedTenant?.id" />
                </label>
                <label class="space-y-1">
                  <span class="text-xs font-medium text-gray-600 dark:text-dark-200">企业兜底底价</span>
                  <input
                    v-model="tenantForm.pricing_floor_factor"
                    class="input"
                    type="number"
                    min="0.01"
                    step="0.01"
                    placeholder="未配置分组底价时使用"
                  />
                </label>
                <label class="space-y-1">
                  <span class="text-xs font-medium text-gray-600 dark:text-dark-200">成员默认售价</span>
                  <input
                    v-model="tenantForm.member_default_pricing_factor"
                    class="input"
                    type="number"
                    min="0"
                    step="0.01"
                    placeholder="0 表示沿用企业底价"
                  />
                </label>
                <label class="space-y-1">
                  <span class="text-xs font-medium text-gray-600 dark:text-dark-200">企业总并发</span>
                  <input
                    v-model="tenantForm.concurrency"
                    class="input"
                    type="number"
                    min="0"
                    step="1"
                    placeholder="0 不限制"
                  />
                </label>
                <label class="space-y-1">
                  <span class="text-xs font-medium text-gray-600 dark:text-dark-200">计费范围</span>
                  <select v-model="tenantForm.pricing_scope" class="input">
                    <option value="balance">仅余额</option>
                  </select>
                </label>
                <label class="space-y-1">
                  <span class="text-xs font-medium text-gray-600 dark:text-dark-200">企业授信额度</span>
                  <input
                    v-model="tenantForm.balance_overdraft_limit"
                    class="input"
                    type="number"
                    min="0"
                    step="0.01"
                    placeholder="例如 10 / 50 / 100"
                  />
                </label>
                <label class="space-y-1">
                  <span class="text-xs font-medium text-gray-600 dark:text-dark-200">门户域名</span>
                  <input v-model="tenantForm.portal_host" class="input" placeholder="可留空" />
                </label>
                <label class="space-y-1">
                  <span class="text-xs font-medium text-gray-600 dark:text-dark-200">企业状态</span>
                  <select v-model="tenantForm.status" class="input">
                    <option value="active">启用</option>
                    <option value="disabled">停用</option>
                  </select>
                </label>
              </div>
              <textarea v-model="tenantForm.notes" class="input mt-3 min-h-[76px]" placeholder="备注"></textarea>
              <p class="mt-2 text-xs text-gray-500 dark:text-dark-300">
                授信额度只用于企业总账可透支，企业管理员给成员分发额度不消耗企业总账；成员真实调用后才按企业底价消耗企业额度。
              </p>
              <div class="mt-3 rounded-xl border border-gray-200 p-3 dark:border-dark-700">
                <div class="mb-2 flex items-center justify-between gap-2">
                  <div class="text-xs font-medium text-gray-600 dark:text-dark-200">企业可用分组</div>
                  <button class="btn btn-secondary btn-sm" type="button" @click="loadGroups">刷新分组</button>
                </div>
                <div class="grid max-h-56 gap-2 overflow-y-auto pr-1 md:grid-cols-2">
                  <label
                    v-for="group in groupOptions"
                    :key="group.id"
                    class="flex cursor-pointer items-start gap-2 rounded-lg border border-gray-100 p-2 text-xs hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/40"
                  >
                    <input v-model="tenantForm.allowed_group_ids" class="mt-1" type="checkbox" :value="group.id" />
                    <span class="min-w-0 flex-1">
                      <span class="block truncate font-medium text-gray-800 dark:text-dark-100">
                        #{{ group.id }} {{ group.name }}
                      </span>
                      <span class="mt-1 block text-gray-500 dark:text-dark-300">
                        {{ group.platform }} · {{ group.subscription_type === 'subscription' ? '订阅' : '余额' }} · {{ group.is_exclusive ? '专属' : '公开' }}
                      </span>
                      <span v-if="tenantForm.allowed_group_ids.includes(group.id)" class="mt-2 flex items-center gap-2">
                        <span class="shrink-0 text-gray-500 dark:text-dark-300">企业底价</span>
                        <input
                          v-model.number="tenantForm.group_rates[group.id]"
                          class="input h-8 min-w-0 flex-1 text-xs"
                          type="number"
                          min="0.01"
                          step="0.001"
                          :placeholder="`默认 ${group.rate_multiplier || 1}`"
                          @click.stop
                        />
                      </span>
                      <span v-if="tenantForm.allowed_group_ids.includes(group.id)" class="mt-2 flex items-center gap-2">
                        <span class="shrink-0 text-gray-500 dark:text-dark-300">成员默认</span>
                        <input
                          v-model.number="tenantForm.member_group_rates[group.id]"
                          class="input h-8 min-w-0 flex-1 text-xs"
                          type="number"
                          min="0.01"
                          step="0.001"
                          :placeholder="`默认 ${tenantMemberDefaultRate(group.id).toFixed(3)}`"
                          @click.stop
                        />
                      </span>
                    </span>
                  </label>
                  <div v-if="!groupOptions.length" class="text-xs text-gray-500 dark:text-dark-300">暂无可选分组</div>
                </div>
                <div v-if="selectedGroupLabels.length || missingGroupIDs.length" class="mt-3 flex flex-wrap gap-2">
                  <span
                    v-for="group in selectedGroupLabels"
                    :key="group.id"
                    class="rounded-full bg-primary-50 px-2 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-200"
                  >
                    #{{ group.id }} {{ group.name }}
                  </span>
                  <span
                    v-for="id in missingGroupIDs"
                    :key="id"
                    class="rounded-full bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-200"
                  >
                    未加载分组 #{{ id }}
                  </span>
                </div>
                <p class="mt-2 text-xs text-gray-500 dark:text-dark-300">
                  企业底价是平台向企业计费的成本；成员默认是企业成员未单独设置时的售价。企业成员可直接使用这里选中的分组，包括专属分组；未选择时只允许公开分组。
                </p>
              </div>
              <div class="mt-3 flex flex-wrap gap-2">
                <button class="btn btn-primary" :disabled="submitting" @click="submitTenant">
                  {{ selectedTenant?.id ? '保存企业' : '创建企业' }}
                </button>
                <button class="btn btn-secondary" type="button" @click="resetTenantForm">清空</button>
              </div>
            </div>

            <div v-if="selectedTenant" class="space-y-4">
            <div class="grid gap-4 md:grid-cols-5">
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">企业总额度</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ selectedTenant.balance_quota_total.toFixed(2) }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">平台已消耗</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ selectedTenant.balance_quota_spent.toFixed(2) }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">可用含授信</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ tenantAvailableBalance(selectedTenant).toFixed(2) }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">已分发成员额度</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ selectedTenant.balance_quota_used.toFixed(2) }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">授信额度</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ selectedTenant.balance_overdraft_limit.toFixed(2) }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">企业总并发</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ selectedTenant.concurrency || '不限' }}</div>
              </div>
            </div>

            <div class="grid gap-4 lg:grid-cols-2">
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 text-sm font-medium text-gray-700 dark:text-dark-200">调整企业额度池</div>
                <div class="grid grid-cols-[1fr_120px] gap-2">
                  <input v-model="quotaForm.amount" class="input" type="number" min="0.01" step="0.01" placeholder="额度" />
                  <select v-model="quotaForm.direction" class="input">
                    <option value="platform_grant">增加</option>
                    <option value="platform_reclaim">回收</option>
                  </select>
                </div>
                <textarea v-model="quotaForm.notes" class="input mt-2 min-h-[72px]" placeholder="备注"></textarea>
                <button class="btn btn-primary mt-3" :disabled="submitting" @click="submitQuota">提交额度调整</button>
              </div>

              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 text-sm font-medium text-gray-700 dark:text-dark-200">绑定现有用户到企业</div>
                <div class="space-y-2">
                  <label class="block space-y-1">
                    <span class="text-xs font-medium text-gray-600 dark:text-dark-200">查找用户</span>
                    <div class="grid grid-cols-[1fr_auto] gap-2">
                      <input
                        v-model="bindUserSearch"
                        class="input"
                        placeholder="输入邮箱 / 用户名搜索"
                        @keyup.enter="searchBindUsers"
                      />
                      <button class="btn btn-secondary" type="button" :disabled="bindUserSearching" @click="searchBindUsers">
                        查询
                      </button>
                    </div>
                  </label>
                  <div
                    v-if="selectedBindUser"
                    class="rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-xs text-primary-800 dark:border-primary-800/50 dark:bg-primary-900/20 dark:text-primary-100"
                  >
                    已选：#{{ selectedBindUser.id }} {{ selectedBindUser.email }}
                    <span v-if="selectedBindUser.username"> / {{ selectedBindUser.username }}</span>
                  </div>
                  <div v-if="bindUserResults.length" class="max-h-52 overflow-y-auto rounded-lg border border-gray-100 dark:border-dark-700">
                    <button
                      v-for="user in bindUserResults"
                      :key="user.id"
                      type="button"
                      class="w-full border-b border-gray-100 px-3 py-2 text-left text-xs last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/40"
                      :class="bindForm.user_id === user.id ? 'bg-primary-50 dark:bg-primary-900/20' : ''"
                      @click="selectBindUser(user)"
                    >
                      <div class="font-medium text-gray-900 dark:text-white">#{{ user.id }} {{ user.email }}</div>
                      <div class="mt-1 text-gray-500 dark:text-dark-300">
                        用户名：{{ user.username || '-' }}
                        <span v-if="user.enterprise"> · 当前企业：{{ user.enterprise.tenant_name }}</span>
                      </div>
                    </button>
                  </div>
                  <p v-else-if="bindUserSearched" class="text-xs text-gray-500 dark:text-dark-300">没有找到匹配用户</p>
                  <div class="grid grid-cols-2 gap-2">
                    <select v-model="bindForm.member_role" class="input">
                      <option value="member">成员</option>
                      <option value="manager">管理员</option>
                    </select>
                    <input
                      v-model="bindForm.pricing_factor"
                      class="input"
                      type="number"
                      min="0"
                      step="0.01"
                      placeholder="留空/0 使用企业默认售价"
                    />
                  </div>
                  <div v-if="selectedGroupLabels.length" class="rounded-lg border border-gray-100 p-2 dark:border-dark-700">
                    <div class="mb-2 text-xs font-medium text-gray-600 dark:text-dark-200">成员分组倍率</div>
                    <div class="grid gap-2 sm:grid-cols-2">
                      <label v-for="group in selectedGroupLabels" :key="group.id" class="text-xs text-gray-500 dark:text-dark-300">
                        <span class="mb-1 block truncate">#{{ group.id }} {{ group.name }}，默认 {{ tenantMemberDefaultRate(group.id).toFixed(3) }}</span>
                        <input
                          v-model.number="bindForm.group_rates[group.id]"
                          class="input h-9 text-xs"
                          type="number"
                          min="0.01"
                          step="0.001"
                          :placeholder="`默认 ${tenantMemberDefaultRate(group.id).toFixed(3)}`"
                        />
                      </label>
                    </div>
                  </div>
                  <textarea v-model="bindForm.member_note" class="input min-h-[72px]" placeholder="成员备注"></textarea>
                </div>
                <button class="btn btn-primary mt-3" :disabled="submitting" @click="submitBindMember">绑定用户</button>
              </div>
            </div>

            <div class="grid gap-4 xl:grid-cols-2">
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
                  <div class="text-sm font-medium text-gray-700 dark:text-dark-200">成员列表</div>
                  <div class="flex flex-wrap items-center gap-2">
                    <input
                      v-model="memberSearch"
                      class="input h-9 w-full text-xs sm:w-56"
                      placeholder="搜索邮箱 / 用户名 / 备注"
                      @keyup.enter="applyMemberFilters"
                    />
                    <button class="btn btn-secondary btn-sm" @click="applyMemberFilters">查询</button>
                    <button class="btn btn-secondary btn-sm" @click="loadMembers">刷新</button>
                  </div>
                </div>
                <div class="overflow-x-auto">
                  <table class="min-w-full text-sm">
                    <thead class="text-left text-gray-500 dark:text-dark-300">
                      <tr>
                        <th class="py-2">用户</th>
                        <th class="py-2">角色</th>
                        <th class="py-2">默认倍率</th>
                        <th class="py-2">分组倍率</th>
                        <th class="py-2">并发</th>
                        <th class="py-2">余额</th>
                        <th class="py-2">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-if="!members.length">
                        <td colspan="7" class="border-t border-gray-100 py-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-300">
                          没有找到成员
                        </td>
                      </tr>
                      <tr v-for="member in members" :key="member.id" class="border-t border-gray-100 dark:border-dark-700">
                        <td class="py-2">
                          <div class="font-medium text-gray-900 dark:text-white">{{ member.user_email }}</div>
                          <div class="text-xs text-gray-500 dark:text-dark-300">
                            {{ member.user_username || '-' }} · {{ member.member_note || '-' }}
                          </div>
                        </td>
                        <td class="py-2">{{ member.member_role }}</td>
                        <td class="py-2">
                          <span v-if="Number(member.pricing_factor) > 0">{{ Number(member.pricing_factor).toFixed(3) }}x</span>
                          <span v-else class="text-xs text-gray-400">继承企业默认</span>
                        </td>
                        <td class="py-2">
                          <div v-if="member.group_rates && Object.keys(member.group_rates).length" class="flex flex-wrap gap-1">
                            <span
                              v-for="(rate, groupID) in member.group_rates"
                              :key="groupID"
                              class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-700 dark:bg-dark-700 dark:text-dark-200"
                            >
                              {{ groupLabel(Number(groupID)) }} {{ Number(rate).toFixed(3) }}x
                            </span>
                          </div>
                          <span v-else class="text-xs text-gray-400">默认企业成员售价</span>
                        </td>
                        <td class="py-2">{{ member.user_concurrency || '不限' }}</td>
                        <td class="py-2">{{ member.user_balance.toFixed(2) }}</td>
                        <td class="py-2">
                          <button class="btn btn-secondary btn-sm" @click="removeMember(member)">移除</button>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div class="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-gray-100 pt-3 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-300">
                  <span>共 {{ memberTotal }} 个成员，第 {{ memberPage }} / {{ memberPages }} 页</span>
                  <div class="flex items-center gap-2">
                    <button class="btn btn-secondary btn-sm" :disabled="memberPage <= 1" @click="changeMemberPage(memberPage - 1)">上一页</button>
                    <button class="btn btn-secondary btn-sm" :disabled="memberPage >= memberPages" @click="changeMemberPage(memberPage + 1)">下一页</button>
                  </div>
                </div>
              </div>

              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 flex items-center justify-between">
                  <div class="text-sm font-medium text-gray-700 dark:text-dark-200">邀请码</div>
                  <button class="btn btn-secondary btn-sm" @click="loadInviteCodes">刷新</button>
                </div>
                <div class="mb-3 grid grid-cols-[1fr_110px] gap-2">
                  <input v-model="inviteForm.code" class="input" placeholder="留空自动生成" />
                  <input v-model="inviteForm.max_uses" class="input" type="number" min="0" placeholder="次数" />
                </div>
                <input v-model="inviteForm.expires_at" class="input mb-2" type="datetime-local" />
                <textarea v-model="inviteForm.notes" class="input min-h-[72px]" placeholder="备注"></textarea>
                <button class="btn btn-primary mt-3" :disabled="submitting" @click="submitInvite">创建邀请码</button>
                <div class="mt-4 space-y-2">
                  <div v-for="invite in inviteCodes" :key="invite.id" class="rounded-lg border border-gray-100 px-3 py-2 dark:border-dark-700">
                    <div class="flex items-center justify-between gap-3">
                      <code class="text-sm font-semibold text-gray-900 dark:text-white">{{ invite.code }}</code>
                      <div class="flex items-center gap-2">
                        <span class="badge" :class="invite.status === 'active' ? 'badge-success' : 'badge-warning'">{{ invite.status }}</span>
                        <span class="text-xs text-gray-500 dark:text-dark-300">{{ invite.used_count }}/{{ invite.max_uses || '∞' }}</span>
                      </div>
                    </div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ invite.notes || '无备注' }}</div>
                    <div class="mt-2">
                      <button class="btn btn-secondary btn-sm" @click="toggleInvite(invite)">
                        {{ invite.status === 'active' ? '停用' : '启用' }}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
              <div class="mb-3 flex items-center justify-between">
                <div class="text-sm font-medium text-gray-700 dark:text-dark-200">额度台账</div>
                <button class="btn btn-secondary btn-sm" @click="loadLedger">刷新</button>
              </div>
              <div class="overflow-x-auto">
                <table class="min-w-full text-sm">
                  <thead class="text-left text-gray-500 dark:text-dark-300">
                    <tr>
                      <th class="py-2">时间</th>
                      <th class="py-2">方向</th>
                      <th class="py-2">金额</th>
                      <th class="py-2">前后</th>
                      <th class="py-2">备注</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="entry in ledger" :key="entry.id" class="border-t border-gray-100 dark:border-dark-700">
                      <td class="py-2 text-xs">{{ formatDate(entry.created_at) }}</td>
                      <td class="py-2">{{ entry.direction }}</td>
                      <td class="py-2">{{ entry.amount.toFixed(2) }}</td>
                      <td class="py-2">{{ entry.balance_before.toFixed(2) }} -> {{ entry.balance_after.toFixed(2) }}</td>
                      <td class="py-2">{{ entry.notes || '-' }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
            </div>
            <div v-else class="rounded-xl border border-dashed border-gray-300 bg-white p-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300">
              选择左侧企业后，可管理额度、成员、邀请码和台账。
            </div>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import enterpriseAdminAPI from '@/api/admin/enterprise'
import groupsAPI from '@/api/admin/groups'
import usersAPI from '@/api/admin/users'
import type { AdminGroup, AdminUser, EnterpriseInviteCode, EnterpriseLedgerEntry, EnterpriseMembership, EnterpriseTenant } from '@/types'
import { useAppStore } from '@/stores'

const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const search = ref('')
const tenantStatusFilter = ref('')
const tenantPage = ref(1)
const tenantPageSize = 20
const tenantTotal = ref(0)
const tenants = ref<EnterpriseTenant[]>([])
const selectedTenant = ref<EnterpriseTenant | null>(null)
const members = ref<EnterpriseMembership[]>([])
const memberSearch = ref('')
const memberPage = ref(1)
const memberPageSize = 20
const memberTotal = ref(0)
const inviteCodes = ref<EnterpriseInviteCode[]>([])
const ledger = ref<EnterpriseLedgerEntry[]>([])
const groups = ref<AdminGroup[]>([])
const bindUserSearch = ref('')
const bindUserSearching = ref(false)
const bindUserSearched = ref(false)
const bindUserResults = ref<AdminUser[]>([])
const selectedBindUser = ref<AdminUser | null>(null)

const tenantForm = reactive({
  name: '',
  code: '',
  status: 'active',
  notes: '',
  portal_host: '',
  pricing_floor_factor: 1,
  member_default_pricing_factor: 0,
  pricing_scope: 'balance',
  concurrency: 0,
  balance_overdraft_limit: 0,
  allowed_group_ids: [] as number[],
  group_rates: {} as Record<number, number | undefined>,
  member_group_rates: {} as Record<number, number | undefined>,
})

const quotaForm = reactive({
  amount: 0,
  direction: 'platform_grant',
  notes: '',
})

const bindForm = reactive({
  user_id: 0,
  member_role: 'member',
  member_note: '',
  pricing_factor: 0,
  group_rates: {} as Record<number, number | undefined>,
})

const inviteForm = reactive({
  code: '',
  max_uses: 0,
  expires_at: '',
  notes: '',
})

function showSuccess(message: string) {
  appStore.showSuccess(message)
}

function showError(error: unknown) {
  const apiError = error as {
    detail?: string
    message?: string
    error?: string
    reason?: string
    code?: string | number
  } | null
  const detail = typeof apiError?.detail === 'string' ? apiError.detail : ''
  const messageText = typeof apiError?.message === 'string' ? apiError.message : ''
  const errorText = typeof apiError?.error === 'string' ? apiError.error : ''
  const reasonText = typeof apiError?.reason === 'string' ? apiError.reason : ''
  const codeText = apiError?.code ? String(apiError.code) : ''
  const message = detail || messageText || errorText || reasonText || (error instanceof Error ? error.message : '') || codeText || '操作失败'
  appStore.showError(message)
}

const groupOptions = computed(() =>
  [...groups.value].sort((a, b) => {
    if (a.platform !== b.platform) return a.platform.localeCompare(b.platform)
    return a.id - b.id
  })
)

const selectedGroupLabels = computed(() => {
  const byID = new Map(groups.value.map((group) => [group.id, group]))
  return tenantForm.allowed_group_ids
    .map((id) => byID.get(id))
    .filter((group): group is AdminGroup => !!group)
})

const missingGroupIDs = computed(() => {
  const known = new Set(groups.value.map((group) => group.id))
  return tenantForm.allowed_group_ids.filter((id) => !known.has(id))
})

const tenantPages = computed(() => Math.max(1, Math.ceil(tenantTotal.value / tenantPageSize)))
const memberPages = computed(() => Math.max(1, Math.ceil(memberTotal.value / memberPageSize)))

function tenantNetBalance(item: EnterpriseTenant): number {
  return Number(item.balance_quota_total || 0) - Number(item.balance_quota_spent || 0)
}

function tenantAvailableBalance(item: EnterpriseTenant): number {
  return tenantNetBalance(item) + Number(item.balance_overdraft_limit || 0)
}

function groupLabel(groupID: number): string {
  const group = groups.value.find((item) => item.id === groupID)
  return group ? `#${group.id} ${group.name}` : `#${groupID}`
}

function resetTenantForm() {
  tenantForm.name = ''
  tenantForm.code = ''
  tenantForm.status = 'active'
  tenantForm.notes = ''
  tenantForm.portal_host = ''
  tenantForm.pricing_floor_factor = 1
  tenantForm.member_default_pricing_factor = 0
  tenantForm.pricing_scope = 'balance'
  tenantForm.concurrency = 0
  tenantForm.balance_overdraft_limit = 0
  tenantForm.allowed_group_ids = []
  tenantForm.group_rates = {}
  tenantForm.member_group_rates = {}
  selectedTenant.value = null
}

function fillTenantForm(item: EnterpriseTenant) {
  tenantForm.name = item.name
  tenantForm.code = item.code
  tenantForm.status = item.status
  tenantForm.notes = item.notes || ''
  tenantForm.portal_host = item.portal_host || ''
  tenantForm.pricing_floor_factor = item.pricing_floor_factor
  tenantForm.member_default_pricing_factor = item.member_default_pricing_factor || 0
  tenantForm.pricing_scope = item.pricing_scope || 'balance'
  tenantForm.concurrency = item.concurrency || 0
  tenantForm.balance_overdraft_limit = item.balance_overdraft_limit || 0
  tenantForm.allowed_group_ids = [...(item.allowed_group_ids || [])]
  tenantForm.group_rates = { ...(item.group_rates || {}) }
  tenantForm.member_group_rates = { ...(item.member_group_rates || {}) }
}

function buildGroupRatesPayload(groupIDs: number[], rates: Record<number, number | undefined>): Record<number, number> {
  const payload: Record<number, number> = {}
  for (const groupID of groupIDs) {
    const value = Number(rates[groupID])
    if (Number.isFinite(value) && value > 0) {
      payload[groupID] = value
    }
  }
  return payload
}

function buildOptionalGroupRatesPayload(groupIDs: number[], rates: Record<number, number | undefined>): Record<number, number | null> {
  const payload: Record<number, number | null> = {}
  for (const groupID of groupIDs) {
    const value = Number(rates[groupID])
    payload[groupID] = Number.isFinite(value) && value > 0 ? value : null
  }
  return payload
}

function buildMemberGroupRatesPayload(groupIDs: number[], rates: Record<number, number | undefined>): Record<number, number | null> | undefined {
  if (!groupIDs.length) {
    return undefined
  }
  const payload: Record<number, number | null> = {}
  for (const groupID of groupIDs) {
    const value = Number(rates[groupID])
    payload[groupID] = Number.isFinite(value) && value > 0 ? value : null
  }
  return payload
}

function tenantGroupFloor(groupID: number): number {
  const selected = selectedTenant.value
  const value = selected?.group_rates?.[groupID]
  if (Number.isFinite(Number(value)) && Number(value) > 0) {
    return Number(value)
  }
  const group = groups.value.find((item) => item.id === groupID)
  if (Number.isFinite(Number(group?.rate_multiplier)) && Number(group?.rate_multiplier) > 0) {
    return Number(group?.rate_multiplier)
  }
  return Number(selected?.pricing_floor_factor || tenantForm.pricing_floor_factor || 1)
}

function tenantMemberDefaultRate(groupID: number): number {
  const selected = selectedTenant.value
  const groupRate = selected?.member_group_rates?.[groupID] ?? tenantForm.member_group_rates[groupID]
  if (Number.isFinite(Number(groupRate)) && Number(groupRate) > 0) {
    return Number(groupRate)
  }
  const fallback = selected?.member_default_pricing_factor ?? tenantForm.member_default_pricing_factor
  if (Number.isFinite(Number(fallback)) && Number(fallback) > 0) {
    return Number(fallback)
  }
  return tenantGroupFloor(groupID)
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

async function loadTenants() {
  loading.value = true
  try {
    const res = await enterpriseAdminAPI.listTenants(tenantPage.value, tenantPageSize, {
      search: search.value.trim() || undefined,
      status: tenantStatusFilter.value || undefined,
    })
    tenants.value = res.items
    tenantTotal.value = res.total
    if (!selectedTenant.value && res.items.length > 0) {
      await selectTenant(res.items[0])
    } else if (selectedTenant.value) {
      const next = res.items.find((item) => item.id === selectedTenant.value?.id)
      if (next) {
        selectedTenant.value = next
        fillTenantForm(next)
      } else if (res.items.length > 0) {
        await selectTenant(res.items[0])
      } else {
        selectedTenant.value = null
        members.value = []
        inviteCodes.value = []
        ledger.value = []
      }
    }
  } catch (error) {
    showError(error)
  } finally {
    loading.value = false
  }
}

async function applyTenantFilters() {
  tenantPage.value = 1
  await loadTenants()
}

async function refreshTenants() {
  await loadTenants()
}

async function changeTenantPage(page: number) {
  tenantPage.value = Math.min(Math.max(1, page), tenantPages.value)
  await loadTenants()
}

async function loadGroups() {
  try {
    groups.value = await groupsAPI.getAll()
  } catch (error) {
    showError(error)
  }
}

async function selectTenant(item: EnterpriseTenant) {
  selectedTenant.value = item
  fillTenantForm(item)
  memberPage.value = 1
  memberSearch.value = ''
  resetBindMemberForm()
  await Promise.all([loadMembers(), loadInviteCodes(), loadLedger()])
}

async function loadMembers() {
  if (!selectedTenant.value) return
  try {
    const res = await enterpriseAdminAPI.listMembers(selectedTenant.value.id, memberPage.value, memberPageSize, {
      search: memberSearch.value.trim() || undefined,
    })
    members.value = res.items
    memberTotal.value = res.total
  } catch (error) {
    showError(error)
  }
}

async function applyMemberFilters() {
  memberPage.value = 1
  await loadMembers()
}

async function changeMemberPage(page: number) {
  memberPage.value = Math.min(Math.max(1, page), memberPages.value)
  await loadMembers()
}

async function searchBindUsers() {
  const keyword = bindUserSearch.value.trim()
  bindUserSearched.value = true
  if (!keyword) {
    bindUserResults.value = []
    return
  }
  bindUserSearching.value = true
  try {
    const res = await usersAPI.list(1, 20, {
      search: keyword,
      sort_by: 'email',
      sort_order: 'asc',
      include_subscriptions: false,
    })
    bindUserResults.value = res.items
  } catch (error) {
    showError(error)
  } finally {
    bindUserSearching.value = false
  }
}

function selectBindUser(user: AdminUser) {
  selectedBindUser.value = user
  bindForm.user_id = user.id
}

function resetBindMemberForm() {
  bindForm.user_id = 0
  bindForm.member_role = 'member'
  bindForm.member_note = ''
  bindForm.pricing_factor = 0
  bindForm.group_rates = {}
  bindUserSearch.value = ''
  bindUserResults.value = []
  bindUserSearched.value = false
  selectedBindUser.value = null
}

async function loadInviteCodes() {
  if (!selectedTenant.value) return
  try {
    const res = await enterpriseAdminAPI.listInviteCodes(selectedTenant.value.id, 1, 100)
    inviteCodes.value = res.items
  } catch (error) {
    showError(error)
  }
}

async function loadLedger() {
  if (!selectedTenant.value) return
  try {
    const res = await enterpriseAdminAPI.listLedger(selectedTenant.value.id, 1, 100)
    ledger.value = res.items
  } catch (error) {
    showError(error)
  }
}

async function submitTenant() {
  submitting.value = true
  try {
    const code = tenantForm.code.trim()
    const payload: {
      name: string
      code?: string
      status: string
      notes: string
      portal_host: string
      pricing_floor_factor: number
      member_default_pricing_factor: number
      pricing_scope: string
      concurrency: number
      balance_overdraft_limit: number
      allowed_group_ids: number[]
      group_rates: Record<number, number>
      member_group_rates: Record<number, number | null>
    } = {
      name: tenantForm.name.trim(),
      status: tenantForm.status,
      notes: tenantForm.notes.trim(),
      portal_host: tenantForm.portal_host.trim(),
      pricing_floor_factor: Number(tenantForm.pricing_floor_factor) || 1,
      member_default_pricing_factor: Math.max(0, Number(tenantForm.member_default_pricing_factor) || 0),
      pricing_scope: tenantForm.pricing_scope,
      concurrency: Math.max(0, Number(tenantForm.concurrency) || 0),
      balance_overdraft_limit: Math.max(0, Number(tenantForm.balance_overdraft_limit) || 0),
      allowed_group_ids: [...tenantForm.allowed_group_ids],
      group_rates: buildGroupRatesPayload(tenantForm.allowed_group_ids, tenantForm.group_rates),
      member_group_rates: buildOptionalGroupRatesPayload(tenantForm.allowed_group_ids, tenantForm.member_group_rates),
    }
    if (code) {
      payload.code = code
    }
    if (selectedTenant.value?.id) {
      await enterpriseAdminAPI.updateTenant(selectedTenant.value.id, payload)
      showSuccess('企业已更新')
    } else {
      await enterpriseAdminAPI.createTenant(payload)
      showSuccess('企业已创建')
    }
    await loadTenants()
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function submitQuota() {
  if (!selectedTenant.value) return
  submitting.value = true
  try {
    selectedTenant.value = await enterpriseAdminAPI.adjustQuota(selectedTenant.value.id, {
      amount: Number(quotaForm.amount),
      direction: quotaForm.direction,
      notes: quotaForm.notes.trim() || undefined,
    })
    showSuccess('额度池已调整')
    await Promise.all([loadTenants(), loadLedger()])
    quotaForm.amount = 0
    quotaForm.notes = ''
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function submitBindMember() {
  if (!selectedTenant.value) return
  if (!bindForm.user_id) {
    showError(new Error('请先搜索并选择要绑定的用户'))
    return
  }
  submitting.value = true
  try {
    await enterpriseAdminAPI.bindMember(selectedTenant.value.id, {
      user_id: Number(bindForm.user_id),
      member_role: bindForm.member_role,
      member_note: bindForm.member_note,
      pricing_factor: Math.max(0, Number(bindForm.pricing_factor) || 0),
      pricing_scope: 'balance',
      group_rates: buildMemberGroupRatesPayload(selectedTenant.value.allowed_group_ids || [], bindForm.group_rates),
      joined_via: 'manual_bind',
      joined_source: 'admin_bind',
    })
    showSuccess('成员已绑定')
    resetBindMemberForm()
    await Promise.all([loadMembers(), loadTenants()])
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function removeMember(member: EnterpriseMembership) {
  if (!selectedTenant.value) return
  if (!window.confirm(`确认移除成员 ${member.user_email} 吗？`)) return
  try {
    await enterpriseAdminAPI.deleteMember(selectedTenant.value.id, member.user_id)
    showSuccess('成员已移除')
    await Promise.all([loadMembers(), loadTenants()])
  } catch (error) {
    showError(error)
  }
}

async function submitInvite() {
  if (!selectedTenant.value) return
  submitting.value = true
  try {
    await enterpriseAdminAPI.createInviteCode(selectedTenant.value.id, {
      code: inviteForm.code.trim() || undefined,
      max_uses: Number(inviteForm.max_uses) || 0,
      expires_at: inviteForm.expires_at ? new Date(inviteForm.expires_at).toISOString() : undefined,
      notes: inviteForm.notes.trim() || undefined,
    })
    showSuccess('邀请码已创建')
    inviteForm.code = ''
    inviteForm.max_uses = 0
    inviteForm.expires_at = ''
    inviteForm.notes = ''
    await loadInviteCodes()
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function toggleInvite(invite: EnterpriseInviteCode) {
  try {
    await enterpriseAdminAPI.updateInviteCode(invite.id, {
      status: invite.status === 'active' ? 'disabled' : 'active',
    })
    showSuccess('邀请码状态已更新')
    await loadInviteCodes()
  } catch (error) {
    showError(error)
  }
}

onMounted(() => {
  void Promise.all([loadTenants(), loadGroups()])
})
</script>
