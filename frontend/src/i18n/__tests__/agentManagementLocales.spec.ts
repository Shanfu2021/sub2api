import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('agent management locale copy', () => {
  it('keeps agent-side delete warning simple and does not expose admin rehome details', () => {
    expect(zh.agentManagement.direct.deleteChildConfirm).toContain('将不再显示在你的直属列表中')
    expect(zh.agentManagement.direct.deleteChildConfirm).not.toContain('不会被真实删除')
    expect(zh.agentManagement.direct.deleteChildConfirm).not.toContain('挂到管理员')

    expect(en.agentManagement.direct.deleteChildConfirm).toContain('will no longer appear in your direct list')
    expect(en.agentManagement.direct.deleteChildConfirm).not.toContain('permanently deleted')
    expect(en.agentManagement.direct.deleteChildConfirm).not.toContain('rehome')
    expect(en.agentManagement.direct.deleteChildConfirm).not.toContain('admin')
  })
})
