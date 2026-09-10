<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
  BarChart3,
  ExternalLink,
  Globe2,
  Key,
  LayoutDashboard,
  LayoutTemplate,
  Link2,
  LogOut,
  Menu,
  Settings,
  UserRound,
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
  { key: 'dashboard', label: '概览', icon: LayoutDashboard, to: '/', adminOnly: false },
  { key: 'links', label: '链接', icon: Link2, to: '/links', adminOnly: false },
  { key: 'share-cards', label: '微信分享卡片', icon: Link2, to: '/share-cards', adminOnly: false },
  { key: 'domains', label: '域名', icon: Globe2, to: '/domains', adminOnly: true },
  { key: 'landing-pages', label: '落地页', icon: LayoutTemplate, to: '/landing-pages', adminOnly: false },
  { key: 'stats', label: '统计', icon: BarChart3, to: '/stats', adminOnly: false },
  { key: 'visitors', label: '访客记录', icon: BarChart3, to: '/visitors', adminOnly: false },
  { key: 'api-keys', label: '开放 API', icon: Key, to: '/api-keys', adminOnly: true },
  { key: 'kami', label: '卡密分发', icon: Key, to: '/kami', adminOnly: true },
  { key: 'users', label: '账号与权限', icon: Users, to: '/users', adminOnly: true },
  { key: 'profile', label: '个人中心', icon: UserRound, to: '/profile', adminOnly: false },
  { key: 'settings', label: '系统设置', icon: Settings, to: '/settings', adminOnly: true },
];

const isAdmin = computed(() => auth.isSuperAdmin || auth.user?.role === 'admin');
const visibleNav = computed(() => nav.filter((item) => !item.adminOnly || isAdmin.value));

const menuOptions = computed<MenuOption[]>(() =>
  visibleNav.value.map((item) => ({
    key: item.key,
    label: item.label,
    icon: () => h(NIcon, { size: 18 }, { default: () => h(item.icon) }),
  })),
);

const activeKey = computed(() => (route.name as string) || 'dashboard');
const pageTitle = computed(() => (route.meta.title as string) || '');
const pageSubtitle = computed(() => (route.meta.subtitle as string) || '集中管理链接、域名与访问数据');
const publicEntryUrl = ref('');

onMounted(async () => {
  try {
    const { request } = await import('../api');
    const data = await request<{ configs: Record<string, string> }>('/api/admin/configs');
    publicEntryUrl.value = data.configs?.['public.base_url'] || '';
  } catch { /* ignore */ }
});

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
  const target = visibleNav.value.find((item) => item.key === key);
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
        <NButton v-if="!collapsed && publicEntryUrl" text tag="a" :href="publicEntryUrl" target="_blank" class="public-link">
          <template #icon><ExternalLink :size="14" /></template>
          查看访问地址
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
        <NButton quaternary circle aria-label="折叠侧栏" @click="collapsed = !collapsed">
          <template #icon><Menu :size="20" /></template>
        </NButton>
        <div class="header-user">
          <NText depth="2">{{ auth.user?.username }}</NText>
          <NAvatar round size="small" class="header-avatar">{{ auth.user?.username?.slice(0, 1).toUpperCase() }}</NAvatar>
        </div>
      </header>
      <main class="main-body">
        <div class="page-heading">
          <h1>{{ pageTitle }}</h1>
          <p>{{ pageSubtitle }}</p>
        </div>
        <RouterView v-slot="{ Component }">
          <div :key="route.path" class="route-content">
            <component :is="Component" />
          </div>
        </RouterView>
      </main>
    </NLayout>
  </NLayout>
</template>

<style scoped>
.admin-shell {
  height: 100vh;
}

@supports (height: 100dvh) {
  .admin-shell {
    height: 100dvh;
  }
}

/* 高度穿透到 NLayout 内部滚动容器，保证侧边栏与主区撑满视口 */
.admin-shell :deep(.n-layout-scroll-container) {
  height: 100%;
}

.admin-sider {
  background: var(--color-bg-sidebar);
  border-right: 1px solid rgba(229, 236, 245, 0.9);
  backdrop-filter: blur(18px);
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
  color: var(--color-text-primary);
  border-bottom: 1px solid var(--color-border);
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
  padding: var(--space-3) var(--space-2);
  --n-item-text-color: var(--color-text-secondary);
  --n-item-text-color-hover: var(--color-primary);
  --n-item-text-color-active: var(--color-primary);
  --n-item-text-color-active-hover: var(--color-primary);
  --n-item-color-hover: #f2f7ff;
  --n-item-color-active: var(--color-primary-soft);
  --n-item-color-active-hover: var(--color-primary-soft);
  --n-item-icon-color: var(--color-text-secondary);
  --n-item-icon-color-hover: var(--color-primary);
  --n-item-icon-color-active: var(--color-primary);
  --n-arrow-color: var(--color-text-tertiary);
}

.sider-menu :deep(.n-menu-item-content) {
  border-radius: var(--radius-md);
}

.sider-footer {
  padding: var(--space-3) var(--space-4) var(--space-4);
  border-top: 1px solid var(--color-border);
  display: grid;
  gap: var(--space-3);
}

.public-link {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-md);
  justify-content: flex-start;
}

.public-link:hover {
  color: var(--color-primary);
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
  background: var(--color-primary-soft);
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
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-role {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-xs);
}

.admin-main {
  background:
    radial-gradient(circle at 16% 8%, rgba(77, 159, 255, 0.14), transparent 30%),
    radial-gradient(circle at 84% 10%, rgba(146, 125, 255, 0.12), transparent 32%),
    radial-gradient(circle at 70% 88%, rgba(75, 205, 205, 0.09), transparent 34%),
    var(--color-bg-page);
}

/* 让 header + main 的纵向 flex 布局作用在 NLayout 内部滚动容器上 */
.admin-main :deep(.n-layout-scroll-container) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.main-header {
  height: 64px;
  padding: 0 var(--space-6);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba(255, 255, 255, 0.82);
  border-bottom: 1px solid var(--color-border);
  backdrop-filter: blur(18px);
}

.header-user {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-size: var(--font-size-md);
}

.header-avatar {
  background: linear-gradient(135deg, var(--color-primary), #7067f0);
  color: #fff;
  font-weight: 700;
}

.main-body {
  flex: 1;
  min-height: 0;
  padding: var(--space-8);
  overflow-y: auto;
}

.main-body > * {
  max-width: var(--layout-content-max);
  margin: 0 auto;
}

.page-heading {
  margin-bottom: var(--space-6);
}

.page-heading h1 {
  font-size: var(--font-size-3xl);
  line-height: 1.25;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.page-heading p {
  margin-top: var(--space-2);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-base);
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
