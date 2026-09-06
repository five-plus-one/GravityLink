<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { Activity, Eye, Users, History, RefreshCw } from '@lucide/vue';
import { NButton, NCard, NEmpty, NSelect, NSkeleton, NStatistic } from 'naive-ui';
import '../echarts';
import VChart from 'vue-echarts';
import type { LinkItem } from '../api';
import { fillDailyWindow, useLinkTraffic } from '../composables/useLinkTraffic';

const props = defineProps<{ links: LinkItem[] }>();
const selected = ref<number | null>(null);
const days = ref(7);
const { summary, daily, hourly, loading, error, load } = useLinkTraffic();
const options = computed(() => props.links.map((link) => ({ label: `${link.Title || '未命名'} · ${link.Code}`, value: link.ID })));
watch(() => props.links, (links) => {
  if (!links.some((link) => link.ID === selected.value)) selected.value = links[0]?.ID ?? null;
}, { immediate: true });
watch(selected, (id) => { void load(id); }, { immediate: true });
const metrics = computed(() => [
  { label: '今日访问', value: summary.value?.today_pv, icon: Eye, note: '访问次数 · PV' },
  { label: '今日访客', value: summary.value?.today_uv, icon: Users, note: '独立访客估算 · UV' },
  { label: '累计访问', value: summary.value?.total_pv, icon: Activity, note: '累计访问次数 · PV' },
  { label: '昨日访问', value: summary.value?.yesterday_pv, icon: History, note: '昨日访问次数 · PV' },
]);
const windowPoints = computed(() => fillDailyWindow(daily.value, days.value));
const hasDaily = computed(() => windowPoints.value.some((point) => point.pv > 0 || point.uv > 0));
const hasHourly = computed(() => hourly.value.some((point) => point.pv > 0));
const tokens = getComputedStyle(document.documentElement);
const primary = tokens.getPropertyValue('--color-primary').trim();
const secondary = tokens.getPropertyValue('--color-success').trim();
const text = tokens.getPropertyValue('--color-text-secondary').trim();
const border = tokens.getPropertyValue('--color-border').trim();
function chart(labels: string[], series: { name: string; values: number[]; color: string }[]) {
  return {
    tooltip: { trigger: 'axis', renderMode: 'richText' },
    legend: { bottom: 0, textStyle: { color: text } },
    grid: { left: 16, right: 20, top: 24, bottom: 56, containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: labels, axisLine: { lineStyle: { color: border } }, axisLabel: { color: text } },
    yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { color: border } }, axisLabel: { color: text } },
    series: series.map((item) => ({ name: item.name, type: 'line', showSymbol: false, data: item.values, itemStyle: { color: item.color }, areaStyle: { opacity: 0.08 } })),
  };
}
const dailyOption = computed(() => chart(windowPoints.value.map((p) => p.date.slice(5)), [
  { name: '访问次数 PV', values: windowPoints.value.map((p) => p.pv), color: primary },
  { name: '访客 UV', values: windowPoints.value.map((p) => p.uv), color: secondary },
]));
const hourlyOption = computed(() => chart(hourly.value.map((p) => `${String(p.hour).padStart(2, '0')}:00`), [
  { name: '访问次数 PV', values: hourly.value.map((p) => p.pv), color: primary },
]));
</script>

<template>
  <section class="traffic" aria-label="链接访问看板">
    <div class="traffic-toolbar">
      <div><h2>访问看板</h2><p class="muted">当前所选链接的访问数据</p></div>
      <div class="traffic-controls">
        <NSelect v-model:value="selected" class="link-select" :options="options" filterable placeholder="选择链接" aria-label="选择统计链接" />
        <NButton :loading="loading" :disabled="selected === null" aria-label="刷新访问统计" @click="load(selected)"><template #icon><RefreshCw :size="16" /></template></NButton>
      </div>
    </div>
    <NCard v-if="selected === null"><NEmpty description="创建链接后，即可在这里查看访问数据" /></NCard>
    <template v-else>
      <NCard v-if="error" role="alert">
        <NEmpty description="访问统计加载失败"><template #extra><p class="error-detail">{{ error }}</p><NButton @click="load(selected)">重新加载</NButton></template></NEmpty>
      </NCard>
      <template v-else>
        <div class="traffic-metrics">
          <NCard v-for="metric in metrics" :key="metric.label">
            <NSkeleton v-if="loading" height="84px" :sharp="false" />
            <template v-else><NStatistic :label="metric.label" :value="metric.value ?? '—'"><template #prefix><span class="metric-icon"><component :is="metric.icon" :size="22" /></span></template></NStatistic><p class="muted metric-note">{{ metric.note }}</p></template>
          </NCard>
        </div>
        <div class="traffic-charts">
          <NCard title="访问趋势">
            <template #header-extra><NSelect v-model:value="days" :options="[{ label: '近 7 天', value: 7 }, { label: '近 30 天', value: 30 }]" style="width: 120px" aria-label="趋势时间范围" /></template>
            <NSkeleton v-if="loading" height="320px" :sharp="false" />
            <VChart v-else-if="hasDaily" :option="dailyOption" autoresize class="chart" />
            <NEmpty v-else :description="`近 ${days} 天暂无访问`" class="chart-empty" />
          </NCard>
          <NCard title="今日小时分布">
            <template #header-extra><span class="muted">00:00–23:00</span></template>
            <NSkeleton v-if="loading" height="320px" :sharp="false" />
            <VChart v-else-if="hasHourly" :option="hourlyOption" autoresize class="chart" />
            <NEmpty v-else description="今日暂无访问" class="chart-empty" />
          </NCard>
        </div>
      </template>
    </template>
  </section>
</template>

<style scoped>
.traffic { display: grid; gap: var(--space-5); min-width: 0; }
.traffic-toolbar, .traffic-controls { display: flex; align-items: center; justify-content: space-between; gap: var(--space-3); }
h2 { margin: 0; font-size: var(--font-size-xl); }
.link-select { width: 300px; }
.traffic-metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: var(--space-4); }
.traffic-charts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--space-5); }
.metric-icon { width: 44px; height: 44px; display: inline-grid; place-items: center; border-radius: var(--radius-md); color: var(--color-primary); background: var(--color-primary-soft); }
.metric-note { margin-top: var(--space-2); font-size: var(--font-size-sm); }
.chart, .chart-empty { height: 320px; }
.chart-empty { display: flex; justify-content: center; }
.error-detail { margin-bottom: var(--space-3); color: var(--color-text-secondary); }
@media (max-width: 1100px) { .traffic-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); } .traffic-charts { grid-template-columns: 1fr; } }
@media (max-width: 600px) { .traffic-toolbar { align-items: stretch; flex-direction: column; } .traffic-metrics { grid-template-columns: 1fr; } .link-select { width: auto; flex: 1; min-width: 0; } }
</style>
