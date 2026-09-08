<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { BarChart3, Copy, ExternalLink, Pencil, Plus, RefreshCw, Search, Send, Trash2 } from '@lucide/vue';
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NDatePicker,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NPopconfirm,
  NRadioButton,
  NRadioGroup,
  NSelect,
  NSwitch,
  NTag,
  useMessage,
  type DataTableColumns,
  type FormInst,
  type FormRules,
} from 'naive-ui';
import {
  createLink,
  deleteLink,
  listLinks,
  listLandingPages,
  type LandingPageItem,
  updateLink,
  type LinkItem,
} from '../api';
import { useDomainStore } from '../stores/domain';
import { useAuthStore } from '../stores/auth';
import TargetManager from '../components/TargetManager.vue';
import MaterialPicker from '../components/MaterialPicker.vue';
import EntryQRCode from '../components/EntryQRCode.vue';
import BatchAdd from '../components/BatchAdd.vue';
import ShareSuccessModal from '../components/ShareSuccessModal.vue';
import LinkShareDrawer from '../components/LinkShareDrawer.vue';

const route = useRoute();
const router = useRouter();
const message = useMessage();
const domains = useDomainStore();
const auth = useAuthStore();

const items = ref<LinkItem[]>([]);
const landingPages = ref<LandingPageItem[]>([]);
const landingPageId = ref<number | null>(null);
const loading = ref(false);
const query = ref('');
const statusFilter = ref<string | null>(null);
const typeFilter = ref<string | null>(null);

const showModal = ref(false);
const modalMode = ref<'create' | 'edit'>('create');
const saving = ref(false);
const modalError = ref('');
const formRef = ref<FormInst | null>(null);
const editingId = ref<number | null>(null);

// P0：创建成功分享面板 + 行内分享抽屉
const shareLink = ref<LinkItem | null>(null);
const showShare = ref(false);
const drawerLink = ref<LinkItem | null>(null);
const showDrawer = ref(false);

const form = reactive({
  type: 'short' as 'short' | 'channel' | 'liveqr',
  title: '',
  code: '',
  entryDomainId: null as number | null,
  targetUrl: '',
  accessRule: 'none' as 'none' | 'wechat' | 'ios' | 'android' | 'mobile' | 'pc',
  strategyMode: 'round_robin' as 'round_robin' | 'weighted',
  targets: [{ label: '', url: '', weight: 1, scanLimit: null as number | null }],
  expireAt: null as number | null,
  status: 'active' as 'active' | 'disabled',
});

const statusOptions = [
  { label: '正常', value: 'active' },
  { label: '已停用', value: 'disabled' },
];

const accessRuleOptions = [
  { label: '不限制', value: 'none' },
  { label: '仅微信内', value: 'wechat' },
  { label: '仅 iOS', value: 'ios' },
  { label: '仅 Android', value: 'android' },
  { label: '仅手机', value: 'mobile' },
  { label: '仅电脑', value: 'pc' },
];

const typeOptions = [
  { label: '短链接', value: 'short' },
  { label: '渠道链接', value: 'channel' },
  { label: '活码', value: 'liveqr' },
];

const strategyOptions = [
  { label: '顺序阈值模式', value: 'round_robin' },
  { label: '权重模式', value: 'weighted' },
];

const statusTypeMap: Record<string, 'success' | 'warning' | 'default' | 'error'> = {
  active: 'success',
  pending: 'warning',
  disabled: 'default',
  expired: 'error',
};
const statusLabelMap: Record<string, string> = {
  active: '正常',
  pending: '待审核',
  disabled: '已停用',
  expired: '已过期',
};
const typeLabelMap: Record<string, string> = {
  short: '短链接',
  channel: '渠道链接',
  liveqr: '活码',
};

const entryDomainOptions = computed(() =>
  domains.entryDomains.map((d) => ({ label: d.Host, value: d.ID })),
);

const canWrite = computed(() => auth.isSuperAdmin || auth.user?.role === 'admin');

