<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import { RefreshCw, Search } from '@lucide/vue';
import {
  NButton,
  NCard,
  NDataTable,
  NDatePicker,
  NEmpty,
  NInput,
  NSelect,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui';
import { listLinks, listVisitors, type LinkItem, type VisitorLogItem } from '../api';

const route = useRoute();
const message = useMessage();

const items = ref<VisitorLogItem[]>([]);
const total = ref(0);
const loading = ref(false);
const links = ref<LinkItem[]>([]);

const filters = reactive({
  linkId: null as number | null,
  dateRange: null as [number, number] | null,
  keyword: '',
});

const activePreset = ref<string>('');

const page = ref(1);
const pageSize = 50;

const linkOptions = ref<{ label: string; value: number }[]>([]);

const deviceLabel: Record<string, string> = {
  mobile: '手机', tablet: '平板', desktop: '电脑', bot: '爬虫', unknown: '未知',
};
const deviceType: Record<string, 'success' | 'info' | 'warning' | 'default'> = {
  mobile: 'success', tablet: 'info', desktop: 'default', bot: 'warning', unknown: 'default',
};

const columns: DataTableColumns<VisitorLogItem> = [
  {
    title: '时间',
    key: 'visited_at',
    width: 170,
    render: (row) => new Date(row.visited_at).toLocaleString('zh-CN', { hour12: false }),
  },
  {
    title: '链接',
    key: 'link_code',
    width: 160,
    render: (row) =>
      h('div', { style: 'display:flex;flex-direction:column;gap:2px' }, [
        h('code', { style: 'font-size:12px' }, row.link_code || '—'),
        row.link_title ? h('span', { style: 'font-size:11px;color:#7a8990;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:140px' }, row.link_title) : null,
      ]),
  },
  { title: 'IP', key: 'ip', width: 140, render: (row) => h('code', { style: 'font-size:12px' }, row.ip) },
  {
    title: '地域',
    key: 'location',
    width: 140,
    render: (row) => {
      const loc = [row.country, row.province, row.city].filter(Boolean).join(' ');
      return loc || h('span', { style: 'color:#94a3b8' }, '—');
    },
  },
  {
    title: '设备',
    key: 'device',
    width: 80,
    render: (row) => h(NTag, { type: deviceType[row.device] || 'default', size: 'small', round: true }, () => deviceLabel[row.device] || row.device),
  },
  {
    title: '系统 / 浏览器',
    key: 'os_browser',
    width: 160,
    render: (row) => {
      const parts = [row.os, row.browser].filter(Boolean);
      return parts.length ? parts.join(' / ') : h('span', { style: 'color:#94a3b8' }, '—');
    },
  },
  {
    title: '来源',
    key: 'source',
    width: 120,
    render: (row) => {
      if (row.source_app) return h(NTag, { size: 'small', round: true }, () => row.source_app);
      if (row.referer) {
        try { return h('span', { style: 'font-size:11px;color:#7a8990;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:110px' }, new URL(row.referer).hostname); } catch { /* ignore */ }
      }
      return h('span', { style: 'color:#94a3b8' }, '直接访问');
    },
  },
];

onMounted(async () => {
  const qLink = route.query.linkId;
  if (qLink) filters.linkId = Number(qLink);
  // 默认显示今天
  applyPreset('today');
  await Promise.all([loadLinks(), refresh()]);
});

function applyPreset(key: string) {
  activePreset.value = key;
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  switch (key) {
    case 'today':
      filters.dateRange = [today.getTime(), now.getTime()];
      break;
    case '24h':
      filters.dateRange = [now.getTime() - 24 * 3600 * 1000, now.getTime()];
      break;
    case '7d':
      filters.dateRange = [today.getTime() - 6 * 86400 * 1000, now.getTime()];
      break;
    case '30d':
      filters.dateRange = [today.getTime() - 29 * 86400 * 1000, now.getTime()];
      break;
    case '180d':
      filters.dateRange = [today.getTime() - 179 * 86400 * 1000, now.getTime()];
      break;
    case 'all':
      filters.dateRange = null;
      break;
  }
  page.value = 1;
  refresh();
}

async function loadLinks() {
  try {
    links.value = (await listLinks()).items;
    linkOptions.value = [
      { label: '全部链接', value: 0 },
      ...links.value.map((l) => ({ label: `${l.Title || '未命名'} · ${l.Code}`, value: l.ID })),
    ];
  } catch { /* ignore */ }
}

async function refresh() {
  loading.value = true;
  try {
    const params: Parameters<typeof listVisitors>[0] = {
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    };
    if (filters.linkId && filters.linkId > 0) params.link_id = filters.linkId;
    if (filters.dateRange) {
      params.start = new Date(filters.dateRange[0]).toISOString().slice(0, 10);
      params.end = new Date(filters.dateRange[1]).toISOString().slice(0, 10);
    }
    if (filters.keyword.trim()) params.keyword = filters.keyword.trim();
    const result = await listVisitors(params);
    items.value = result.items;
    total.value = result.total;
  } catch (err) {
    message.error(err instanceof Error ? err.message : '加载访客记录失败');
  } finally {
    loading.value = false;
  }
}

function handlePageChange(p: number) {
  page.value = p;
  refresh();
}

function handleSearch() {
  page.value = 1;
  refresh();
}
</script>

<template>
  <div class="page-view">
    <NCard>
      <template #header>
        <div class="card-head">
          <div>
            <strong>访客记录</strong>
            <p class="muted">查看全部或指定链接的详细访问明细</p>
          </div>
          <div class="card-actions">
            <NButton :loading="loading" @click="refresh">
              <template #icon><RefreshCw :size="16" /></template>
            </NButton>
          </div>
        </div>
      </template>

      <div class="filter-bar">
        <NSelect
          v-model:value="filters.linkId"
          :options="linkOptions"
          placeholder="全部链接"
          clearable
          filterable
          style="width: 240px"
          @update:value="handleSearch"
        />
        <NDatePicker
          v-model:value="filters.dateRange"
          type="daterange"
          clearable
          style="width: 280px"
          @update:value="handleSearch"
        />
        <NInput
          v-model:value="filters.keyword"
          placeholder="搜索 IP、短码或名称"
          clearable
          style="width: 220px"
          @keyup.enter="handleSearch"
        >
          <template #prefix><Search :size="14" /></template>
        </NInput>
        <NButton type="primary" @click="handleSearch">查询</NButton>
      </div>

      <NDataTable
        :columns="columns"
        :data="items"
        :loading="loading"
        :bordered="false"
        size="small"
        :pagination="{
          page: page,
          pageSize: pageSize,
          itemCount: total,
          showSizePicker: false,
          onUpdatePage: handlePageChange,
        }"
      >
        <template #empty>
          <NEmpty description="暂无访客记录" />
        </template>
      </NDataTable>

      <p v-if="total" class="muted" style="margin-top: 12px; font-size: 12px">共 {{ total }} 条记录</p>
    </NCard>
  </div>
</template>

<style scoped>
.page-view {
  display: grid;
  gap: var(--space-4);
  align-content: start;
}

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
  gap: var(--space-2);
}

.filter-bar {
  display: flex;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
  flex-wrap: wrap;
  align-items: center;
}

.muted {
  color: var(--color-text-tertiary, #7a8990);
}
</style>
