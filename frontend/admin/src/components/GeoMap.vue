<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { echarts } from '../echarts';
import VChart from 'vue-echarts';
import { NButton, NButtonGroup, NEmpty, NSkeleton, NStatistic } from 'naive-ui';
import type { LabelValue } from '../api';
import { toChinaMapName, toWorldMapName } from '../geoNames';

const props = defineProps<{
  country: LabelValue[];
  province: LabelValue[];
  city?: LabelValue[];
  loading?: boolean;
}>();

type MapScope = 'world' | 'china';
const scope = ref<MapScope>('world');
const mapsReady = ref(false);
const mapError = ref('');

let worldJson: unknown = null;
let chinaJson: unknown = null;

async function ensureMaps() {
  if (worldJson && chinaJson) {
    mapsReady.value = true;
    return;
  }
  try {
    const [w, c] = await Promise.all([
      fetch('/maps/world.json').then((r) => {
        if (!r.ok) throw new Error('world');
        return r.json();
      }),
      fetch('/maps/china.json').then((r) => {
        if (!r.ok) throw new Error('china');
        return r.json();
      }),
    ]);
    worldJson = w;
    chinaJson = c;
    echarts.registerMap('world', w as never);
    echarts.registerMap('china', c as never);
    mapsReady.value = true;
    mapError.value = '';
  } catch {
    mapError.value = '地图资源加载失败';
    mapsReady.value = false;
  }
}

const mapName = computed(() => (scope.value === 'world' ? 'world' : 'china'));

interface MapPoint {
  name: string;
  value: number;
  sourceLabel: string;
}

const mapData = computed<MapPoint[]>(() => {
  const items = scope.value === 'world' ? props.country : props.province;
  const mapper = scope.value === 'world' ? toWorldMapName : toChinaMapName;
  const merged = new Map<string, MapPoint>();
  for (const item of items) {
    const name = mapper(item.label);
    if (!name) continue;
    const prev = merged.get(name);
    if (prev) prev.value += item.value;
    else merged.set(name, { name, value: item.value, sourceLabel: item.label });
  }
  return [...merged.values()];
});

const maxValue = computed(() => mapData.value.reduce((m, p) => Math.max(m, p.value), 0));

function filteredRank(items: LabelValue[] | undefined): LabelValue[] {
  return (items ?? []).filter((i) => i.label && i.label !== '未知' && i.label !== 'Reserved' && i.label !== '直接访问' || i.label === '直接访问');
}

const countryRank = computed(() =>
  (props.country ?? [])
    .filter((i) => i.label && i.label !== '未知' && i.label !== 'Reserved')
    .slice(0, 12),
);
const provinceRank = computed(() => (props.province ?? []).slice(0, 12));
const cityRank = computed(() => (props.city ?? []).slice(0, 12));

const activeRank = computed(() => (scope.value === 'world' ? countryRank.value : provinceRank.value));
const rankTitle = computed(() => (scope.value === 'world' ? '国家 / 地区 TOP' : '省份 TOP'));

const rankTotal = computed(() => activeRank.value.reduce((s, i) => s + i.value, 0) || 1);
const countryTotal = computed(() => props.country.reduce((s, i) => s + i.value, 0));
const provinceTotal = computed(() => props.province.reduce((s, i) => s + i.value, 0));

const option = computed(() => {
  const max = maxValue.value || 1;
  return {
    tooltip: {
      trigger: 'item',
      formatter: (params: { name: string; value?: number | string; data?: MapPoint }) => {
        const v = Number(params.value ?? 0) || 0;
        const label = params.data?.sourceLabel ?? params.name;
        return `${label}<br/>访问量 ${v}`;
      },
    },
    visualMap: {
      min: 0,
      max,
      left: 12,
      bottom: 12,
      calculable: false,
      orient: 'horizontal',
      itemWidth: 14,
      itemHeight: 90,
      text: ['高', '低'],
      textStyle: { color: '#48565e', fontSize: 12 },
      inRange: { color: ['#e8eef3', '#f0c15b', '#d4922a'] },
      seriesIndex: 0,
    },
    series: [
      {
        type: 'map',
        map: mapName.value,
        roam: false,
        selectedMode: false,
        emphasis: { label: { show: false }, itemStyle: { areaColor: '#7eb6d9' } },
        itemStyle: {
          areaColor: '#f4f6f8',
          borderColor: '#d5dde2',
          borderWidth: 0.6,
        },
        label: { show: false },
        data: mapData.value,
      },
    ],
  };
});

