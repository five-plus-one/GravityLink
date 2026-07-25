<script setup lang="ts">
import { computed, ref } from 'vue';
import { BarChart3, LoaderCircle, Search } from '@lucide/vue';
import { getDailyStats, getHourlyStats, getSummaryStats, type DailyPoint, type HourlyPoint, type SummaryStats } from '../api';

const linkId = ref(1);
const loading = ref(false);
const error = ref('');
const summary = ref<SummaryStats | null>(null);
const daily = ref<DailyPoint[]>([]);
const hourly = ref<HourlyPoint[]>([]);
const dailyMax = computed(() => Math.max(1, ...daily.value.map((point) => point.pv)));
const hourlyMax = computed(() => Math.max(1, ...hourly.value.map((point) => point.pv)));
async function load() {
  loading.value = true; error.value = '';
  try { [summary.value, daily.value, hourly.value] = await Promise.all([getSummaryStats(linkId.value), getDailyStats(linkId.value), getHourlyStats(linkId.value)]); }
  catch (err) { error.value = err instanceof Error ? err.message : '加载统计失败'; } finally { loading.value = false; }
}
</script>
<template><div class="view">
  <header class="view-header"><div><h1>统计</h1><p>按链接查看访问量和时段分布。</p></div>
    <form class="query-form" @submit.prevent="load"><label>链接 ID<input v-model.number="linkId" min="1" type="number" /></label><button class="primary icon-text" :disabled="loading"><LoaderCircle v-if="loading" class="spin" :size="16" /><Search v-else :size="16" />查询</button></form></header>
  <p v-if="error" class="error banner">{{ error }}</p>
  <div v-if="!summary" class="blank-state"><BarChart3 :size="28" /><strong>选择链接查看数据</strong><p>输入链接 ID 后查询累计、每日和小时访问。</p></div>
  <template v-else><section class="metrics"><div class="metric-block"><span>累计 PV</span><strong>{{ summary.total_pv }}</strong></div><div class="metric-block"><span>累计 UV</span><strong>{{ summary.total_uv }}</strong></div><div class="metric-block"><span>今日 PV</span><strong>{{ summary.today_pv }}</strong></div><div class="metric-block"><span>今日 UV</span><strong>{{ summary.today_uv }}</strong></div></section>
    <div class="chart-grid"><section class="surface chart-surface"><div class="surface-head"><h2>每日访问</h2></div><div class="bar-chart"><div v-for="point in daily" :key="point.date" class="bar-item"><div class="bar" :style="{ height: `${Math.max(3, point.pv / dailyMax * 100)}%` }" :title="`${point.date}: ${point.pv}`"></div><small>{{ point.date.slice(5) }}</small></div></div></section>
    <section class="surface chart-surface"><div class="surface-head"><h2>小时分布</h2></div><div class="bar-chart"><div v-for="point in hourly" :key="point.hour" class="bar-item"><div class="bar secondary" :style="{ height: `${Math.max(3, point.pv / hourlyMax * 100)}%` }" :title="`${point.hour}:00: ${point.pv}`"></div><small>{{ point.hour }}</small></div></div></section></div>
  </template>
</div></template>
