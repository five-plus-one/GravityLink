import { onScopeDispose, ref } from 'vue';
import {
  getDailyStats, getHourlyStats, getSummaryStats,
  getOverviewDaily, getOverviewHourly, getOverviewSummary,
  type DailyPoint, type HourlyPoint, type SummaryStats,
} from '../api';

export function useLinkTraffic() {
  const summary = ref<SummaryStats | null>(null);
  const daily = ref<DailyPoint[]>([]);
  const hourly = ref<HourlyPoint[]>([]);
  const loading = ref(false);
  const error = ref('');
  let requestID = 0;
  onScopeDispose(() => { requestID++; });

  // id=0 表示全部链接聚合；null 表示未选择
  async function load(id: number | null) {
    const current = ++requestID;
    summary.value = null;
    daily.value = [];
    hourly.value = [];
    error.value = '';
    loading.value = id !== null;
    if (id === null) return;
    try {
      const [nextSummary, nextDaily, nextHourly] = id === 0
        ? await Promise.all([getOverviewSummary(), getOverviewDaily(), getOverviewHourly()])
        : await Promise.all([getSummaryStats(id), getDailyStats(id), getHourlyStats(id)]);
      if (current !== requestID) return;
      summary.value = nextSummary;
      daily.value = nextDaily ?? [];
      hourly.value = nextHourly ?? [];
    } catch (err) {
      if (current !== requestID) return;
      error.value = err instanceof Error ? err.message : '加载访问统计失败';
    } finally {
      if (current === requestID) loading.value = false;
    }
  }
  return { summary, daily, hourly, loading, error, load };
}

// The API includes today's point even when its count is zero, anchoring the
// window to the server's reporting date instead of the browser's timezone.
export function fillDailyWindow(points: DailyPoint[], days: number): DailyPoint[] {
  const sorted = [...points].sort((a, b) => a.date.localeCompare(b.date));
  const end = sorted.at(-1)?.date;
  if (!end) return [];
  const last = new Date(`${end}T00:00:00Z`);
  const byDate = new Map(sorted.map((point) => [point.date, point]));
  return Array.from({ length: days }, (_, index) => {
    const date = new Date(last);
    date.setUTCDate(last.getUTCDate() - days + index + 1);
    const key = date.toISOString().slice(0, 10);
    return byDate.get(key) ?? { date: key, pv: 0, uv: 0 };
  });
}
