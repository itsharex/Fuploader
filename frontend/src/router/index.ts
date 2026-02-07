import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard'
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('../views/Dashboard.vue'),
    meta: { title: '仪表盘', icon: 'Odometer' }
  },
  {
    path: '/accounts',
    name: 'Accounts',
    component: () => import('../views/Accounts.vue'),
    meta: { title: '账号管理', icon: 'User' }
  },
  {
    path: '/videos',
    name: 'Videos',
    component: () => import('../views/Videos.vue'),
    meta: { title: '视频管理', icon: 'VideoCamera' }
  },
  {
    path: '/publish',
    name: 'Publish',
    component: () => import('../views/Publish.vue'),
    meta: { title: '发布中心', icon: 'Upload' }
  },
  {
    path: '/tasks',
    name: 'Tasks',
    component: () => import('../views/Tasks.vue'),
    meta: { title: '任务队列', icon: 'List' }
  },
  {
    path: '/logs',
    name: 'Logs',
    component: () => import('../views/Logs.vue'),
    meta: { title: '日志', icon: 'Document' }
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('../views/Settings.vue'),
    meta: { title: '设置', icon: 'Setting' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
