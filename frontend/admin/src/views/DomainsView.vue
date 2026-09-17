<script setup lang="ts">
import ViewportTable from '../components/ViewportTable.vue';
import { computed, h, onMounted, reactive, ref } from 'vue';
import { Pencil, Plus, RefreshCw, Trash2 } from '@lucide/vue';
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
  NPopconfirm,
  NSelect,
  NTag,
  useMessage,
  type DataTableColumns,
  type FormInst,
  type FormRules,
} from 'naive-ui';
import {
  createDomain,
  deleteDomain,
  listDomains,
  listLandingPages,
  updateDomain,
  type DomainItem,
  type LandingPageItem,
} from '../api';
import { useAuthStore } from '../stores/auth';
import { useDomainStore } from '../stores/domain';

const message = useMessage();
const auth = useAuthStore();
const domainStore = useDomainStore();

const items = ref<DomainItem[]>([]);
const landingPages = ref<LandingPageItem[]>([]);
const loading = ref(false);
const keyword = ref('');
const selectedType = ref<string | null>(null);
const filteredItems = computed(() => {
  const query = keyword.value.trim().toLowerCase();
  return items.value.filter((item) =>
    (!selectedType.value || item.Type === selectedType.value) &&
    (!query || `${item.Scheme}://${item.Host} ${item.Remark || ''}`.toLowerCase().includes(query)),
  );
});
const activeCount = computed(() => items.value.filter((item) => item.Status === 'active').length);
const disabledCount = computed(() => items.value.filter((item) => item.Status === 'disabled').length);
function clearFilters() {
  keyword.value = '';
  selectedType.value = null;
}
const showModal = ref(false);
const saving = ref(false);
const modalError = ref('');
const editingId = ref<number | null>(null);
const formRef = ref<FormInst | null>(null);

const form = reactive({
  host: '',
  type: 'entry' as 'entry' | 'transit' | 'landing',
  scheme: 'https' as 'http' | 'https',
  remark: '',
  homeMode: 'default' as 'default' | 'redirect' | 'landing',
  homeRedirectUrl: '',
  homeLandingPageId: null as number | null,
});

const canWrite = computed(() => auth.isSuperAdmin || auth.user?.role === 'admin');

const typeOptions = [
  { label: '入口域', value: 'entry' },
  { label: '中转域', value: 'transit' },
  { label: '落地域', value: 'landing' },
];

const schemeOptions = [
  { label: 'HTTPS', value: 'https' },
  { label: 'HTTP', value: 'http' },
];

const homeModeOptions = [
  { label: '跟随全局', value: 'default' },
  { label: '自动跳转', value: 'redirect' },
  { label: '展示落地页', value: 'landing' },
];

const typeLabelMap: Record<string, string> = { entry: '入口', transit: '中转', landing: '落地页' };
const typeTagMap: Record<string, 'info' | 'warning' | 'success'> = { entry: 'success', transit: 'warning', landing: 'info' };
const statusLabelMap: Record<string, string> = { active: '正常', disabled: '已停用' };
const statusTypeMap: Record<string, 'success' | 'default'> = { active: 'success', disabled: 'default' };
const homeModeLabelMap: Record<string, string> = { default: '跟随全局', redirect: '自动跳转', landing: '落地页' };

const homeLandingOptions = computed(() =>
  landingPages.value
    .filter((p) => p.Template === 'custom' || p.Template === 'redirect_notice')
    .map((p) => ({
      label: `${p.Title}（${p.Template === 'custom' ? '自定义' : '跳转提示'}）`,
      value: p.ID,
    })),
);

