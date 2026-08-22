import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from './stores/auth';
import { useSetupStore } from './stores/setup';
import AdminLayout from './layouts/AdminLayout.vue';
import LoginView from './views/LoginView.vue';
import SetupView from './views/SetupView.vue';
import AuthCallbackView from './views/AuthCallbackView.vue';
import DashboardView from './views/DashboardView.vue';
import DomainsView from './views/DomainsView.vue';
import LandingPagesView from './views/LandingPagesView.vue';
import LinksView from './views/LinksView.vue';
import SettingsView from './views/SettingsView.vue';
import StatsView from './views/StatsView.vue';
import UsersView from './views/UsersView.vue';
import NotFoundView from './views/NotFoundView.vue';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { public: true, title: '登录' } },
    { path: '/setup', name: 'setup', component: SetupView, meta: { public: true, title: '系统初始化' } },
    {
      path: '/auth/callback',
      name: 'auth-callback',
      component: AuthCallbackView,
      meta: { public: true, title: '登录回调' },
    },
    {
      path: '/setup/auth/callback',
      name: 'setup-callback',
      component: AuthCallbackView,
      meta: { public: true, title: '初始化回调' },
    },
    {
      path: '/',
      component: AdminLayout,
      children: [
        { path: '', name: 'dashboard', component: DashboardView, meta: { title: '概览' } },
        { path: 'links', name: 'links', component: LinksView, meta: { title: '链接' } },
        { path: 'domains', name: 'domains', component: DomainsView, meta: { title: '域名' } },
        { path: 'landing-pages', name: 'landing-pages', component: LandingPagesView, meta: { title: '落地页' } },
        { path: 'stats', name: 'stats', component: StatsView, meta: { title: '统计' } },
        { path: 'users', name: 'users', component: UsersView, meta: { title: '账号与权限' } },
        { path: 'settings', name: 'settings', component: SettingsView, meta: { title: '系统设置' } },
      ],
    },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView, meta: { public: true, title: '页面不存在' } },
  ],
});

/**
 * 全局守卫：
 * 1. 系统未初始化 → 强制 /setup
 * 2. 已初始化但访问 /setup → 跳 /
 * 3. 未登录访问受保护路由 → 跳 /login，记录 redirect
 * 4. 已登录访问 /login → 跳原目标或 /
 */
router.beforeEach(async (to) => {
  const setup = useSetupStore();
  const auth = useAuthStore();

  if (!setup.loaded) {
    try {
      await setup.refresh();
    } catch {
      // setup API 不可用（如本地 dev 只代理业务端口）：视为已初始化，
      // 继续走登录检查，而不是直接放行导致用户态未加载。
      setup.loaded = true;
    }
  }

  if (setup.setupRequired) {
    return to.name === 'setup' || to.name === 'setup-callback' ? true : { name: 'setup' };
  }
  if (to.name === 'setup' || to.name === 'setup-callback') {
    return { name: 'dashboard' };
  }

  if (to.meta.public) {
    if (to.name === 'login') {
      if (!auth.loaded) await auth.fetchUser();
      if (auth.isLoggedIn) {
        const redirect = typeof to.query.redirect === 'string' ? to.query.redirect : '/';
        return redirect;
      }
    }
    return true;
  }

  if (!auth.loaded) await auth.fetchUser();
  if (!auth.isLoggedIn) {
    return { name: 'login', query: to.fullPath === '/' ? {} : { redirect: to.fullPath } };
  }
  return true;
});

router.afterEach((to) => {
  const title = (to.meta.title as string) || '';
  document.title = title ? `${title} · GravityLink Admin` : 'GravityLink Admin';
});

export default router;
