import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import admin from './admin'
import misc from './misc'
import agent from './agent'
import enterprise from './enterprise'
import purchase from './purchase'

export default {
  ...landing,
  ...common,
  ...dashboard,
  admin,
  ...misc,
  ...agent,
  ...enterprise,
  ...purchase,
}
