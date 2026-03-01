import type { RouteRecordRaw } from 'vue-router'
import { ApiOutlined } from '@ant-design/icons-vue'

export const apisixRoutes: RouteRecordRaw[] = [
  {
    path: 'apisix',
    name: 'APISIX',
    component: () => import('@/views/apisix/ApisixConsole.vue'),
    meta: {
      name: () => 'APISIX',
      icon: ApiOutlined,
    },
  },
]