const rules: FormRules = {
  entryDomainId: [{ required: true, type: 'number', message: '请选择入口域名', trigger: ['change'] }],
  targetUrl: [
    {
      required: true,
      validator: (_rule, value: string) => {
        if (form.type === 'liveqr') return true;
        if (!value) return new Error('请输入目标 URL');
        try {
          new URL(value);
          return true;
        } catch {
          return new Error('URL 格式不正确');
        }
      },
      trigger: ['blur', 'input'],
    },
  ],
  targets: [
    {
      required: true,
      validator: () => {
        if (form.type !== 'liveqr') return true;
        if (!form.targets.length) return new Error('请至少添加一个二维码目标');
        for (const item of form.targets) {
          const url = item.url.trim();
          // 站内素材相对路径与 http(s) 绝对地址均可（与后端口径一致）
          const isRelative = url.startsWith('/uploads/') && !url.includes('..');
          let ok = isRelative;
          if (!ok) {
            try {
              const parsed = new URL(url);
              ok = parsed.protocol === 'http:' || parsed.protocol === 'https:';
            } catch {
              ok = false;
            }
          }
          if (!ok) return new Error(`目标“${item.label || '未命名'}”需为 http(s) 地址或站内素材路径`);
        }
        return true;
      },
      trigger: ['blur'],
    },
  ],
};

const filtered = computed(() => {
  const value = query.value.trim().toLowerCase();
  return items.value.filter((item) => {
    if (statusFilter.value && item.Status !== statusFilter.value) return false;
    if (typeFilter.value && item.Type !== typeFilter.value) return false;
    if (!value) return true;
    return `${item.Code} ${item.Title || ''} ${item.TargetURL || ''}`.toLowerCase().includes(value);
  });
});

const shortUrlBase = computed(() => {
  const map = new Map<number, string>();
  domains.entryDomains.filter((d) => d.Status === 'active').forEach((d) => map.set(d.ID, `${d.Scheme}://${d.Host}`));
  return map;
});

function shortUrlOf(row: LinkItem): string {
  if (row.PublicURL !== undefined) return row.PublicURL;
  const base = shortUrlBase.value.get(row.EntryDomainID);
  return base ? `${base}/${encodeURIComponent(row.Code)}` : '';
}

