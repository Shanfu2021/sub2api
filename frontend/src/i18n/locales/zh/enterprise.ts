export default {
  enterpriseManagement: {
    employees: {
      subtitle: '管理员或代理分配给企业的余额、并发和 RPM，可以继续分配给员工。',
      create: '创建员工',
      created: '员工已创建',
      createFailed: '创建员工失败',
      deleted: '员工已删除，余额和额度已返还',
      deleteTitle: '删除员工',
      deleteConfirm: "确定删除员工 '{email}' 吗？余额和额度会返还给企业。",
      deleteFailed: '删除员工失败',
      groupsTitle: "员工分组：{email}",
      emptyGroups: '暂无可分配给员工的专属分组',
      groupSaved: '员工分组已保存',
      groupRemoved: '员工分组已移除',
      groupFailed: '保存员工分组失败',
      defaultGroups: {
        title: '默认员工分组',
        description: '创建员工或批量导入员工后，会自动把勾选的分组分配给员工；倍率沿用企业当前有效倍率，企业不能单独改倍率。',
        saved: '默认员工分组已保存',
        removed: '默认员工分组已移除',
        failed: '保存默认员工分组失败'
      },
      insufficientAllocation: '剩余并发或 RPM 不足，不能分配',
      loadFailed: '加载员工失败',
      import: {
        title: '导入员工',
        file: 'JSON 文件',
        formatTitle: '导入格式：每条员工记录必须完整包含 email、username、password、concurrency、rpm 这 5 个字段',
        skipHint: '余额不会通过导入分配，所有导入员工余额固定为 0；没有完整 5 个字段或字段无效的记录会跳过。',
        validRows: '文件记录数',
        requiredQuota: '预计需要',
        row: '行',
        reason: '原因',
        submit: '开始导入',
        importing: '导入中...',
        result: '导入完成：创建 {created} 个，跳过 {skipped} 个',
        failed: '导入员工失败',
        invalidJson: 'JSON 文件解析失败',
        invalidJsonShape: 'JSON 必须是员工数组，或者包含 employees 数组',
        emptyFile: '文件里没有可导入的员工记录',
        quotaExceeded: '企业剩余额度不足：当前并发 {currentConcurrency}，需要 {requiredConcurrency}；当前 RPM {currentRpm}，需要 {requiredRpm}'
      },
      balanceInit: {
        title: '初始化余额',
        target: '所有员工余额设为',
        hint: '提交后会把所有直属员工余额统一设为这个数值。员工余额不足的部分会从企业余额扣除，员工余额高于目标的部分会退回企业。',
        submit: '初始化',
        success: '已初始化 {count} 个员工余额为 {balance}，本次净扣除 {required}',
        failed: '初始化员工余额失败',
        exceeded: '企业余额不足：当前企业余额 {current}，本次需要 {required}'
      }
    },
    groups: {
      subtitle: '查看上级分配给企业的公共和专属分组；这里只展示企业自己的有效倍率。',
      loadFailed: '加载企业分组失败'
    }
  },
}
