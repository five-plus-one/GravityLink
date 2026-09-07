import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from './stores/auth';
import { useSetupStore } from './stores/setup';

// 视图全部懒加载：登录页/初始化向导不再连带下载全部业务代码与图表库。
const AdminLayout = () => import('./layouts/AdminLayout.vue');
const LoginView = () => import('./views/LoginView.vue');
const SetupView = () => import('./views/SetupView.vue');
const AuthCallbackView = () => import('./views/AuthCallbackView.vue');
const DashboardView = () => import('./views/DashboardView.vue');
const DomainsView = () => import('./views/DomainsView.vue');
const LandingPagesView = () => import('./views/LandingPagesView.vue');
const LinksView = () => import('./views/LinksView.vue');
const SettingsView = () => import('./views/SettingsView.vue');
const StatsView = () => import('./views/StatsView.vue');
const UsersView = () => import('./views/UsersView.vue');
const ProfileView = () => import('./views/ProfileView.vue');
const NotFoundView = () => import('./views/NotFoundView.vue');

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
        { path: 'share-cards', name: 'share-cards', component: () => import('./views/ShareCardsView.vue'), meta: { title: '微信分享卡片', subtitle: '维护分享内容、封面和公众号配置' } },
        { path: '', name: 'dashboard', component: DashboardView, meta: { title: '数据概览', subtitle: '掌握链接、域名和落地页的运行情况' } },
        { path: 'links', name: 'links', component: LinksView, meta: { title: '链接管理', subtitle: '统一创建与维护短链接、渠道链接和活码' } },
        { path: 'domains', name: 'domains', component: DomainsView, meta: { title: '域名管理', subtitle: '维护入口、中转与落地域名' } },
        { path: 'landing-pages', name: 'landing-pages', component: LandingPagesView, meta: { title: '落地页', subtitle: '管理公开访问页面与展示模板' } },
        { path: 'stats', name: 'stats', component: StatsView, meta: { title: '数据看板', subtitle: '查看访问趋势、设备与地域分布' } },
        { path: 'users', name: 'users', component: UsersView, meta: { title: '账号与权限', subtitle: '管理后台账号、角色与访问权限' } },
        { path: 'profile', name: 'profile', component: ProfileView, meta: { title: '个人中心', subtitle: '查看当前账号信息与安全状态' } },
        { path: 'settings', name: 'settings', component: SettingsView, meta: { title: '系统设置', subtitle: '维护站点配置、认证方式与运行参数' } },
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
