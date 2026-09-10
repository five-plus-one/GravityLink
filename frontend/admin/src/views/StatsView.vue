<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
// 统一由 echarts.ts 注册图表组件（含饼图），模块加载即完成注册
import '../echarts';
import VChart from 'vue-echarts';
import { RefreshCw, RotateCcw } from '@lucide/vue';
import { NButton, NCard, NDatePicker, NEmpty, NGrid, NGridItem, NPopconfirm, NSelect, NSkeleton, NStatistic, useMessage } from 'naive-ui';
import {
  getDailyStats,
  getDeviceStats,
  getGeoStats,
  getHourlyStats,
  getOverviewDaily,
  getOverviewHourly,
  getOverviewSummary,
  getSummaryStats,
  listLinks,
  resetLinkStats,
  type DailyPoint,
  type DeviceStats,
  type HourlyPoint,
  type LabelValue,
  type LinkItem,
  type SummaryStats,
} from '../api';
import { useAuthStore } from '../stores/auth';

const message = useMessage();
const route = useRoute();
const auth = useAuthStore();

const links = ref<LinkItem[]>([]);
const selectedLinkId = ref<number>(0); // 0 = 全部链接
const loading = ref(false);
const summary = ref<SummaryStats | null>(null);
const daily = ref<DailyPoint[]>([]);
const hourly = ref<HourlyPoint[]>([]);
const deviceStats = ref<DeviceStats | null>(null);
const geo = ref<LabelValue[]>([]);

const deviceLabels: Record<string, string> = {
  mobile: '手机',
  tablet: '平板',
  desktop: '桌面',
  bot: '机器人',
  unknown: '未知',
};

function labelOf(value: string): string {
  return deviceLabels[value] ?? (value || '未知');
}

const linkOptions = computed(() => [
  { label: '全部链接', value: 0 },
  ...links.value.map((link) => ({
    label: `${link.Code}${link.Title ? ` · ${link.Title}` : ''}`,
    value: link.ID,
  })),
]);

const dailyOption = computed(() => ({
  grid: { left: 40, right: 20, top: 30, bottom: 40 },
  tooltip: { trigger: 'axis' },
  legend: { data: ['PV', 'UV'], bottom: 0 },
  xAxis: {
    type: 'category',
    data: daily.value.map((p) => p.date.slice(5)),
    axisLine: { lineStyle: { color: '#c2cdd2' } },
    axisLabel: { color: '#48565e' },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { color: '#edf1f2' } },
    axisLabel: { color: '#48565e' },
  },
  series: [
    {
      name: 'PV',
      type: 'line',
      smooth: true,
      data: daily.value.map((p) => p.pv),
      itemStyle: { color: '#0f766e' },
      areaStyle: { color: 'rgba(15, 118, 110, 0.12)' },
    },
    {
      name: 'UV',
      type: 'line',
      smooth: true,
      data: daily.value.map((p) => p.uv),
      itemStyle: { color: '#1d4ed8' },
      areaStyle: { color: 'rgba(29, 78, 216, 0.08)' },
    },
  ],
}));

const hourlyOption = computed(() => ({
  grid: { left: 40, right: 20, top: 30, bottom: 30 },
  tooltip: { trigger: 'axis' },
  xAxis: {
    type: 'category',
    data: hourly.value.map((p) => `${p.hour}:00`),
    axisLine: { lineStyle: { color: '#c2cdd2' } },
    axisLabel: { color: '#48565e' },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { color: '#edf1f2' } },
    axisLabel: { color: '#48565e' },
  },
  series: [
    {
      name: 'PV',
      type: 'bar',
      data: hourly.value.map((p) => p.pv),
      itemStyle: { color: '#0f766e', borderRadius: [3, 3, 0, 0] },
      barMaxWidth: 20,
    },
  ],
}));

function pieOption(items: LabelValue[]) {
  return {
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        type: 'pie',
        radius: ['40%', '65%'],
        center: ['50%', '45%'],
        data: items.map((item) => ({ name: labelOf(item.label), value: item.value })),
        label: { formatter: '{b} {d}%' },
      },
    ],
  };
}