const rules: FormRules = {
  host: [
    {
      required: true,
      validator: (_rule, value: string) => {
        if (!value) return new Error('请输入域名');
        try {
          if (/[\s/?#@\\]/.test(value.trim())) throw new Error();
          const url = new URL(`http://${value.trim()}`);
          if (!url.hostname || (!url.hostname.includes('.') && url.hostname !== 'localhost' && !url.hostname.includes(':'))) throw new Error();
        } catch { return new Error('请输入域名或本机地址，可带端口，例如 localhost:18080'); }
        return true;
      },
      trigger: ['blur', 'input'],
    },
  ],
};

function sanitizeRedirectUrl(raw: string): string {
  let s = raw.trim().replace(/^["']+|["']+$/g, '').trim();
  if (!s) return '';
  try {
    const u = new URL(s);
    if (u.protocol !== 'http:' && u.protocol !== 'https:') return '';
    return u.toString();
  } catch {
    return '';
  }
}

function homeSummary(row: DomainItem): string {
  if (row.HomeMode === 'redirect') {
    return row.HomeRedirectURL ? `跳转 ${row.HomeRedirectURL}` : '跳转（回退全局）';
  }
  if (row.HomeMode === 'landing') {
    const page = landingPages.value.find((p) => p.ID === row.HomeLandingPageID);
    return page ? `落地页：${page.Title}` : `落地页 #${row.HomeLandingPageID ?? '—'}`;
  }
  return '跟随全局';
}

const columns: DataTableColumns<DomainItem> = [
  { title: '域名', key: 'Host', render: (row) => h('code', {}, `${row.Scheme}://${row.Host}`) },
  {
    title: '用途',
    key: 'Type',
    width: 90,
    render: (row) => h(NTag, { type: typeTagMap[row.Type] || 'default', size: 'small' }, () => typeLabelMap[row.Type] || row.Type),
  },
  {
    title: '首页',
    key: 'HomeMode',
    width: 180,
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'muted' }, homeSummary(row)),
  },
  { title: '备注', key: 'Remark', ellipsis: { tooltip: true }, render: (row) => row.Remark || h('span', { class: 'muted' }, '—') },
  {
    title: '状态',
    key: 'Status',
    width: 90,
    render: (row) =>
      h(NTag, { type: statusTypeMap[row.Status] || 'default', size: 'small', round: true }, () => statusLabelMap[row.Status] || row.Status),
  },
  {
    title: '操作',
    key: 'actions',
    width: 110,
    render: (row) =>
      canWrite.value
        ? h('div', { style: 'display:flex;gap:4px;align-items:center' }, [
            h(
              NButton,
              { text: true, size: 'small', onClick: () => openEdit(row) },
              { icon: () => h(NIcon, { size: 14 }, { default: () => h(Pencil) }) },
            ),
            h(
              NPopconfirm,
              { onPositiveClick: () => removeDomain(row), positiveText: '删除', negativeText: '取消' },
              {
                trigger: () =>
                  h(
                    NButton,
                    { text: true, size: 'small', type: 'error' },
                    { icon: () => h(NIcon, { size: 14 }, { default: () => h(Trash2) }) },
                  ),
                default: () => `确认删除域名 ${row.Host}？删除后相关短链将无法访问。`,
              },
            ),
          ])
        : h('span', { class: 'muted' }, '—'),
  },
];

onMounted(async () => {
  await refresh();
  try {
    landingPages.value = (await listLandingPages()).items;
  } catch {
    // 落地页列表仅用于首页下拉，失败不阻塞域名页
  }
});

async function refresh() {
  loading.value = true;
  try {
    items.value = (await listDomains()).items;
    domainStore.invalidate();
  } catch (err) {
    message.error(messageOf(err));
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingId.value = null;
  Object.assign(form, {
    host: '',
    type: 'entry',
    scheme: 'https',
    remark: '',
    homeMode: 'default',
    homeRedirectUrl: '',
    homeLandingPageId: null,
  });
  modalError.value = '';
  showModal.value = true;
}

function openEdit(row: DomainItem) {
  editingId.value = row.ID;
  Object.assign(form, {
    host: row.Host,
    type: row.Type,
    scheme: row.Scheme,
    remark: row.Remark || '',
    homeMode: row.HomeMode || 'default',
    homeRedirectUrl: row.HomeRedirectURL || '',
    homeLandingPageId: row.HomeLandingPageID ?? null,
  });
  modalError.value = '';
  showModal.value = true;
}

function buildHomePayload() {
  const mode = form.homeMode;
  const redirectUrl = mode === 'redirect' || mode === 'landing' ? sanitizeRedirectUrl(form.homeRedirectUrl) : '';
  if ((mode === 'redirect') && form.homeRedirectUrl.trim() && !redirectUrl) {
    throw new Error('首页自动跳转地址无效，需为 http/https 完整地址');
  }
  if (mode === 'landing' && !form.homeLandingPageId) {
    throw new Error('请选择作为首页的落地页（仅自定义 / 跳转提示）');
  }
  return {
    home_mode: mode,
    home_redirect_url: redirectUrl || undefined,
    home_landing_page_id: mode === 'landing' ? form.homeLandingPageId : null,
  };
}

async function submit() {
  modalError.value = '';
  try {
    await formRef.value?.validate();
  } catch {
    return;
  }
  saving.value = true;
  try {
    const home = buildHomePayload();
    if (editingId.value == null) {
      await createDomain({
        host: form.host.trim(),
        type: form.type,
        scheme: form.scheme,
        remark: form.remark || undefined,
        ...home,
      });
      message.success(`域名 ${form.host} 已添加`);
    } else {
      await updateDomain(editingId.value, {
        host: form.host.trim(),
        type: form.type,
        scheme: form.scheme,
        remark: form.remark || undefined,
        ...home,
      });
      message.success(`域名 ${form.host} 已更新`);
    }
    showModal.value = false;
    await refresh();
  } catch (err) {
    modalError.value = messageOf(err);
  } finally {
    saving.value = false;
  }
}

async function removeDomain(item: DomainItem) {
  try {
    await deleteDomain(item.ID);
    message.success(`已删除 ${item.Host}`);
    await refresh();
  } catch (err) {
    message.error(messageOf(err));
  }
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : '操作失败';
}
</script>

<template>
  <div class="page-view">
    <div class="domain-summary" aria-label="域名概况">
      <NCard><span class="muted">全部域名</span><strong>{{ loading ? '—' : items.length }}</strong></NCard>
      <NCard><span class="muted">正常域名</span><strong>{{ loading ? '—' : activeCount }}</strong></NCard>
      <NCard><span class="muted">已停用</span><strong>{{ loading ? '—' : disabledCount }}</strong></NCard>
    </div>
    <NCard>
    <template #header>
      <div class="card-head">
        <div>
          <strong>域名</strong>
          <p class="muted">管理入口、中转和落地页域名</p>
        </div>
        <div class="card-actions">
          <NButton :loading="loading" aria-label="刷新域名" @click="refresh">
            <template #icon><RefreshCw :size="16" /></template>
          </NButton>
          <NButton v-if="canWrite" type="primary" @click="openCreate">
            <template #icon><Plus :size="16" /></template>
            添加域名
          </NButton>
        </div>
      </div>
    </template>

    <div class="domain-filters">
      <NInput v-model:value="keyword" clearable placeholder="搜索域名或备注" aria-label="搜索域名或备注" />
      <NSelect v-model:value="selectedType" clearable :options="typeOptions" placeholder="全部用途" aria-label="筛选域名用途" />
      <span class="muted" aria-live="polite">共 {{ filteredItems.length }} 个域名</span>
    </div>
    <ViewportTable :columns="columns" :data="filteredItems" :loading="loading" :pagination="{ pageSize: 20 }" :row-key="(row: DomainItem) => row.ID" :scroll-x="780" :bordered="false" size="small">
      <template #empty>
        <NEmpty :description="items.length ? '没有符合条件的域名' : '尚未配置域名'">
          <template #extra>
            <NButton v-if="items.length" size="small" @click="clearFilters">清空筛选</NButton>
            <NButton v-else-if="canWrite" type="primary" size="small" @click="openCreate">添加第一个域名</NButton>
          </template>
        </NEmpty>
      </template>
    </ViewportTable>
  </NCard>

  <NModal v-model:show="showModal" preset="card" :title="editingId == null ? '添加域名' : '编辑域名'" style="width: min(640px, calc(100vw - 32px))" :mask-closable="false" :closable="!saving" :close-on-esc="!saving">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="top">
      <NFormItem label="域名（不含协议和路径）" path="host">
        <NInput v-model:value="form.host" placeholder="go.example.com" :disabled="editingId != null" />
      </NFormItem>
      <div class="form-row">
        <NFormItem label="用途">
          <NSelect v-model:value="form.type" :options="typeOptions" />
        </NFormItem>
        <NFormItem label="协议">
          <NSelect v-model:value="form.scheme" :options="schemeOptions" />
        </NFormItem>
      </div>
      <p class="type-description">{{ form.type === 'entry' ? '入口域名：用于生成对外分享的短链接。' : form.type === 'transit' ? '中转域名：作为入口与落地页之间的可选中间层。' : '落地域名：用于展示系统生成的落地页。' }}</p>
      <NFormItem label="备注">
        <NInput v-model:value="form.remark" placeholder="（可选）" />
      </NFormItem>

      <NFormItem label="首页展示">
        <NSelect v-model:value="form.homeMode" :options="homeModeOptions" />
      </NFormItem>
      <p class="type-description">
        跟随全局：使用系统设置的首页；自动跳转：访问该域名根路径时跳转；展示落地页：根路径渲染指定落地页。
      </p>
      <NFormItem v-if="form.homeMode === 'redirect' || form.homeMode === 'landing'" label="跳转地址">
        <NInput v-model:value="form.homeRedirectUrl" placeholder="https://example.com/ （redirect 必填；landing 的跳转提示页用作目标）" />
      </NFormItem>
      <NFormItem v-if="form.homeMode === 'landing'" label="首页落地页">
        <NSelect
          v-model:value="form.homeLandingPageId"
          :options="homeLandingOptions"
          placeholder="仅支持自定义 / 跳转提示模板"
          clearable
        />
      </NFormItem>

      <NAlert v-if="modalError" type="error" :show-icon="true" style="margin-bottom: var(--space-3)">{{ modalError }}</NAlert>
      <NAlert type="info" :show-icon="true">DNS 解析与证书需要在反向代理层完成。编辑时域名不可改。</NAlert>
    </NForm>

    <template #footer>
      <div style="display: flex; justify-content: flex-end; gap: var(--space-2)">
        <NButton :disabled="saving" @click="showModal = false">取消</NButton>
        <NButton type="primary" :loading="saving" @click="submit">{{ editingId == null ? '添加' : '保存' }}</NButton>
      </div>
    </template>
  </NModal>
  </div>
</template>

<style scoped>
.domain-summary { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--space-4); }
.domain-summary strong { display: block; margin-top: var(--space-2); font-size: var(--font-size-3xl); color: var(--color-text-primary); }
.domain-filters { display: grid; grid-template-columns: minmax(180px, 1fr) 160px auto; align-items: center; gap: var(--space-3); margin-bottom: var(--space-5); }
.type-description { color: var(--color-text-secondary); margin-bottom: var(--space-5); }
@media (max-width: 600px) {
  .domain-summary, .domain-filters, .form-row { grid-template-columns: 1fr; }
}
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

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}
@media (max-width: 600px) {
  .form-row { grid-template-columns: 1fr; }
}
</style>
