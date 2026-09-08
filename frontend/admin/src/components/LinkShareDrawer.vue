<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { NAlert, NButton, NDrawer, NDrawerContent, NSkeleton, useMessage } from 'naive-ui';
import { getSummaryStats, type LinkItem, type SummaryStats } from '../api';

/**
 * 行内分享抽屉（P0 方案 B）：
 * 点击「分享」打开，聚合 短链复制 / 二维码下载 / 今日与累计数据摘要 / 统计与编辑入口。
 */
const props = defineProps<{ show: boolean; link: LinkItem | null; url: string }>();
const emit = defineEmits<{ (e: 'update:show', v: boolean): void; (e: 'edit', link: LinkItem): void }>();

const router = useRouter();
const message = useMessage();
const qr = ref('');
const summary = ref<SummaryStats | null>(null);
const summaryLoading = ref(false);

const typeLabels: Record<string, string> = { short: '短链接', channel: '渠道链接', liveqr: '活码' };

const display = computed(() => {
  if (!props.link) return null;
  return {
    type: typeLabels[props.link.Type] || props.link.Type,
    title: props.link.Title || '未命名',
    createdAt: new Date(props.link.CreatedAt).toLocaleString('zh-CN', { hour12: false }),
  };
});

watch(
  () => props.show,
  async (visible) => {
    if (!visible || !props.link) return;
    qr.value = '';
    summary.value = null;
    if (props.url) {
      try {
        const { default: QRCode } = await import('qrcode');
        qr.value = await QRCode.toDataURL(props.url, { width: 480, margin: 3, errorCorrectionLevel: 'M' });
      } catch {
        qr.value = '';
      }
    }
    summaryLoading.value = true;
    try {
      summary.value = await getSummaryStats(props.link.ID);
    } catch {
      summary.value = null;
    } finally {
      summaryLoading.value = false;
    }
  },
);

async function copy(text: string, tip = '已复制') {
  try {
    await navigator.clipboard.writeText(text);
    message.success(tip);
  } catch {
    message.error('复制失败，请手动复制');
  }
}

function gotoStats() {
  emit('update:show', false);
  router.push({ path: '/stats', query: { linkId: String(props.link?.ID ?? '') } });
}
</script>

<template>
  <NDrawer :show="show" :width="360" placement="right" @update:show="emit('update:show', $event)">
    <NDrawerContent :title="`${link?.Code || ''} · ${display?.title || ''}`" closable>
      <div class="drawer-body">
        <div class="meta-grid">
          <div><span class="meta-label">类型</span>{{ display?.type }}</div>
          <div><span class="meta-label">创建时间</span>{{ display?.createdAt }}</div>
        </div>

        <div class="copy-row">
          <span class="row-url">{{ url || '入口域不可用' }}</span>
          <button class="row-btn solid" :disabled="!url" @click="copy(url)">复制</button>
        </div>

        <div class="qr-row">
          <img v-if="qr" :src="qr" alt="入口二维码" />
          <div v-else class="qr-fallback">二维码不可用</div>
          <div class="qr-actions">
            <NButton v-if="qr" size="small" tag="a" :href="qr" :download="`${link?.Code || 'gravitylink'}-qr.png`">下载 PNG</NButton>
            <NButton size="small" :disabled="!url" tag="a" :href="url || undefined" target="_blank" rel="noopener noreferrer">打开链接</NButton>
          </div>
        </div>

        <div class="summary-card">
          <div class="summary-title">访问数据</div>
          <NSkeleton v-if="summaryLoading" text :repeat="2" />
          <template v-else-if="summary">
            <div class="summary-grid">
              <div class="summary-item"><b>{{ summary.today_pv }}</b><span>今日 PV</span></div>
              <div class="summary-item"><b>{{ summary.today_uv }}</b><span>今日 UV</span></div>
              <div class="summary-item"><b>{{ summary.total_pv }}</b><span>累计 PV</span></div>
              <div class="summary-item"><b>{{ summary.yesterday_pv }}</b><span>昨日 PV</span></div>
            </div>
          </template>
          <div v-else class="summary-empty">统计数据加载失败</div>
        </div>

        <NAlert v-if="link?.Type === 'liveqr'" type="info" :show-icon="true">
          活码的分发目标与扫码明细请在列表「二维码配置」中查看。
        </NAlert>
      </div>

      <template #footer>
        <div class="drawer-footer">
          <NButton size="small" @click="gotoStats">查看完整统计</NButton>
          <NButton size="small" @click="emit('edit', link!)">编辑</NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped>
.drawer-body { display: grid; gap: var(--space-4); }
.meta-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-2); font-size: 13px; }
.meta-label { display: block; font-size: 12px; color: var(--color-text-secondary, #6b7f88); margin-bottom: 2px; }
.copy-row {
  display: flex; align-items: center; gap: 8px;
  background: var(--color-bg-subtle, #f4f7f8);
  border: 1px solid var(--color-border, #e3eaed);
  border-radius: 8px; padding: 10px 12px;
}
.row-url { flex: 1; font-family: Consolas, Menlo, monospace; font-size: 12.5px; word-break: break-all; }
.row-btn { flex-shrink: 0; border: none; background: var(--color-primary, #0f766e); color: #fff; border-radius: 6px; padding: 5px 14px; font-size: 12px; cursor: pointer; }
.row-btn:disabled { opacity: .5; cursor: not-allowed; }
.qr-row { display: flex; gap: 14px; align-items: flex-start; }
.qr-row img { width: 116px; height: 116px; border-radius: 6px; border: 1px solid var(--color-border, #e3eaed); }
.qr-fallback { width: 116px; height: 116px; display: grid; place-items: center; font-size: 12px; color: var(--color-text-secondary, #6b7f88); background: var(--color-bg-subtle, #f4f7f8); border-radius: 6px; }
.qr-actions { display: grid; gap: 8px; }
.summary-card { border: 1px solid var(--color-border, #e3eaed); border-radius: 10px; padding: 14px; }
.summary-title { font-size: 13px; font-weight: 600; margin-bottom: 10px; }
.summary-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.summary-item b { font-size: 20px; display: block; }
.summary-item span { font-size: 12px; color: var(--color-text-secondary, #6b7f88); }
.summary-empty { font-size: 12px; color: var(--color-text-secondary, #6b7f88); }
.drawer-footer { display: flex; gap: var(--space-2); justify-content: flex-end; }
</style>