function horizontalBarOption(items: LabelValue[], color: string) {
  return {
    grid: { left: 90, right: 40, top: 10, bottom: 30 },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: '#edf1f2' } },
      axisLabel: { color: '#48565e' },
    },
    yAxis: {
      type: 'category',
      data: items.map((item) => labelOf(item.label)),
      axisLine: { lineStyle: { color: '#c2cdd2' } },
      axisLabel: { color: '#48565e' },
    },
    series: [
      {
        type: 'bar',
        data: items.map((item) => item.value),
        itemStyle: { color, borderRadius: [0, 3, 3, 0] },
        barMaxWidth: 16,
      },
    ],
  };
}

const deviceOption = computed(() => pieOption(deviceStats.value?.device ?? []));
const osOption = computed(() => horizontalBarOption(deviceStats.value?.os ?? [], '#1d4ed8'));
const browserOption = computed(() => horizontalBarOption(deviceStats.value?.browser ?? [], '#0f766e'));
const geoOption = computed(() => pieOption(geo.value));

onMounted(async () => {
  try {
    links.value = (await listLinks()).items;
    // 支持从链接列表/概览页 ?linkId= 直达指定链接的统计
    const wanted = Number(route.query.linkId);
    if (wanted > 0 && links.value.some((l) => l.ID === wanted)) {
      selectedLinkId.value = wanted;
    } else {
      selectedLinkId.value = 0; // 默认全部链接
      // 初始值保持 0 时 watch 不会触发，需主动拉取聚合数据
      await load();
    }
  } catch (err) {
    message.error(err instanceof Error ? err.message : '加载链接列表失败');
  }
});

watch(selectedLinkId, async () => {
  await load();
});

// P1：统计日期范围（趋势图与分布图适用；最长 90 天由后端兜底）
const dateRange = ref<[number, number] | null>(null);
watch(dateRange, () => {
  load();
});

function rangeQuery(): { start?: string; end?: string } {
  if (!dateRange.value) return {};
  const fmt = (n: number) => new Date(n).toISOString().slice(0, 10);
  return { start: fmt(dateRange.value[0]), end: fmt(dateRange.value[1]) };
}

async function load() {
  const id = selectedLinkId.value;
  loading.value = true;
  try {
    const range = rangeQuery();
    const rangeParams = new URLSearchParams(Object.entries(range).filter(([, v]) => v)).toString();
    const suffix = rangeParams ? `?${rangeParams}` : '';
    if (id === 0) {
      // 全部链接：用聚合接口
      const [s, d, h] = await Promise.all([
        getOverviewSummary(),
        getOverviewDaily(suffix),
        getOverviewHourly(),
      ]);
      summary.value = s;
      daily.value = d;
      hourly.value = h;
      deviceStats.value = null; // 聚合暂无设备/地域数据
      geo.value = [];
    } else {
      const [s, d, h, dev, g] = await Promise.all([
        getSummaryStats(id),
        getDailyStats(id, suffix),
        getHourlyStats(id),
        getDeviceStats(id, suffix),
        getGeoStats(id, suffix),
      ]);
      summary.value = s;
      daily.value = d;
      hourly.value = h;
      deviceStats.value = dev;
      geo.value = g;
    }
  } catch (err) {
    message.error(err instanceof Error ? err.message : '加载统计失败');
  } finally {
    loading.value = false;
  }
}

// P1：重置该链接统计（管理员）
const resetting = ref(false);
async function doReset() {
  if (selectedLinkId.value <= 0 || resetting.value) return;
  resetting.value = true;
  try {
    await resetLinkStats(selectedLinkId.value);
    message.success('统计已重置（访问日志保留）');
    await load();
  } catch (err) {
    message.error(err instanceof Error ? err.message : '重置失败');
  } finally {
    resetting.value = false;
  }
}
</script>

