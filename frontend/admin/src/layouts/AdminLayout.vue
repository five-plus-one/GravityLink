<script setup lang="ts">
import { computed, h, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
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
import { NAvatar, NButton, NDropdown, NIcon, NLayout, NLayoutSider, NMenu, NText, useMessage, type MenuOption } from 'naive-ui';
import { useAuthStore } from '../stores/auth';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const message = useMessage();

const collapsed = ref(false);

const nav = [
  { key: 'dashboard', label: '概览', icon: LayoutDashboard, to: '/' },
  { key: 'links', label: '链接', icon: Link2, to: '/links' },
  { key: 'domains', label: '域名', icon: Globe2, to: '/domains' },
  { key: 'landing-pages', label: '落地页', icon: LayoutTemplate, to: '/landing-pages' },
  { key: 'stats', label: '统计', icon: BarChart3, to: '/stats' },
  { key: 'users', label: '账号与权限', icon: Users, to: '/users' },
  { key: 'settings', label: '系统设置', icon: Settings, to: '/settings' },
];

const menuOptions: MenuOption[] = nav.map((item) => ({
  key: item.key,
  label: item.label,
  icon: () => h(NIcon, { size: 18 }, { default: () => h(item.icon) }),
}));

const activeKey = computed(() => (route.name as string) || 'dashboard');
const pageTitle = computed(() => (route.meta.title as string) || '');

const roleLabel = computed(() => {
  const role = auth.user?.role;
  if (role === 'super_admin') return '超级管理员';
  if (role === 'admin') return '管理员';
  return '普通用户';
});

const userMenuOptions = [
  { label: '退出登录', key: 'logout', icon: () => h(NIcon, { size: 16 }, { default: () => h(LogOut) }) },
];

async function handleUserMenu(key: string) {
  if (key !== 'logout') return;
  try {
    await auth.signOut();
    message.success('已退出登录');
    await router.replace({ name: 'login' });
  } catch (err) {
    message.error(err instanceof Error ? err.message : '退出失败');
  }
}

function handleMenuSelect(key: string) {
  const target = nav.find((item) => item.key === key);
  if (target) router.push(target.to);
}
</script>

<template>
  <NLayout has-sider class="admin-shell">
    <NLayoutSider
      v-model:collapsed="collapsed"
      :width="232"
      :collapsed-width="64"
      collapse-mode="width"
      show-trigger
      class="admin-sider"
    >
      <div class="sider-brand">
        <span class="brand-mark">G</span>
        <Transition name="fade">
          <div v-if="!collapsed" class="brand-text">
            <strong>GravityLink</strong>
            <small>Admin</small>
          </div>
        </Transition>
      </div>

      <NMenu
        :value="activeKey"
        :options="menuOptions"
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="20"
        class="sider-menu"
        @update:value="handleMenuSelect"
      />

      <div class="sider-footer">
        <NButton v-if="!collapsed" text tag="a" href="/" target="_blank" class="public-link">
          <template #icon><ExternalLink :size="14" /></template>
          访问公开入口
        </NButton>

        <NDropdown :options="userMenuOptions" trigger="click" @select="handleUserMenu">
          <div class="sider-user">
            <NAvatar round size="small" class="user-avatar">{{ auth.user?.username?.slice(0, 1).toUpperCase() }}</NAvatar>
            <Transition name="fade">
              <div v-if="!collapsed" class="user-meta">
                <NText strong class="user-name">{{ auth.user?.username }}</NText>
                <NText depth="3" class="user-role">{{ roleLabel }}</NText>
              </div>
            </Transition>
          </div>
        </NDropdown>
      </div>
    </NLayoutSider>

    <NLayout class="admin-main">
      <header class="main-header">
        <h1>{{ pageTitle }}</h1>
      </header>
      <main class="main-body">
        <RouterView v-slot="{ Component }">
          <Transition name="page" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </main>
    </NLayout>
  </NLayout>
</template>

<style scoped>
.admin-shell {
  min-height: 100vh;
}

.admin-sider {
  background: var(--color-bg-sidebar);
}

.admin-sider :deep(.n-layout-sider-scroll-container) {
  display: flex;
  flex-direction: column;
}

.sider-brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-5) var(--space-4);
  color: var(--color-text-inverse);
}

.brand-text strong,
.brand-text small {
  display: block;
}

.brand-text small {
  margin-top: 2px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-sm);
}

.sider-menu {
  flex: 1;
  background: transparent;
  --n-item-text-color: rgba(255, 255, 255, 0.72);
  --n-item-text-color-hover: #fff;
  --n-item-text-color-active: #fff;
  --n-item-color-hover: rgba(255, 255, 255, 0.06);
  --n-item-color-active: var(--color-primary);
  --n-item-icon-color: rgba(255, 255, 255, 0.72);
  --n-item-icon-color-hover: #fff;
  --n-item-icon-color-active: #fff;
  --n-arrow-color: rgba(255, 255, 255, 0.6);
}

.sider-footer {
  padding: var(--space-3) var(--space-4) var(--space-4);
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  display: grid;
  gap: var(--space-3);
}

.public-link {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-md);
  justify-content: flex-start;
}

.public-link:hover {
  color: var(--color-text-inverse);
}

.sider-user {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background 0.15s;
}

.sider-user:hover {
  background: rgba(255, 255, 255, 0.05);
}

.user-avatar {
  background: var(--color-primary);
  color: #fff;
  font-weight: 600;
}

.user-meta {
  min-width: 0;
}

.user-name,
.user-role {
  display: block;
  color: var(--color-text-inverse);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-role {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-xs);
}

.admin-main {
  display: flex;
  flex-direction: column;
  background: var(--color-bg-page);
}

.main-header {
  height: 64px;
  padding: 0 var(--space-6);
  display: flex;
  align-items: center;
  background: var(--color-bg-surface);
  border-bottom: 1px solid var(--color-border);
}

.main-header h1 {
  font-size: var(--font-size-xl);
  font-weight: 600;
}

.main-body {
  flex: 1;
  padding: var(--space-6);
  overflow-y: auto;
}

.main-body > * {
  max-width: var(--layout-content-max);
  margin: 0 auto;
}

/* 过渡 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.page-enter-active,
.page-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.page-enter-from {
  opacity: 0;
  transform: translateY(4px);
}
.page-leave-to {
  opacity: 0;
}

@media (max-width: 767px) {
  .main-body {
    padding: var(--space-4);
  }
  .main-header {
    padding: 0 var(--space-4);
  }
}
</style>