const hasData = computed(() => mapData.value.length > 0 || activeRank.value.length > 0);

onMounted(ensureMaps);
watch(scope, () => {
  void ensureMaps();
});
</script>

<template>
  <div class="geo-map">
    <div class="geo-toolbar">
      <div class="geo-summary">
        <NStatistic label="有地域记录" :value="countryTotal" />
        <NStatistic label="国内访问" :value="provinceTotal" />
      </div>
      <NButtonGroup size="medium">
        <NButton :type="scope === 'world' ? 'primary' : 'default'" @click="scope = 'world'">世界</NButton>
        <NButton :type="scope === 'china' ? 'primary' : 'default'" @click="scope = 'china'">中国</NButton>
      </NButtonGroup>
    </div>

    <NEmpty v-if="!hasData && !loading" description="所选时段暂无地域数据" style="padding: 80px 0" />
    <div v-else-if="mapError" class="geo-fallback">
      <NEmpty :description="mapError" style="padding: 80px 0" />
    </div>
    <div v-else class="geo-body">
      <div class="geo-chart">
        <NSkeleton v-if="loading || !mapsReady" height="100%" :sharp="false" style="height: 520px" />
        <VChart v-else :option="option" autoresize class="geo-canvas" />
      </div>
      <div class="geo-side">
        <section class="geo-rank">
          <div class="geo-rank-head">
            <span>{{ rankTitle }}</span>
            <span>PV</span>
          </div>
          <div v-if="activeRank.length === 0" class="geo-rank-empty">暂无数据</div>
          <div v-else class="geo-rank-list">
            <div v-for="item in activeRank" :key="item.label" class="geo-rank-row">
              <span class="geo-rank-label" :title="item.label">{{ item.label }}</span>
              <span class="geo-rank-meta">
                <span class="geo-rank-pct">{{ ((item.value / rankTotal) * 100).toFixed(1) }}%</span>
                <span class="geo-rank-num">{{ item.value }}</span>
              </span>
            </div>
          </div>
        </section>
        <section v-if="scope === 'china' && cityRank.length" class="geo-rank">
          <div class="geo-rank-head">
            <span>城市 TOP</span>
            <span>PV</span>
          </div>
          <div class="geo-rank-list">
            <div v-for="item in cityRank" :key="item.label" class="geo-rank-row">
              <span class="geo-rank-label" :title="item.label">{{ item.label }}</span>
              <span class="geo-rank-meta">
                <span class="geo-rank-num">{{ item.value }}</span>
              </span>
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.geo-map {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-height: 560px;
}

.geo-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.geo-summary {
  display: flex;
  gap: var(--space-6);
}

.geo-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: var(--space-5);
  align-items: stretch;
  flex: 1;
  min-height: 520px;
}

.geo-chart {
  min-width: 0;
  min-height: 520px;
  border: 1px solid var(--border-color, #e5eaee);
  border-radius: 10px;
  background: #fafbfc;
  overflow: hidden;
}

.geo-canvas {
  width: 100%;
  height: 520px;
}

.geo-side {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}

.geo-rank {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  flex: 1;
  min-height: 0;
}

.geo-rank-head {
  display: flex;
  justify-content: space-between;
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-color-2, #48565e);
  padding-bottom: 6px;
  border-bottom: 1px solid var(--border-color, #e5eaee);
}

.geo-rank-empty {
  font-size: var(--font-size-sm);
  color: var(--text-color-3, #8a97a0);
  padding: var(--space-4) 0;
}

.geo-rank-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: auto;
}

.geo-rank-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
}

.geo-rank-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-color-1, #1f2933);
}

.geo-rank-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.geo-rank-pct {
  font-variant-numeric: tabular-nums;
  color: var(--text-color-3, #8a97a0);
  min-width: 44px;
  text-align: right;
}

.geo-rank-num {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: var(--text-color-1, #1f2933);
  min-width: 32px;
  text-align: right;
}

@media (max-width: 960px) {
  .geo-body {
    grid-template-columns: 1fr;
    min-height: auto;
  }

  .geo-canvas,
  .geo-chart {
    height: 360px;
    min-height: 360px;
  }

  .geo-map {
    min-height: auto;
  }
}
</style>
