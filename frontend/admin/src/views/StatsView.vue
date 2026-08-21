<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { use } from 'echarts/core';
import { BarChart, LineChart } from 'echarts/charts';
import { DataZoomComponent, GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import VChart from 'vue-echarts';
import { BarChart3, RefreshCw } from '@lucide/vue';
import { NButton, NCard, NEmpty, NGrid, NGridItem, NSelect, NSkeleton, NStatistic, useMessage } from 'naive-ui';
import { getDailyStats, getHourlyStats, getSummaryStats, listLinks, type DailyPoint, type HourlyPoint, type LinkItem, type SummaryStats } from '../api';

use([BarChart, LineChart, GridComponent, TooltipComponent, LegendComponent, DataZoomComponent, CanvasRenderer]);

const message = useMessage();

const links = ref<LinkItem[]>([]);
const selectedLinkId = ref<number | null>(null);
const loading = ref(false);
const summary = ref<SummaryStats | null>(null);
const daily = ref<DailyPoint[]>([]);
const hourly = ref<HourlyPoint[]>([]);

const linkOptions = computed(() =>
  links.value.map((link) => ({
    label: `${link.Code}${link.Title ? ` · ${link.Title}` : ''}`,
    value: link.ID,
  })),
);

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

onMounted(async () => {
  try {
    links.value = (await listLinks()).items;
    if (links.value.length > 0) {
      selectedLinkId.value = links.value[0].ID;
    }
  } catch (err) {
    message.error(err instanceof Error ? err.message : '加载链接列表失败');
  }
});

watch(selectedLinkId, async (id) => {
  if (id === null) return;
  await load();
});

async function load() {
  if (selectedLinkId.value === null) return;
  loading.value = true;
  try {
    const id = selectedLinkId.value;
    [summary.value, daily.value, hourly.value] = await Promise.all([getSummaryStats(id), getDailyStats(id), getHourlyStats(id)]);
  } catch (err) {
    message.error(err instanceof Error ? err.message : '加载统计失败');
  } finally {
    loading.value = false;
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
            <NButton :loading="loading" :disabled="selectedLinkId === null" @click="load">
              <template #icon><RefreshCw :size="16" /></template>
            </NButton>
          </div>
        </div>
      </template>

      <NEmpty v-if="!selectedLinkId" description="选择链接查看数据">
        <template #icon><BarChart3 :size="32" /></template>
      </NEmpty>

      <template v-else>
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

        <NGrid :cols="2" :x-gap="16" responsive="screen" item-responsive>
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
