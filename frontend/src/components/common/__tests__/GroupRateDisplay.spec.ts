import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import GroupBadge from '../GroupBadge.vue'
import GroupOptionItem from '../GroupOptionItem.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('group rate display', () => {
  it('does not reveal the default rate when GroupBadge shows a user-specific rate', () => {
    const wrapper = mount(GroupBadge, {
      props: {
        name: 'OpenAI',
        platform: 'openai',
        rateMultiplier: 0.165,
        userRateMultiplier: 0.24
      },
      global: {
        stubs: {
          PlatformIcon: true
        }
      }
    })

    expect(wrapper.text()).toContain('0.24x')
    expect(wrapper.text()).not.toContain('0.165x')
    expect(wrapper.find('.line-through').exists()).toBe(false)
  })

  it('does not reveal the default rate when GroupOptionItem shows a user-specific rate', () => {
    const wrapper = mount(GroupOptionItem, {
      props: {
        name: 'OpenAI',
        platform: 'openai',
        rateMultiplier: 0.165,
        userRateMultiplier: 0.24
      },
      global: {
        stubs: {
          GroupBadge: {
            props: ['name'],
            template: '<span>{{ name }}</span>'
          }
        }
      }
    })

    expect(wrapper.text()).toContain('0.24x')
    expect(wrapper.text()).not.toContain('0.165x')
    expect(wrapper.find('.line-through').exists()).toBe(false)
  })
})
