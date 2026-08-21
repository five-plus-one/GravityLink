<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue';
import { Plus, RefreshCw, Trash2 } from '@lucide/vue';
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
import { createDomain, deleteDomain, listDomains, type DomainItem } from '../api';
import { useAuthStore } from '../stores/auth';
import { useDomainStore } from '../stores/domain';

const message = useMessage();
const auth = useAuthStore();
const domainStore = useDomainStore();

const items = ref<DomainItem[]>([]);
const loading = ref(false);
const showModal = ref(false);
const saving = ref(false);
const modalError = ref('');
const formRef = ref<FormInst | null>(null);

const form = reactive({
  host: '',
  type: 'entry' as 'entry' | 'transit' | 'landing',
  scheme: 'https' as 'http' | 'https',
  remark: '',
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

const typeLabelMap: Record<string, string> = { entry: '入口', transit: '中转', landing: '落地页' };
const typeTagMap: Record<string, 'info' | 'warning' | 'success'> = { entry: 'success', transit: 'warning', landing: 'info' };
const statusLabelMap: Record<string, string> = { active: '正常', disabled: '已停用' };
const statusTypeMap: Record<string, 'success' | 'default'> = { active: 'success', disabled: 'default' };

const rules: FormRules = {
  host: [
    {
      required: true,
      validator: (_rule, value: string) => {
        if (!value) return new Error('请输入域名');
        if (!/^([a-z0-9-]+\.)+[a-z]{2,}(:\d+)?$/i.test(value.trim())) return new Error('域名格式不正确，例如 go.example.com');
        return true;
      },
      trigger: ['blur', 'input'],
    },
  ],
};

const columns: DataTableColumns<DomainItem> = [
  { title: 'Host', key: 'Host', render: (row) => h('code', {}, `${row.Scheme}://${row.Host}`) },
  {
    title: '用途',
    key: 'Type',
    width: 110,
    render: (row) => h(NTag, { type: typeTagMap[row.Type] || 'default', size: 'small' }, () => typeLabelMap[row.Type] || row.Type),
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
    width: 90,
    render: (row) =>
      canWrite.value
        ? h(
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
          )
        : h('span', { class: 'muted' }, '—'),
  },
];

onMounted(refresh);

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
  Object.assign(form, { host: '', type: 'entry', scheme: 'https', remark: '' });
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
    await createDomain({ ...form, host: form.host.trim(), remark: form.remark || undefined });
    message.success(`域名 ${form.host} 已添加`);
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
  <NCard>
    <template #header>
      <div class="card-head">
        <div>
          <strong>域名</strong>
          <p class="muted">管理入口、中转和落地页域名</p>
        </div>
        <div class="card-actions">
          <NButton :loading="loading" @click="refresh">
            <template #icon><RefreshCw :size="16" /></template>
          </NButton>
          <NButton v-if="canWrite" type="primary" @click="openCreate">
            <template #icon><Plus :size="16" /></template>
            添加域名
          </NButton>
        </div>
      </div>
    </template>

    <NDataTable :columns="columns" :data="items" :loading="loading" :pagination="{ pageSize: 20 }" :bordered="false" size="small">
      <template #empty>
        <NEmpty description="尚未配置域名">
          <template v-if="canWrite" #extra>
            <NButton type="primary" size="small" @click="openCreate">添加第一个域名</NButton>
          </template>
        </NEmpty>
      </template>
    </NDataTable>
  </NCard>

  <NModal v-model:show="showModal" preset="card" title="添加域名" style="width: 520px" :mask-closable="false">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="top">
      <NFormItem label="Host" path="host">
        <NInput v-model:value="form.host" placeholder="go.example.com" />
      </NFormItem>
      <div class="form-row">
        <NFormItem label="用途">
          <NSelect v-model:value="form.type" :options="typeOptions" />
        </NFormItem>
        <NFormItem label="协议">
          <NSelect v-model:value="form.scheme" :options="schemeOptions" />
        </NFormItem>
      </div>
      <NFormItem label="备注">
        <NInput v-model:value="form.remark" placeholder="（可选）" />
      </NFormItem>
      <NAlert v-if="modalError" type="error" :show-icon="true" style="margin-bottom: var(--space-3)">{{ modalError }}</NAlert>
      <NAlert type="info" :show-icon="true">DNS 解析与证书需要在反向代理层完成。</NAlert>
    </NForm>

    <template #footer>
      <div style="display: flex; justify-content: flex-end; gap: var(--space-2)">
        <NButton @click="showModal = false">取消</NButton>
        <NButton type="primary" :loading="saving" @click="submit">添加</NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
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
</style>