<template>
  <div class="stats-page">
    <NCard>
      <template #header>
        <div class="card-head">
          <div>
            <strong>访问统计</strong>
            <p class="muted">按链接查看访问量与时段分布</p>
          </div>
          <div class="card-actions">
            <NSelect
              v-model:value="selectedLinkId"
              :options="linkOptions"
              placeholder="选择链接"
              filterable
              style="width: 280px"
            />
            <NDatePicker
              v-model:value="dateRange"
              type="daterange"
              clearable
              :is-date-disabled="(ts: number) => ts > Date.now()"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              style="width: 260px"
            />
            <NButton :loading="loading" @click="load">
              <template #icon><RefreshCw :size="16" /></template>
            </NButton>
            <NPopconfirm v-if="(auth.isSuperAdmin || auth.user?.role === 'admin') && selectedLinkId > 0" @positive-click="doReset">
              <template #trigger>
                <NButton type="error" ghost :loading="resetting" title="清空 Redis 计数与聚合表，原始访问日志保留">
                  <template #icon><RotateCcw :size="16" /></template>
                  重置统计
                </NButton>
              </template>
              确认重置该链接的全部统计数据？此操作不可恢复（访问日志保留，仅统计口径归零）。
            </NPopconfirm>
          </div>
        </div>
      </template>

      <template>
        <NGrid :cols="4" :x-gap="16" :y-gap="16" responsive="screen" item-responsive style="margin-bottom: var(--space-5)">
          <NGridItem span="4 s:2 m:1">
            <NCard size="small" embedded>
              <NSkeleton v-if="loading && !summary" height="60px" :sharp="false" />
              <NStatistic v-else label="累计 PV" :value="summary?.total_pv ?? 0" />
            </NCard>
          </NGridItem>
          <NGridItem span="4 s:2 m:1">
            <NCard size="small" embedded>
              <NSkeleton v-if="loading && !summary" height="60px" :sharp="false" />
              <NStatistic v-else label="累计 UV" :value="summary?.total_uv ?? 0" />
            </NCard>
          </NGridItem>
          <NGridItem span="4 s:2 m:1">
            <NCard size="small" embedded>
              <NSkeleton v-if="loading && !summary" height="60px" :sharp="false" />
              <NStatistic v-else label="今日 PV" :value="summary?.today_pv ?? 0" />
            </NCard>
          </NGridItem>
          <NGridItem span="4 s:2 m:1">
            <NCard size="small" embedded>
              <NSkeleton v-if="loading && !summary" height="60px" :sharp="false" />
              <NStatistic v-else label="今日 UV" :value="summary?.today_uv ?? 0" />
            </NCard>
          </NGridItem>
        </NGrid>

        <NGrid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
          <NGridItem span="2 m:1">
            <NCard title="每日访问（近 30 天）" size="small" embedded>
              <VChart :option="dailyOption" autoresize style="height: 320px" />
            </NCard>
          </NGridItem>
          <NGridItem span="2 m:1">
            <NCard title="今日 24 小时分布" size="small" embedded>
              <VChart :option="hourlyOption" autoresize style="height: 320px" />
            </NCard>
          </NGridItem>
          <NGridItem span="2 m:1">
            <NCard title="设备分布" size="small" embedded>
              <NEmpty
                v-if="(deviceStats?.device?.length ?? 0) === 0"
                description="暂无数据，今日访问将在次日汇总"
                style="padding: 60px 0"
              />
              <VChart v-else :option="deviceOption" autoresize style="height: 320px" />
            </NCard>
          </NGridItem>
          <NGridItem span="2 m:1">
            <NCard title="操作系统" size="small" embedded>
              <NEmpty
                v-if="(deviceStats?.os?.length ?? 0) === 0"
                description="暂无数据，今日访问将在次日汇总"
                style="padding: 60px 0"
              />
              <VChart v-else :option="osOption" autoresize style="height: 320px" />
            </NCard>
          </NGridItem>
          <NGridItem span="2 m:1">
            <NCard title="浏览器" size="small" embedded>
              <NEmpty
                v-if="(deviceStats?.browser?.length ?? 0) === 0"
                description="暂无数据，今日访问将在次日汇总"
                style="padding: 60px 0"
              />
              <VChart v-else :option="browserOption" autoresize style="height: 320px" />
            </NCard>
          </NGridItem>
          <NGridItem span="2 m:1">
            <NCard title="地域分布" size="small" embedded>
              <NEmpty v-if="geo.length === 0" description="暂无数据" style="padding: 60px 0" />
              <VChart v-else :option="geoOption" autoresize style="height: 320px" />
            </NCard>
          </NGridItem>
        </NGrid>
      </template>
    </NCard>
  </div>
</template>

<style scoped>
.stats-page {
  display: grid;
  gap: var(--space-6);
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
</style>
