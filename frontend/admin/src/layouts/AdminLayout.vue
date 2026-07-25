<script setup lang="ts">
import {
  BarChart3,
  ExternalLink,
  Globe2,
  LayoutDashboard,
  LayoutTemplate,
  Link2,
  LogOut,
  Settings,
  Users,
} from '@lucide/vue';
import type { AuthConfig, AuthUser } from '../auth';

defineProps<{ user: AuthUser; authConfig: AuthConfig | null }>();
defineEmits<{ logout: [] }>();

const nav = [
  { to: '/', label: '概览', icon: LayoutDashboard },
  { to: '/links', label: '链接', icon: Link2 },
  { to: '/domains', label: '域名', icon: Globe2 },
  { to: '/landing-pages', label: '落地页', icon: LayoutTemplate },
  { to: '/stats', label: '统计', icon: BarChart3 },
  { to: '/users', label: '账号与权限', icon: Users },
  { to: '/settings', label: '系统设置', icon: Settings },
];
</script>

<template>
  <div class="admin-shell">
    <aside class="sidebar">
      <div class="brand"><span class="brand-mark">G</span><div><strong>GravityLink</strong><small>Admin</small></div></div>
      <nav class="nav">
        <RouterLink v-for="item in nav" :key="item.to" :to="item.to" class="nav-item">
          <component :is="item.icon" :size="18" /><span>{{ item.label }}</span>
        </RouterLink>
      </nav>
      <a class="public-link" href="/" target="_blank"><ExternalLink :size="16" />访问公开入口</a>
      <div class="sidebar-user">
        <div><strong>{{ user.username }}</strong><small>{{ user.role === 'super_admin' ? '超级管理员' : user.role }}</small></div>
        <button class="icon-button" title="退出登录" type="button" @click="$emit('logout')"><LogOut :size="18" /></button>
      </div>
    </aside>
    <main class="admin-main"><RouterView /></main>
  </div>
</template>