const columns: DataTableColumns<LinkItem> = [
  {
    title: '短码',
    key: 'Code',
    width: 140,
    render: (row) =>
      h('div', { style: 'display:flex;align-items:center;gap:6px' }, [
        h('code', {}, row.Code),
        h(
          NButton,
          { text: true, size: 'tiny', disabled: !shortUrlOf(row), onClick: () => copyShortUrl(row) },
          { icon: () => h(NIcon, { size: 14 }, { default: () => h(Copy) }) },
        ),
      ]),
  },
  { title: '访问地址', key: 'url', ellipsis: { tooltip: true }, render: (row) => shortUrlOf(row) || '入口域不可用' },
  { title: '名称', key: 'Title', render: (row) => row.Title || h('span', { class: 'muted' }, '未命名') },
  {
    title: '类型',
    key: 'Type',
    width: 100,
    render: (row) => typeLabelMap[row.Type] || row.Type,
  },
  {
    title: '目标',
    key: 'TargetURL',
    ellipsis: { tooltip: true },
    render: (row) => row.TargetURL || h('span', { class: 'muted' }, '动态路由'),
  },
  {
    title: '创建时间',
    key: 'CreatedAt',
    width: 170,
    render: (row) => new Date(row.CreatedAt).toLocaleString('zh-CN', { hour12: false }),
  },
  {
    title: '状态',
    key: 'Status',
    width: 90,
    render: (row) => {
      if (row.Status === 'expired' || !canWrite.value) {
        return h(NTag, { type: statusTypeMap[row.Status] || 'default', size: 'small', round: true }, () => statusLabelMap[row.Status] || row.Status);
      }
      return h(NSwitch, {
        size: 'small',
        value: row.Status === 'active',
        onUpdateValue: (value: boolean) => toggleStatus(row, value),
      });
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 300,
    render: (row) =>
      h('div', { style: 'display:flex;gap:4px;align-items:center' }, [
        h(
          NButton,
          { text: true, size: 'small', title: '分享（链接+二维码+数据）', disabled: !shortUrlOf(row), onClick: () => { drawerLink.value = row; showDrawer.value = true; } },
          { icon: () => h(NIcon, { size: 14 }, { default: () => h(Send) }) },
        ),
        h(
          NButton,
          { text: true, size: 'small', title: '查看统计', onClick: () => router.push({ path: '/stats', query: { linkId: String(row.ID) } }) },
          { icon: () => h(NIcon, { size: 14 }, { default: () => h(BarChart3) }) },
        ),
        h(EntryQRCode, {url:shortUrlOf(row),name:row.Code}),
        canWrite.value && row.Type === 'liveqr' ? h(TargetManager, { linkId: row.ID, origin: shortUrlOf(row) ? new URL(shortUrlOf(row)).origin : '' }) : null,
        h(
          NButton,
          { text: true, size: 'small', tag: 'a', disabled: !shortUrlOf(row), href: shortUrlOf(row) || undefined, target: '_blank', rel: 'noopener noreferrer' },
          { icon: () => h(NIcon, { size: 14 }, { default: () => h(ExternalLink) }) },
        ),
        canWrite.value
          ? h(
              NButton,
              { text: true, size: 'small', onClick: () => openEdit(row) },
              { icon: () => h(NIcon, { size: 14 }, { default: () => h(Pencil) }) },
            )
          : null,
        canWrite.value
          ? h(
              NPopconfirm,
              {
                onPositiveClick: () => removeLink(row),
                positiveText: '删除',
                negativeText: '取消',
              },
              {
                trigger: () =>
                  h(
                    NButton,
                    { text: true, size: 'small', type: 'error' },
                    { icon: () => h(NIcon, { size: 14 }, { default: () => h(Trash2) }) },
                  ),
                default: () => `确认删除链接 ${row.Code}？此操作不可恢复。`,
              },
            )
          : null,
      ]),
  },
];

onMounted(async () => {
  await refresh();
  if (canWrite.value) {
    try { await domains.refresh(); landingPages.value = (await listLandingPages()).items; }
    catch (err) { message.error(messageOf(err)); }
  }
  if (route.query.create === '1' && canWrite.value) openCreate();
});

async function refresh() {
  loading.value = true;
  try {
    items.value = (await listLinks()).items;
  } catch (err) {
    message.error(messageOf(err));
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  landingPageId.value = null;
  modalMode.value = 'create';
  editingId.value = null;
  Object.assign(form, {
    type: 'short',
    title: '',
    code: '',
    entryDomainId: null,
    targetUrl: '',
    strategyMode: 'round_robin',
    targets: [{ label: '', url: '', weight: 1, scanLimit: null }],
    expireAt: null,
    status: 'active',
  });
  modalError.value = '';
  showModal.value = true;
}

function openEdit(row: LinkItem) {
  modalMode.value = 'edit';
  editingId.value = row.ID;
  Object.assign(form, {
    type: row.Type as typeof form.type,
    title: row.Title || '',
    code: row.Code,
    entryDomainId: null,
    targetUrl: row.TargetURL || '',
    strategyMode: 'round_robin',
    targets: [{ label: '', url: '', weight: 1, scanLimit: null }],
    expireAt: null,
    status: (row.Status === 'disabled' ? 'disabled' : 'active') as 'active' | 'disabled',
  });
  modalError.value = '';
  showModal.value = true;
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
    if (modalMode.value === 'create') {
      const page = landingPages.value.find(p => p.ID === landingPageId.value && p.Template === 'liveqr');
      if (form.type === 'liveqr' && !page) throw new Error('请选择活码落地页；没有可选项时请先创建 liveqr 落地页');
      const created = await createLink({
        landing_page_id: form.type === 'liveqr' ? page?.ID : undefined,
        landing_domain_id: form.type === 'liveqr' ? page?.DomainID : undefined,
        type: form.type,
        code: form.code || undefined,
        title: form.title || undefined,
        entry_domain_id: form.entryDomainId!,
        target_url: form.type === 'liveqr' ? '' : form.targetUrl,
        access_rule: form.accessRule !== 'none' ? form.accessRule : undefined,
        expire_at: form.expireAt ? new Date(form.expireAt).toISOString() : undefined,
        strategy:
          form.type === 'liveqr'
            ? {
                mode: form.strategyMode,
                targets: form.targets.map((item) => ({
                  label: item.label.trim() || undefined,
                  target_url: item.url.trim(),
                  weight: form.strategyMode === 'weighted' ? item.weight : 1,
                  scan_limit: item.scanLimit ?? undefined,
                })),
              }
            : undefined,
      });
      showModal.value = false;
      await refresh();
      // P0 方案 A：创建成功直接进分享面板（短链已自动复制）
      shareLink.value = created;
      showShare.value = true;
      message.success('链接已创建');
    } else if (editingId.value !== null) {
      await updateLink(editingId.value, {
        title: form.title,
        target_url: form.type === 'liveqr' ? undefined : form.targetUrl,
        expire_at: form.expireAt ? new Date(form.expireAt).toISOString() : null,
        status: form.status,
      });
      message.success('链接已更新');
    }
    showModal.value = false;
    await refresh();
  } catch (err) {
    // 关键修复：错误显示在弹窗内，而不是弹窗背后的页面
    modalError.value = messageOf(err);
  } finally {
    saving.value = false;
  }
}

async function removeLink(row: LinkItem) {
  try {
    await deleteLink(row.ID);
    message.success(`已删除 ${row.Code}`);
    await refresh();
  } catch (err) {
    message.error(messageOf(err));
  }
}

async function copyShortUrl(row: LinkItem) {
  if (!shortUrlOf(row)) {
    message.error('入口域不可用，请检查域名配置');
    return;
  }
  try {
    await navigator.clipboard.writeText(shortUrlOf(row));
    message.success(`已复制 ${shortUrlOf(row)}`);
  } catch {
    message.error('复制失败，请手动复制');
  }
}

// P0：行内启停（旧版 switch 交互回归）
async function toggleStatus(row: LinkItem, active: boolean) {
  const next = active ? 'active' : 'disabled';
  const prev = row.Status;
  row.Status = next;
  try {
    await updateLink(row.ID, { status: next });
    message.success(`${row.Code} 已${active ? '启用' : '停用'}`);
  } catch (err) {
    row.Status = prev;
    message.error(messageOf(err));
  }
}

// 编辑弹窗中只读展示入口域名（换绑能力由后端支持后开放）
const editingEntryDomainHost = computed(() => {
  if (modalMode.value !== 'edit' || editingId.value === null) return '';
  const row = items.value.find((i) => i.ID === editingId.value);
  if (!row) return '';
  return domains.items.find((d) => d.ID === row.EntryDomainID)?.Host || `#${row.EntryDomainID}`;
});

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : '操作失败';
}

