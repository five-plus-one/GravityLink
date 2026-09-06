<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import { Activity, ArrowRight, Globe2, LayoutTemplate, Link2, Plus, QrCode } from '@lucide/vue';
import { NButton, NCard, NDataTable, NEmpty, NGrid, NGridItem, NSkeleton, NStatistic, NTag, useMessage, type DataTableColumns } from 'naive-ui';
import { listDomains, listLandingPages, listLinks, type DomainItem, type LandingPageItem, type LinkItem } from '../api';
import TrafficOverview from '../components/TrafficOverview.vue';
import { useAuthStore } from '../stores/auth';

const message = useMessage();
const auth = useAuthStore();
const canWrite = computed(() => auth.isSuperAdmin || auth.user?.role === 'admin');
const linksError = ref('');
const domainsError = ref('');
const pagesError = ref('');

const links = ref<LinkItem[]>([]);
const domains = ref<DomainItem[]>([]);
const pages = ref<LandingPageItem[]>([]);
const loading = ref(true);

const activeLinks = computed(() => links.value.filter((item) => item.Status === 'active').length);
const liveQrLinks = computed(() => links.value.filter((item) => item.Type === 'liveqr').length);
const recentLinks = computed(() => links.value.slice(0, 8));
const linkTypes = computed(() => {
  const total = Math.max(links.value.length, 1);
  return [
    { key: 'short', label: '短链接', count: links.value.filter((item) => item.Type === 'short').length, color: '#2787f5' },
    { key: 'channel', label: '渠道链接', count: links.value.filter((item) => item.Type === 'channel').length, color: '#45c4b0' },
    { key: 'liveqr', label: '活码', count: liveQrLinks.value, color: '#806ff3' },
  ].map((item) => ({ ...item, percent: Math.round((item.count / total) * 100) }));
});

const statusTypeMap: Record<string, 'success' | 'warning' | 'default' | 'error'> = {
  active: 'success',
  pending: 'warning',
  disabled: 'default',
  expired: 'error',
};
const statusLabelMap: Record<string, string> = {
  active: '正常',
  pending: '待审核',
  disabled: '已停用',
  expired: '已过期',
};
const typeLabelMap: Record<string, string> = {
  short: '短链接',
  channel: '渠道链接',
  liveqr: '活码',
};

const columns: DataTableColumns<LinkItem> = [
  { title: '短码', key: 'Code', width: 120, render: (row) => h('code', {}, row.Code) },
  { title: '名称', key: 'Title', render: (row) => row.Title || h('span', { class: 'muted' }, '未命名') },
  { title: '类型', key: 'Type', width: 100, render: (row) => typeLabelMap[row.Type] || row.Type },
  {
    title: '目标',
    key: 'TargetURL',
    ellipsis: { tooltip: true },
    render: (row) => row.TargetURL || h('span', { class: 'muted' }, '动态路由'),
  },
  {
    title: '状态',
    key: 'Status',
    width: 100,
    render: (row) =>
      h(NTag, { type: statusTypeMap[row.Status] || 'default', size: 'small', round: true }, () => statusLabelMap[row.Status] || row.Status),
  },
];

onMounted(refresh);
async function refresh() {
  if (loading.value && started) return;
  started = true;
  loading.value = true;
  linksError.value = domainsError.value = pagesError.value = '';
  const [linkResult, domainResult, pageResult] = await Promise.allSettled([
    listLinks(), canWrite.value ? listDomains() : Promise.resolve({ items: [] }), listLandingPages(),
  ]);
  if (linkResult.status === 'fulfilled') links.value = linkResult.value.items;
  else { links.value = []; linksError.value = '链接列表加载失败'; }
  if (domainResult.status === 'fulfilled') domains.value = domainResult.value.items;
  else { domains.value = []; domainsError.value = '域名数据加载失败'; }
  if (pageResult.status === 'fulfilled') pages.value = pageResult.value.items;
  else { pages.value = []; pagesError.value = '落地页数据加载失败'; }
  const errors = [linksError.value, domainsError.value, pagesError.value].filter(Boolean);
  if (errors.length) message.error(errors.join('；'));
  loading.value = false;
}
let started = false;
</script>

