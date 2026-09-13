<script setup lang="ts">
import ViewportTable from '../components/ViewportTable.vue';
import { h, onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import { RefreshCw, Search } from '@lucide/vue';
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NDatePicker,
  NEmpty,
  NInput,
  NPagination,
  NSelect,
  NTag,
  useMessage,
  type DataTableColumns,
  type PaginationInfo,
} from 'naive-ui';
import { listLinks, listVisitors, type LinkItem, type VisitorLogItem } from '../api';

import ResponsiveDateRange from '../components/ResponsiveDateRange.vue';

const route = useRoute();
const message = useMessage();

const items = ref<VisitorLogItem[]>([]);
const total = ref(0);
const error = ref('');
let requestId = 0;
let applied: Parameters<typeof listVisitors>[0] = {};
const presets = [{key:'today',label:'今天'},{key:'24h',label:'最近24小时'},{key:'7d',label:'近7天'},{key:'30d',label:'近30天'},{key:'all',label:'全部记录'}];
const loading = ref(false);
const links = ref<LinkItem[]>([]);

const filters = reactive({
  linkId: null as number | null,
  dateRange: null as [number, number] | null,
  keyword: '',
});

const activePreset = ref<string>('');

const page = ref(1);
// 服务端分页：pageSize 需要跟随选择器变化，因此用 ref
const pageSize = ref(50);

const linkOptions = ref<{ label: string; value: number }[]>([]);

function locationOf(row:VisitorLogItem){const parts=[row.country,row.province,row.city].filter(Boolean);return parts.includes('Reserved')?'内网或保留地址':[...new Set(parts)].join(' ') || '地域未知';}
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
      const loc = locationOf(row);
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
  applyPreset('all');
  await loadLinks();
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
  handleSearch();
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
  const id = ++requestId;
  loading.value = true;
  error.value = '';
  try {
    const result = await listVisitors({ ...applied, limit: pageSize.value, offset: (page.value - 1) * pageSize.value });
    if (id !== requestId) return;
    items.value = result.items ?? [];
    total.value = result.total;
  } catch (err) {
    if (id !== requestId) return;
    items.value = []; total.value = 0;
    error.value = err instanceof Error ? err.message : '加载访客记录失败';
  } finally {
    if (id === requestId) loading.value = false;
  }
}

function handlePageChange(p: number) {
  page.value = p;
  refresh();
}

function handlePageSizeChange(size: number) {
  pageSize.value = size;
  page.value = 1;
  refresh();
}

// 分页条左侧展示总条数（替代表格下方的独立说明文字）
function renderPaginationPrefix(info: PaginationInfo) {
  return h('span', { class: 'muted', style: 'font-size:12px' }, `共 ${info.itemCount ?? 0} 条记录`);
}

function handleSearch() {
  applied = {};
  if (filters.linkId) applied.link_id = filters.linkId;
  if (filters.dateRange) {
    applied.start = new Date(filters.dateRange[0]).toISOString();
    applied.end = new Date(filters.dateRange[1]).toISOString();
  }
  if (filters.keyword.trim()) applied.keyword = filters.keyword.trim();
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
            <NButton :loading="loading" @click="activePreset ? applyPreset(activePreset) : refresh()" aria-label="刷新访客记录">
              <template #icon><RefreshCw :size="16" /></template>
            </NButton>
          </div>
        </div>
      </template>

      <div class="filter-bar">
        <NSelect
          v-model:value="filters.linkId"
          class="f-link"
          :options="linkOptions"
          placeholder="全部链接"
          clearable
          filterable
          @update:value="handleSearch"
        />
        <ResponsiveDateRange v-model:value="filters.dateRange" class="f-date" with-time @update:value="activePreset = ''; handleSearch()" />
        <NInput
          v-model:value="filters.keyword"
          class="f-kw"
          placeholder="搜索 IP、短码或名称"
          clearable
          @keyup.enter="handleSearch"
        >
          <template #prefix><Search :size="14" /></template>
        </NInput>
        <NButton type="primary" :loading="loading" @click="handleSearch">查询</NButton>
      </div>

      <NAlert v-if="error" type="error" style="margin-bottom:16px">{{ error }}<NButton text @click="refresh">重试</NButton></NAlert>
      <!-- remote 必须与 itemCount 成对出现，否则 Naive UI 按本地数据算页数，翻页不显示 -->
      <ViewportTable class="desktop-visitors"
        :max-height="600"
        :columns="columns"
        :data="items"
        :loading="loading"
        remote
        :scroll-x="980"
        :bordered="false"
        size="small"
        :pagination="{
          page: page,
          pageSlot: 5,
          pageSize: pageSize,
          itemCount: total,
          showSizePicker: true,
          pageSizes: [20, 50, 100],
          prefix: renderPaginationPrefix,
          onUpdatePage: handlePageChange,
          onUpdatePageSize: handlePageSizeChange,
        }"
      >
        <template #empty>
          <NEmpty description="暂无访客记录" />
        </template>
      </ViewportTable>
      <div class="mobile-visitors">
        <div class="visitor-total">{{ loading ? '正在查询…' : `共 ${total} 条记录` }}</div>
        <NPagination :page="page" :page-size="pageSize" :item-count="total" :page-slot="5" @update:page="handlePageChange" />
        <NEmpty v-if="!items.length && !loading" description="所选条件暂无访客记录" />
        <article v-for="item in items" :key="item.id" class="visitor-card">
          <strong>{{ item.link_title || item.link_code || '链接已移除' }}</strong>
          <time>{{ new Date(item.visited_at).toLocaleString('zh-CN',{hour12:false}) }}</time>
          <div><code>{{ item.ip }}</code> · {{ deviceLabel[item.device] || '未知设备' }}</div>
          <div>{{ locationOf(item) }}</div>
          <div>{{ [item.os,item.browser].filter(Boolean).join(' / ') || '系统与浏览器未知' }}</div>
          <div class="visitor-source">{{ item.source_app || item.referer || '直接访问' }}</div>
        </article>
      </div>
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

.preset-bar {display:flex;flex-wrap:wrap;gap:8px;margin-bottom:12px}
.filter-bar {
  display: flex;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
  flex-wrap: wrap;
  align-items: center;
}

.f-link {
  width: 240px;
}

.f-date {
  width: min(100%, 390px);
}

.f-kw {
  width: 220px;
}

@media (max-width: 640px) {
  .f-link,
  .f-date,
  .f-kw {
    width: 100%;
  }
}

.muted {
  color: var(--color-text-tertiary, #7a8990);
}
.mobile-visitors{display:none}
@media(max-width:768px){.desktop-visitors{display:none}.mobile-visitors{display:grid;gap:12px}.visitor-card{border:1px solid var(--color-border);border-radius:10px;padding:14px;display:grid;gap:6px;font-size:13px;overflow-wrap:anywhere}.visitor-card time,.visitor-source{color:var(--color-text-tertiary)}.visitor-total{font-size:13px}}
</style>