function addTarget() {
  form.targets.push({ label: '', url: '', weight: 1, scanLimit: null });
}

function removeTarget(index: number) {
  if (form.targets.length === 1) return;
  form.targets.splice(index, 1);
}
</script>

<template>
  <div class="page-view">
    <NCard>
    <template #header>
      <div class="card-head">
        <div>
          <strong>链接</strong>
          <p class="muted">{{ items.length }} 条记录，{{ items.filter((i) => i.Status === 'active').length }} 条可访问</p>
        </div>
        <div class="card-actions">
          <BatchAdd v-if="canWrite" :domains="domains.items" @saved="refresh"/>
          <NInput v-model:value="query" placeholder="搜索短码、名称或目标" clearable style="width: 260px">
            <template #prefix><Search :size="14" /></template>
          </NInput>
          <NSelect v-model:value="typeFilter" :options="typeOptions" placeholder="类型" clearable style="width: 130px" />
          <NSelect v-model:value="statusFilter" :options="statusOptions" placeholder="状态" clearable style="width: 120px" />
          <NButton :loading="loading" @click="refresh">
            <template #icon><RefreshCw :size="16" /></template>
          </NButton>
          <NButton v-if="canWrite" type="primary" @click="openCreate">
            <template #icon><Plus :size="16" /></template>
            创建链接
          </NButton>
        </div>
      </div>
    </template>

    <NDataTable
      :columns="columns"
      :data="filtered"
      :loading="loading"
      :pagination="{ pageSize: 20, showSizePicker: true, pageSizes: [10, 20, 50, 100] }"
      :bordered="false"
      size="small"
    >
      <template #empty>
        <NEmpty description="没有匹配的链接">
          <template v-if="canWrite" #extra>
            <NButton type="primary" size="small" @click="openCreate">创建第一条链接</NButton>
          </template>
        </NEmpty>
      </template>
    </NDataTable>
  </NCard>

  <NModal
    v-model:show="showModal"
    preset="card"
    :title="modalMode === 'create' ? '创建链接' : '编辑链接'"
    :style="{ width: form.type === 'liveqr' && modalMode === 'create' ? 'min(920px, 94vw)' : 'min(620px, 94vw)' }"
    :mask-closable="false"
  >
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="top" require-mark-placement="right">
      <NFormItem v-if="modalMode === 'create'" label="类型">
        <NRadioGroup v-model:value="form.type">
          <NRadioButton v-for="opt in typeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadioButton>
        </NRadioGroup>
      </NFormItem>

      <NFormItem label="名称">
        <NInput v-model:value="form.title" placeholder="便于管理端识别（可选）" />
      </NFormItem>

      <div class="form-row">
        <NFormItem v-if="modalMode === 'create'" label="短码">
          <NInput v-model:value="form.code" placeholder="留空自动生成" />
        </NFormItem>
        <NFormItem v-else label="短码">
          <NInput :value="form.code" disabled />
        </NFormItem>

        <NFormItem v-if="modalMode === 'create'" label="入口域名" path="entryDomainId">
          <NSelect v-model:value="form.entryDomainId" :options="entryDomainOptions" placeholder="选择入口域名" :loading="domains.loading" />
        </NFormItem>
        <template v-else>
          <NFormItem label="入口域名">
            <NInput :value="editingEntryDomainHost" disabled />
          </NFormItem>
        </template>
      </div>
      <NFormItem v-if="modalMode === 'edit'" label="状态">
        <NSelect v-model:value="form.status" :options="statusOptions" />
      </NFormItem>

      <NAlert v-if="modalMode === 'create'" type="info" style="margin-bottom:16px">入口域名须指向公开服务。localhost 仅供本机测试；自动测试的 .test 域名没有公共 DNS，不能直接用于对外分享。</NAlert>
      <NFormItem v-if="form.type !== 'liveqr'" label="目标 URL" path="targetUrl">
        <NInput v-model:value="form.targetUrl" placeholder="https://example.com/path" />
      </NFormItem>
      <template v-else-if="modalMode === 'create'">
        <NFormItem label="活码落地页（必选）">
          <NSelect v-model:value="landingPageId" :options="landingPages.filter(p => p.Template === 'liveqr').map(p => ({ label: p.Title, value: p.ID }))" placeholder="先在落地页管理中创建活码页面" />
        </NFormItem>
        <div class="liveqr-section">
          <div class="section-title">
            <div><strong>分发策略</strong><p class="muted">决定多个二维码目标的展示方式</p></div>
            <NSelect v-model:value="form.strategyMode" :options="strategyOptions" style="width: 160px" />
          </div>
          <NAlert type="info" :show-icon="true">
            {{ form.strategyMode === 'round_robin' ? '按添加顺序使用目标，达到扫码上限后切换下一项；上限留空时会持续使用该目标。' : '权重模式按比例随机分发，权重越高，被选中的概率越大。' }}
          </NAlert>
        </div>

        <NFormItem label="二维码目标" path="targets">
          <div class="target-list">
            <div v-for="(target, index) in form.targets" :key="index" class="target-row">
              <div class="target-index">{{ index + 1 }}</div>
              <NInput v-model:value="target.label" placeholder="名称（可选）" />
              <div><NInput v-model:value="target.url" placeholder="https://example.com/qr.png 或从素材库选择" /><MaterialPicker relative @select="target.url=$event" /></div>
              <NInputNumber
                v-if="form.strategyMode === 'weighted'"
                v-model:value="target.weight"
                :min="1"
                :precision="0"
                placeholder="权重"
              />
              <NInputNumber v-model:value="target.scanLimit" :min="1" :precision="0" clearable placeholder="扫码上限" />
              <NButton text type="error" :disabled="form.targets.length === 1" aria-label="删除目标" @click="removeTarget(index)">
                <template #icon><Trash2 :size="16" /></template>
              </NButton>
            </div>
            <NButton dashed block @click="addTarget">
              <template #icon><Plus :size="16" /></template>
              添加二维码目标
            </NButton>
          </div>
        </NFormItem>
      </template>

      <NFormItem label="访问限制">
        <NSelect v-model:value="form.accessRule" :options="accessRuleOptions" style="width: 100%" />
      </NFormItem>

      <NFormItem label="过期时间（可选）">
        <NDatePicker v-model:value="form.expireAt" type="datetime" clearable style="width: 100%" />
      </NFormItem>

      <NAlert v-if="modalError" type="error" :show-icon="true" style="margin-bottom: var(--space-3)">{{ modalError }}</NAlert>
    </NForm>

    <template #footer>
      <div style="display: flex; justify-content: flex-end; gap: var(--space-2)">
        <NButton @click="showModal = false">取消</NButton>
        <NButton type="primary" :loading="saving" @click="submit">
          {{ modalMode === 'create' ? '创建' : '保存' }}
        </NButton>
      </div>
    </template>
  </NModal>

  <!-- P0 方案 A：创建成功分享面板 -->
  <ShareSuccessModal
    v-model:show="showShare"
    :link="shareLink"
    :url="shareLink ? shortUrlOf(shareLink) : ''"
    :target-url="shareLink?.TargetURL || ''"
    @recreate="showShare = false; openCreate()"
  />

  <!-- P0 方案 B：行内分享抽屉 -->
  <LinkShareDrawer
    v-model:show="showDrawer"
    :link="drawerLink"
    :url="drawerLink ? shortUrlOf(drawerLink) : ''"
    @edit="((l) => { showDrawer = false; openEdit(l); })($event)"
  />
  </div>