<template>
  <div class="dashboard">
    <NSkeleton v-if="loading" height="160px" :sharp="false" />
    <NCard v-else-if="linksError"><NEmpty :description="linksError"><template #extra><NButton @click="refresh">重新加载</NButton></template></NEmpty></NCard>
    <TrafficOverview v-else :links="links" />
    <div class="card-head"><strong>资源概况</strong><NButton :loading="loading" @click="refresh">刷新资源</NButton></div>
    <NGrid :cols="4" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem span="4 s:2 l:1">
        <NCard class="metric-card metric-blue" hoverable @click="$router.push('/links')">
          <NStatistic label="链接总数" :value="loading || linksError ? '—' : links.length">
            <template #prefix><Link2 :size="20" /></template>
            <template #suffix><span class="metric-note">全部链接</span></template>
          </NStatistic>
        </NCard>
      </NGridItem>
      <NGridItem span="4 s:2 l:1">
        <NCard class="metric-card metric-green" hoverable @click="$router.push('/links')">
          <NStatistic label="正常运行" :value="loading || linksError ? '—' : activeLinks">
            <template #prefix><Activity :size="20" /></template>
            <template #suffix><span class="metric-note">可访问链接</span></template>
          </NStatistic>
        </NCard>
      </NGridItem>
      <NGridItem span="4 s:2 l:1">
        <NCard class="metric-card metric-purple" hoverable @click="$router.push('/links')">
          <NStatistic label="活码" :value="loading || linksError ? '—' : liveQrLinks">
            <template #prefix><QrCode :size="20" /></template>
            <template #suffix><span class="metric-note">动态路由</span></template>
          </NStatistic>
        </NCard>
      </NGridItem>
      <NGridItem span="4 s:2 l:1">
        <NCard class="metric-card metric-orange" :hoverable="canWrite" @click="canWrite && $router.push('/domains')">
          <NStatistic label="域名" :value="loading || domainsError || !canWrite ? '—' : domains.length">
            <template #prefix><Globe2 :size="20" /></template>
            <template #suffix><span class="metric-note">{{ !canWrite ? '仅管理员可查看' : domainsError || '入口 / 落地 / 中转' }}</span></template>
          </NStatistic>
        </NCard>
      </NGridItem>
    </NGrid>

    <div class="dashboard-grid">
    <NCard class="recent-card">
      <template #header>
        <div class="card-head">
          <div>
            <strong>最近链接</strong>
            <p class="muted">最新创建的 8 条链接</p>
          </div>
          <div class="card-actions">
            <NButton text type="primary" tag="a" @click="$router.push('/links')">
              查看全部
              <template #icon><ArrowRight :size="14" /></template>
            </NButton>
            <NButton v-if="canWrite" type="primary" @click="$router.push({ path: '/links', query: { create: '1' } })">
              <template #icon><Plus :size="16" /></template>
              创建链接
            </NButton>
          </div>
        </div>
      </template>

      <NSkeleton v-if="loading" :repeat="4" height="40px" :sharp="false" />
      <NEmpty v-else-if="linksError" :description="linksError" />
      <NDataTable
        v-else-if="recentLinks.length"
        :columns="columns"
        :data="recentLinks"
        :pagination="false"
        :scroll-x="600"
        size="small"
        :bordered="false"
      />
      <NEmpty v-else description="尚未创建链接">
        <template #extra>
          <NButton v-if="canWrite" type="primary" @click="$router.push({ path: '/links', query: { create: '1' } })">
            <template #icon><Plus :size="16" /></template>
            立即创建
          </NButton>
        </template>
      </NEmpty>
    </NCard>

    <div class="side-stack">
      <NCard class="distribution-card" title="链接类型">
        <template #header-extra><span class="muted">{{ loading || linksError ? '—' : `共 ${links.length} 条` }}</span></template>
        <NSkeleton v-if="loading" height="100px" />
        <NEmpty v-else-if="linksError" :description="linksError" />
        <div v-else class="distribution-list">
          <div v-for="item in linkTypes" :key="item.key" class="distribution-item">
            <div class="distribution-head">
              <span><i :style="{ background: item.color }"></i>{{ item.label }}</span>
              <strong>{{ item.count }}</strong>
            </div>
            <div class="distribution-track"><span :style="{ width: `${item.percent}%`, background: item.color }"></span></div>
          </div>
        </div>
      </NCard>

      <NCard class="landing-card" hoverable @click="$router.push('/landing-pages')">
        <div class="landing-summary">
          <span class="landing-icon"><LayoutTemplate :size="22" /></span>
          <div><strong>{{ loading || pagesError ? '—' : pages.length }}</strong><p>{{ pagesError || '落地页模板' }}</p></div>
          <ArrowRight :size="18" />
        </div>
      </NCard>
    </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  display: grid;
  gap: var(--space-6);
}

