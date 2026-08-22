<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import { Copy, ExternalLink, Pencil, Plus, RefreshCw, Search, Trash2 } from '@lucide/vue';
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
  NModal,
  NPopconfirm,
  NRadioButton,
  NRadioGroup,
  NSelect,
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
  updateLink,
  type LinkItem,
} from '../api';
import { useDomainStore } from '../stores/domain';
import { useAuthStore } from '../stores/auth';

const route = useRoute();
const message = useMessage();
const domains = useDomainStore();
const auth = useAuthStore();

const items = ref<LinkItem[]>([]);
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

const form = reactive({
  type: 'short' as 'short' | 'channel' | 'liveqr',
  title: '',
  code: '',
  entryDomainId: null as number | null,
  targetUrl: '',
  targets: '',
  expireAt: null as number | null,
  status: 'active' as 'active' | 'disabled',
});

const statusOptions = [
  { label: '正常', value: 'active' },
  { label: '已停用', value: 'disabled' },
];

const typeOptions = [
  { label: '短链接', value: 'short' },
  { label: '渠道链接', value: 'channel' },
  { label: '活码', value: 'liveqr' },
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
      validator: (_rule, value: string) => {
        if (form.type !== 'liveqr') return true;
        const lines = value.split('\n').map((v) => v.trim()).filter(Boolean);
        if (!lines.length) return new Error('请至少填写一个轮询目标');
        for (const line of lines) {
          try {
            new URL(line);
          } catch {
            return new Error(`存在非法 URL：${line}`);
          }
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
  domains.items.forEach((d) => map.set(d.ID, `${d.Scheme}://${d.Host}`));
  return map;
});

function shortUrlOf(row: LinkItem): string {
  // 后端 LinkItem 当前未返回 EntryDomainID，退化为使用第一个 entry 域。
  const firstEntry = domains.entryDomains[0];
  const base = firstEntry ? `${firstEntry.Scheme}://${firstEntry.Host}` : shortUrlBase.value.get(0) || '';
  return base ? `${base}/${row.Code}` : row.Code;
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
          { text: true, size: 'tiny', onClick: () => copyShortUrl(row) },
          { icon: () => h(NIcon, { size: 14 }, { default: () => h(Copy) }) },
        ),
      ]),
  },
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
    render: (row) =>
      h(NTag, { type: statusTypeMap[row.Status] || 'default', size: 'small', round: true }, () => statusLabelMap[row.Status] || row.Status),
  },
  {
    title: '操作',
    key: 'actions',
    width: 160,
    render: (row) =>
      h('div', { style: 'display:flex;gap:4px' }, [
        h(
          NButton,
          { text: true, size: 'small', tag: 'a', href: shortUrlOf(row), target: '_blank' },
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
  await Promise.all([refresh(), domains.refresh()]);
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
  modalMode.value = 'create';
  editingId.value = null;
  Object.assign(form, {
    type: 'short',
    title: '',
    code: '',
    entryDomainId: domains.entryDomains[0]?.ID ?? null,
    targetUrl: '',
    targets: '',
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
    targets: '',
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
      await createLink({
        type: form.type,
        code: form.code || undefined,
        title: form.title || undefined,
        entry_domain_id: form.entryDomainId!,
        target_url: form.type === 'liveqr' ? '' : form.targetUrl,
        expire_at: form.expireAt ? new Date(form.expireAt).toISOString() : undefined,
        strategy:
          form.type === 'liveqr'
            ? {
                mode: 'round_robin',
                targets: form.targets
                  .split('\n')
                  .map((v) => v.trim())
                  .filter(Boolean)
                  .map((target_url) => ({ target_url, weight: 1 })),
              }
            : undefined,
      });
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
  try {
    await navigator.clipboard.writeText(shortUrlOf(row));
    message.success(`已复制 ${shortUrlOf(row)}`);
  } catch {
    message.error('复制失败，请手动复制');
  }
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : '操作失败';
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

  <NModal v-model:show="showModal" preset="card" :title="modalMode === 'create' ? '创建链接' : '编辑链接'" style="width: 620px" :mask-closable="false">
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
        <NFormItem v-else label="状态">
          <NSelect v-model:value="form.status" :options="statusOptions" />
        </NFormItem>
      </div>

      <NFormItem v-if="form.type !== 'liveqr'" label="目标 URL" path="targetUrl">
        <NInput v-model:value="form.targetUrl" placeholder="https://example.com/path" />
      </NFormItem>
      <NFormItem v-else label="轮询目标" path="targets">
        <NInput v-model:value="form.targets" type="textarea" :rows="4" placeholder="每行一个完整 URL" />
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

@media (max-width: 640px) {
  .form-row {
    grid-template-columns: 1fr;
  }
}
</style>
