<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { ArrowRight, Globe2, LayoutTemplate, Link2, Plus } from '@lucide/vue';
import { NButton, NCard, NDataTable, NEmpty, NGrid, NGridItem, NSkeleton, NStatistic, NTag, useMessage, type DataTableColumns } from 'naive-ui';
import { listDomains, listLandingPages, listLinks, type DomainItem, type LandingPageItem, type LinkItem } from '../api';

const message = useMessage();

const links = ref<LinkItem[]>([]);
const domains = ref<DomainItem[]>([]);
const pages = ref<LandingPageItem[]>([]);
const loading = ref(true);

const activeLinks = computed(() => links.value.filter((item) => item.Status === 'active').length);
const recentLinks = computed(() => links.value.slice(0, 8));

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

onMounted(async () => {
  try {
    const [linkData, domainData, pageData] = await Promise.all([listLinks(), listDomains(), listLandingPages()]);
    links.value = linkData.items;
    domains.value = domainData.items;
    pages.value = pageData.items;
  } catch (err) {
    message.error(err instanceof Error ? err.message : '加载概览失败');
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="dashboard">
    <NGrid :cols="3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem span="3 m:1">
        <NCard class="metric-card" hoverable @click="$router.push('/links')">
          <NStatistic label="链接总数" :value="links.length">
            <template #prefix><Link2 :size="20" /></template>
            <template #suffix><span class="muted">{{ activeLinks }} 条可访问</span></template>
          </NStatistic>
        </NCard>
      </NGridItem>
      <NGridItem span="3 m:1">
        <NCard class="metric-card" hoverable @click="$router.push('/domains')">
          <NStatistic label="域名" :value="domains.length">
            <template #prefix><Globe2 :size="20" /></template>
            <template #suffix><span class="muted">入口 / 落地 / 中转</span></template>
          </NStatistic>
        </NCard>
      </NGridItem>
      <NGridItem span="3 m:1">
        <NCard class="metric-card" hoverable @click="$router.push('/landing-pages')">
          <NStatistic label="落地页" :value="pages.length">
            <template #prefix><LayoutTemplate :size="20" /></template>
            <template #suffix><span class="muted">公开页面模板</span></template>
          </NStatistic>
        </NCard>
      </NGridItem>
    </NGrid>

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
            <NButton type="primary" @click="$router.push({ path: '/links', query: { create: '1' } })">
              <template #icon><Plus :size="16" /></template>
              创建链接
            </NButton>
          </div>
        </div>
      </template>

      <NSkeleton v-if="loading" :repeat="4" height="40px" :sharp="false" />
      <NDataTable
        v-else-if="recentLinks.length"
        :columns="columns"
        :data="recentLinks"
        :pagination="false"
        size="small"
        :bordered="false"
      />
      <NEmpty v-else description="尚未创建链接">
        <template #extra>
          <NButton type="primary" @click="$router.push({ path: '/links', query: { create: '1' } })">
            <template #icon><Plus :size="16" /></template>
            立即创建
          </NButton>
        </template>
      </NEmpty>
    </NCard>

    <div class="quick-links">
      <RouterLink to="/stats" class="muted">查看访问统计 →</RouterLink>
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
}

.metric-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.metric-card :deep(.n-statistic-value__content) {
  font-size: var(--font-size-3xl);
  font-weight: 700;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  width: 100%;
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

.quick-links {
  text-align: center;
  font-size: var(--font-size-md);
}

.quick-links a {
  color: var(--color-primary);
  text-decoration: none;
}

.quick-links a:hover {
  text-decoration: underline;
}
</style>