</template>

<style scoped>
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
  flex-wrap: wrap;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}

.liveqr-section {
  display: grid;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-subtle);
}

.section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.section-title p { margin-top: var(--space-1); }

.target-list {
  width: 100%;
  display: grid;
  gap: var(--space-3);
}

.target-row {
  display: grid;
  grid-template-columns: 28px minmax(110px, .55fr) minmax(220px, 1.4fr) minmax(100px, .45fr) 32px;
  align-items: center;
  gap: var(--space-2);
}

.target-row:has(.n-input-number:nth-of-type(2)) {
  grid-template-columns: 28px minmax(100px, .5fr) minmax(190px, 1.25fr) 90px 110px 32px;
}

.target-index {
  width: 26px;
  height: 26px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-primary);
  background: var(--color-primary-soft);
  font-size: var(--font-size-xs);
  font-weight: 700;
}

@media (max-width: 640px) {
  .form-row {
    grid-template-columns: 1fr;
  }
  .target-row,
  .target-row:has(.n-input-number:nth-of-type(2)) {
    grid-template-columns: 28px 1fr 32px;
  }
  .target-row > :not(.target-index):not(button) {
    grid-column: 2;
  }
  .target-row > button {
    grid-column: 3;
    grid-row: 1;
  }
}
</style>
