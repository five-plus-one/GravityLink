import { effectScope } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getDailyStats, getHourlyStats, getSummaryStats, type SummaryStats } from '../api';
import { fillDailyWindow, useLinkTraffic } from './useLinkTraffic';

vi.mock('../api', () => ({ getDailyStats: vi.fn(), getHourlyStats: vi.fn(), getSummaryStats: vi.fn() }));
const summary: SummaryStats = { today_pv: 5, today_uv: 2, total_pv: 20, total_uv: 10, yesterday_pv: 3 };
beforeEach(() => {
  vi.resetAllMocks();
  vi.mocked(getSummaryStats).mockResolvedValue(summary);
  vi.mocked(getDailyStats).mockResolvedValue([]);
  vi.mocked(getHourlyStats).mockResolvedValue([]);
});
function setup() {
  const scope = effectScope();
  return { scope, traffic: scope.run(useLinkTraffic)! };
}
describe('访问看板请求状态', () => {
  it('快速切换后，旧请求不能覆盖新链接统计', async () => {
    let resolveOld!: (value: SummaryStats) => void;
    vi.mocked(getSummaryStats).mockImplementationOnce(() => new Promise((resolve) => { resolveOld = resolve; }));
    const { scope, traffic } = setup();
    const old = traffic.load(1);
    await traffic.load(2);
    resolveOld({ ...summary, today_pv: 999 });
    await old;
    expect(traffic.summary.value?.today_pv).toBe(5);
    expect(traffic.loading.value).toBe(false);
    scope.stop();
  });
  it('失败清除旧指标并允许重试，零访问是成功结果', async () => {
    const { scope, traffic } = setup();
    await traffic.load(1);
    vi.mocked(getDailyStats).mockRejectedValueOnce(new Error('连接中断'));
    await traffic.load(2);
    expect(traffic.summary.value).toBeNull();
    expect(traffic.error.value).toBe('连接中断');
    vi.mocked(getSummaryStats).mockResolvedValue({ ...summary, today_pv: 0 });
    await traffic.load(2);
    expect(traffic.error.value).toBe('');
    expect(traffic.summary.value?.today_pv).toBe(0);
    scope.stop();
  });
  it('清空选择后丢弃仍在途的请求', async () => {
    let resolve!: (value: SummaryStats) => void;
    vi.mocked(getSummaryStats).mockImplementationOnce(() => new Promise((done) => { resolve = done; }));
    const { scope, traffic } = setup();
    const pending = traffic.load(1);
    await traffic.load(null);
    resolve(summary);
    await pending;
    expect(traffic.summary.value).toBeNull();
    expect(traffic.loading.value).toBe(false);
    scope.stop();
  });
});
describe('日趋势时间窗口', () => {
  it('跨月补零，排序且不改变接口数据', () => {
    const points = [{ date: '2026-03-02', pv: 2, uv: 1 }, { date: '2026-02-28', pv: 4, uv: 2 }];
    expect(fillDailyWindow(points, 3)).toEqual([
      { date: '2026-02-28', pv: 4, uv: 2 },
      { date: '2026-03-01', pv: 0, uv: 0 },
      { date: '2026-03-02', pv: 2, uv: 1 },
    ]);
    expect(points[0].date).toBe('2026-03-02');
  });
  it('没有服务端日期时保持空态', () => { expect(fillDailyWindow([], 7)).toEqual([]); });
});