.metric-card {
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
  min-height: 132px;
  background: rgba(255, 255, 255, 0.88);
}

.metric-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.metric-card :deep(.n-statistic-value__content) {
  font-size: var(--font-size-3xl);
  font-weight: 700;
  color: var(--color-text-primary);
}

.metric-card :deep(.n-statistic-value__prefix) {
  width: 44px;
  height: 44px;
  display: inline-grid;
  place-items: center;
  margin-right: var(--space-3);
  border-radius: var(--radius-md);
  color: var(--color-primary);
  background: var(--color-primary-soft);
}

.metric-card :deep(.n-statistic-value__suffix) {
  margin-left: var(--space-3);
}

.metric-note {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-xs);
  font-weight: 400;
}

.metric-green :deep(.n-statistic-value__prefix) { color: #159b83; background: #e2f8f2; }
.metric-purple :deep(.n-statistic-value__prefix) { color: #725fe8; background: #efecff; }
.metric-orange :deep(.n-statistic-value__prefix) { color: #dc8a17; background: #fff3dc; }

.recent-card {
  background: rgba(255, 255, 255, 0.9);
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(280px, 0.75fr);
  gap: var(--space-5);
  align-items: start;
}

.side-stack,
.distribution-list {
  display: grid;
  gap: var(--space-4);
}

.distribution-item { display: grid; gap: var(--space-2); }
.distribution-head { display: flex; align-items: center; justify-content: space-between; }
.distribution-head span { display: flex; align-items: center; gap: var(--space-2); color: var(--color-text-secondary); }
.distribution-head i { width: 8px; height: 8px; border-radius: 50%; }
.distribution-track { height: 7px; overflow: hidden; border-radius: var(--radius-pill); background: var(--color-bg-subtle); }
.distribution-track span { display: block; height: 100%; border-radius: inherit; }

.landing-card { cursor: pointer; background: linear-gradient(135deg, rgba(39, 135, 245, .96), rgba(112, 103, 240, .94)); color: #fff; }
.landing-summary { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: var(--space-3); }
.landing-summary strong { font-size: var(--font-size-2xl); }
.landing-summary p { color: rgba(255,255,255,.76); font-size: var(--font-size-md); }
.landing-icon { width: 44px; height: 44px; display: grid; place-items: center; border-radius: var(--radius-md); background: rgba(255,255,255,.16); }

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  width: 100%;
  flex-wrap: wrap;
}

.card-head strong {
  font-size: var(--font-size-lg);
  display: block;
}

.card-head p {
  margin-top: var(--space-1);
  font-size: var(--font-size-md);
}

.card-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

@media (max-width: 1100px) {
  .dashboard-grid { grid-template-columns: 1fr; }
}
</style>
