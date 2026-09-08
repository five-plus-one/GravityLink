<script setup lang="ts">
import { ref, watch } from 'vue';
import { NAlert, NButton, NModal, useMessage } from 'naive-ui';
import type { LinkItem } from '../api';

/**
 * 创建成功分享面板（P0 方案 A）：
 * 打开即自动复制短链，短链/原文一键复制、二维码预览与下载、连建下一条。
 */
const props = defineProps<{ show: boolean; link: LinkItem | null; url: string; targetUrl?: string | null }>();
const emit = defineEmits<{ (e: 'update:show', v: boolean): void; (e: 'recreate'): void }>();

const message = useMessage();
const qr = ref('');
const autoCopied = ref(false);
const localAddr = ref(false);

watch(
  () => props.show,
  async (visible) => {
    if (!visible || !props.url) return;
    qr.value = '';
    autoCopied.value = false;
    try {
      const parsed = new URL(props.url);
      localAddr.value = ['localhost', '127.0.0.1', '[::1]'].includes(parsed.hostname);
    } catch {
      localAddr.value = true;
    }
    try {
      const { default: QRCode } = await import('qrcode');
      qr.value = await QRCode.toDataURL(props.url, { width: 480, margin: 3, errorCorrectionLevel: 'M' });
    } catch {
      qr.value = '';
    }
    try {
      await navigator.clipboard.writeText(props.url);
      autoCopied.value = true;
    } catch {
      autoCopied.value = false;
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
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="链接创建成功"
    style="width: min(460px, 94vw)"
    @update:show="emit('update:show', $event)"
  >
    <div class="share-panel">
      <div class="auto-copy-tip" :class="{ ok: autoCopied }">
        {{ autoCopied ? '✓ 短链已自动复制到剪贴板，可直接粘贴分享' : '短链复制失败，请点击下方按钮手动复制' }}
      </div>

      <div class="copy-row">
        <span class="row-label primary">短链</span>
        <span class="row-url strong">{{ url }}</span>
        <button class="row-btn solid" @click="copy(url, '短链已复制')">复制</button>
      </div>
      <div v-if="targetUrl" class="copy-row">
        <span class="row-label">原文</span>
        <span class="row-url">{{ targetUrl }}</span>
        <button class="row-btn" @click="copy(targetUrl, '原文已复制')">复制</button>
      </div>

      <div class="qr-area">
        <img v-if="qr" :src="qr" alt="入口二维码" />
        <div v-else class="qr-fallback">二维码生成失败，可复制短链后用外部工具生成</div>
        <div class="qr-meta">
          <div class="meta-line">类型：{{ ({ short: '短链接', channel: '渠道链接', liveqr: '活码' } as Record<string, string>)[link?.Type || ''] || link?.Type }}</div>
          <div class="meta-line">名称：{{ link?.Title || '未命名' }}</div>
          <div style="display:flex;gap:8px;margin-top:10px;flex-wrap:wrap">
            <NButton v-if="qr" size="small" tag="a" :href="qr" :download="`${link?.Code || 'gravitylink'}-qr.png`">下载二维码 PNG</NButton>
            <NButton size="small" @click="emit('recreate')">再建一条</NButton>
            <NButton size="small" type="primary" @click="emit('update:show', false)">完成</NButton>
          </div>
        </div>
      </div>

      <NAlert v-if="localAddr" type="warning" :show-icon="true">
        当前是本机测试地址，手机扫码前请绑定可公网访问的入口域名。
      </NAlert>
    </div>
  </NModal>
</template>

<style scoped>
.share-panel { display: grid; gap: var(--space-3); }
.auto-copy-tip {
  padding: 10px 12px;
  border-radius: var(--radius-md, 8px);
  background: var(--color-bg-subtle, #f4f7f8);
  color: var(--color-text-secondary, #6b7f88);
  font-size: 13px;
}
.auto-copy-tip.ok { background: rgba(21, 128, 61, 0.08); color: var(--green, #15803d); }
.copy-row {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--color-bg-subtle, #f4f7f8);
  border: 1px solid var(--color-border, #e3eaed);
  border-radius: 8px;
  padding: 10px 12px;
}
.row-label {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--color-text-secondary, #6b7f88);
  padding: 2px 8px;
  border-radius: 4px;
  background: #fff;
  border: 1px solid var(--color-border, #e3eaed);
}
.row-label.primary { color: var(--color-primary, #0f766e); border-color: var(--color-primary, #0f766e); }
.row-url { flex: 1; font-family: Consolas, Menlo, monospace; font-size: 12.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.row-url.strong { font-weight: 600; }
.row-btn { flex-shrink: 0; border: 1px solid var(--color-border, #e3eaed); background: #fff; color: var(--color-text-secondary, #6b7f88); border-radius: 6px; padding: 5px 12px; font-size: 12px; cursor: pointer; }
.row-btn.solid { background: var(--color-primary, #0f766e); color: #fff; border-color: var(--color-primary, #0f766e); }
.qr-area { display: flex; gap: 16px; align-items: flex-start; }
.qr-area img { width: 132px; height: 132px; border-radius: 6px; border: 1px solid var(--color-border, #e3eaed); }
.qr-fallback { width: 132px; height: 132px; display: grid; place-items: center; font-size: 12px; color: var(--color-text-secondary, #6b7f88); text-align: center; background: var(--color-bg-subtle, #f4f7f8); border-radius: 6px; padding: 8px; }
.qr-meta { flex: 1; font-size: 12.5px; color: var(--color-text-secondary, #6b7f88); display: grid; gap: 4px; align-content: start; }
</style>
