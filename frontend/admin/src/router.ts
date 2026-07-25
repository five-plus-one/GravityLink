import { createRouter, createWebHistory } from 'vue-router';
import AdminLayout from './layouts/AdminLayout.vue';
import DashboardView from './views/DashboardView.vue';
import DomainsView from './views/DomainsView.vue';
import LandingPagesView from './views/LandingPagesView.vue';
import LinksView from './views/LinksView.vue';
import SettingsView from './views/SettingsView.vue';
import StatsView from './views/StatsView.vue';
import UsersView from './views/UsersView.vue';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: AdminLayout,
      children: [
        { path: '', name: 'dashboard', component: DashboardView },
        { path: 'links', name: 'links', component: LinksView },
        { path: 'domains', name: 'domains', component: DomainsView },
        { path: 'landing-pages', name: 'landing-pages', component: LandingPagesView },
        { path: 'stats', name: 'stats', component: StatsView },
        { path: 'users', name: 'users', component: UsersView },
        { path: 'settings', name: 'settings', component: SettingsView },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
});

export default router;
