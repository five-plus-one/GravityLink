<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue';
import { Key, Plus, RefreshCw, ShieldCheck, Trash2 } from '@lucide/vue';
import {
  NAlert, NButton, NCard, NDataTable, NDatePicker, NEmpty, NForm, NFormItem, NIcon, NInput,
  NInputNumber, NModal, NPopconfirm, NSelect, NSwitch, NTag, useMessage,
  type DataTableColumns, type FormInst, type FormRules,
} from 'naive-ui';
import { request } from '../api';
import { useAuthStore } from '../stores/auth';

interface APIKey {
  ID: number; Name: string; Status: string;
  sign_enabled: boolean; Quota: number | null; Used: number;
  expire_at: string | null; ip_whitelist: string | null;
  last_used_at: string | null; created_at: string;
}

const message = useMessage();
const auth = useAuthStore();
const keys = ref<APIKey[]>([]);
const loading = ref(false);
const showModal = ref(false);
const saving = ref(false);
const modalError = ref('');
const formRef = ref<FormInst | null>(null);
// 创建结果（token/hmac 只显示一次）
const createdResult = ref<{ token: string; hmac_secret?: string } | null>(null);

const form = reactive({
  name: '',
  signEnabled: false,
  quota: null as number | null,
  expireAt: null as number | null,
  ipWhitelist: '',
});

const canWrite = computed(() => auth.isSuperAdmin || auth.user?.role === 'admin');

const rules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
};

const columns: DataTableColumns<APIKey> = [
  { title: '名称', key: 'Name' },
  {
    title: 'Token（已掩码）',
    key: 'token_display',
    render: (row) => h('code', { style: 'font-size:12px;color:#6b7f88' }, `gl_live_${'*'.repeat(20)}`),
  },
  {
    title: '签名',
    key: 'sign_enabled',
    width: 80,
    render: (row) => row.sign_enabled
      ? h(NTag, { type: 'warning', size: 'small' }, () => 'HMAC')
      : h(NTag, { size: 'small' }, () => 'Token'),
  },
  {
    title: '配额',
    key: 'Quota',
    render: (row) => {
      if (!row.Quota) return h('span', { class: 'muted' }, '不限');
      const pct = Math.min(100, Math.round((row.Used / row.Quota) * 100));
      return h('div', { style: 'display:flex;align-items:center;gap:8px' }, [
        h('span', {}, `${row.Used} / ${row.Quota}`),
        h('div', { style: `width:70px;height:6px;background:#e8edf0;border-radius:99px;overflow:hidden` }, [
          h('div', { style: `height:100%;width:${pct}%;background:${pct >= 90 ? '#b45309' : '#0f766e'};border-radius:99px;display:block` }),
        ]),
      ]);
    },
  },
  {
    title: 'IP 白名单',
    key: 'ip_whitelist',
    render: (row) => row.ip_whitelist ? h('code', { style: 'font-size:12px' }, row.ip_whitelist) : h('span', { class: 'muted' }, '不限'),
  },
  {
    title: '有效期',
    key: 'expire_at',
    render: (row) => row.expire_at ? new Date(row.expire_at).toLocaleDateString() : '永久',
  },
  {
    title: '状态',
    key: 'Status',
    width: 80,
    render: (row) => h(NSwitch, {
      size: 'small',
      value: row.Status === 'active',
      onUpdateValue: async (v: boolean) => {
        try {
          await request(`/api/admin/api-keys/${row.ID}`, { method: 'PUT', body: JSON.stringify({ status: v ? 'active' : 'disabled' }) });
          row.Status = v ? 'active' : 'disabled';
          message.success(`已${v ? '启用' : '停用'}`);
        } catch (e) { message.error(String(e)); }
      },
    }),
  },
  {
    title: '',
    key: 'actions',
    width: 70,
    render: (row) =>
      h(NPopconfirm, { onPositiveClick: () => remove(row) }, {
        trigger: () => h(NButton, { text: true, type: 'error', size: 'small' }, { icon: () => h(NIcon, { size: 14 }, { default: () => h(Trash2) }), default: () => '删除' }),
        default: () => `确认删除「${row.Name}」？`,
      }),
  },
];

onMounted(refresh);

async function refresh() {
  loading.value = true;
  try {
    keys.value = (await request<{ items: APIKey[] }>('/api/admin/api-keys')).items;
  } catch (e) { message.error(String(e)); }
  finally { loading.value = false; }
}

function openCreate() {
  Object.assign(form, { name: '', signEnabled: false, quota: null, expireAt: null, ipWhitelist: '' });
  modalError.value = '';
  createdResult.value = null;
  showModal.value = true;
}

async function submit() {
  modalError.value = '';
  try { await formRef.value?.validate(); } catch { return; }
  saving.value = true;
  try {
    const result = await request<{ id: number; name: string; token: string; hmac_secret?: string; sign_enabled: boolean }>('/api/admin/api-keys', {
      method: 'POST',
      body: JSON.stringify({
        name: form.name,
        sign_enabled: form.signEnabled,
        quota: form.quota,
        expire_at: form.expireAt ? new Date(form.expireAt).toISOString() : undefined,
        ip_whitelist: form.ipWhitelist || undefined,
      }),
    });
    createdResult.value = { token: result.token, hmac_secret: result.hmac_secret };
    message.success('API Key 创建成功');
    await refresh();
  } catch (e) { modalError.value = String(e); }
  finally { saving.value = false; }
}

async function remove(row: APIKey) {
  try {
    await request(`/api/admin/api-keys/${row.ID}`, { method: 'DELETE' });
    message.success(`已删除「${row.Name}」`);
    await refresh();
  } catch (e) { message.error(String(e)); }
}

function maskToken(t: string) { return t.length > 12 ? t.slice(0, 10) + '...' + t.slice(-4) : t; }
</script>

<template>
  <div class="page-view">
    <NCard>
    <template #header>
      <div class="card-head">
        <div><strong>开放 API</strong><p class="muted">管理 API Key，通过 Bearer Token 或 HMAC 签名创建短链接</p></div>
        <div class="card-actions">
          <NButton :loading="loading" @click="refresh"><template #icon><RefreshCw :size="16" /></template></NButton>
          <NButton v-if="canWrite" type="primary" @click="openCreate"><template #icon><Plus :size="16" /></template>创建 API Key</NButton>
        </div>
      </div>
    </template>
    <NDataTable :columns="columns" :data="keys" :loading="loading" :pagination="{ pageSize: 20 }" :scroll-x="900" :bordered="false" size="small">
      <template #empty>
        <NEmpty description="尚未创建 API Key"><template v-if="canWrite" #extra><NButton type="primary" size="small" @click="openCreate">创建第一个 Key</NButton></template></NEmpty>
      </template>
    </NDataTable>
    </NCard>

    <NModal v-model:show="showModal" preset="card" title="创建 API Key" style="width:min(560px,94vw)" :mask-closable="false">
      <template v-if="createdResult">
        <div class="created-box">
          <div class="created-icon"><ShieldCheck :size="36" /></div>
          <p style="font-size:14px;margin-bottom:12px"><b>API Key 创建成功</b>，以下凭据仅显示一次，请立即保存。</p>
          <div class="copy-row"><span class="row-label">Token</span><code class="row-val">{{ createdResult.token }}</code></div>
          <div v-if="createdResult.hmac_secret" class="copy-row"><span class="row-label">Secret</span><code class="row-val">{{ createdResult.hmac_secret }}</code></div>
        </div>
      </template>
      <template v-else>
        <NForm ref="formRef" :model="form" :rules="rules" label-placement="top">
          <NFormItem label="Key 名称" path="name"><NInput v-model:value="form.name" placeholder="如：订单系统 / CRM 同步" /></NFormItem>
          <NFormItem label="HMAC 签名"><div style="display:flex;align-items:center;gap:12px"><NSwitch v-model:value="form.signEnabled" /><span class="muted">开启后每次请求需带 X-Timestamp + X-Signature 头（HMAC-SHA256），防重放</span></div></NFormItem>
          <div class="form-row">
            <NFormItem label="请求配额（留空不限）"><NInputNumber v-model:value="form.quota" :min="1" clearable style="width:100%" /></NFormItem>
            <NFormItem label="有效期（留空永久）"><NDatePicker v-model:value="form.expireAt" type="datetime" clearable style="width:100%" /></NFormItem>
          </div>
          <NFormItem label="IP 白名单（逗号分隔多 IP 或 CIDR，留空不限）"><NInput v-model:value="form.ipWhitelist" placeholder="如：1.2.3.4, 10.0.0.0/24" /></NFormItem>
          <NAlert v-if="modalError" type="error">{{ modalError }}</NAlert>
        </NForm>
      </template>
      <template #footer>
        <div v-if="createdResult" style="display:flex;justify-content:flex-end">
          <NButton type="primary" @click="showModal = false">我已保存，关闭</NButton>
        </div>
        <div v-else style="display:flex;gap:8px;justify-content:flex-end">
          <NButton @click="showModal=false">取消</NButton>
          <NButton type="primary" :loading="saving" @click="submit">创建</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.page-view{display:grid;gap:16px}
.card-head{display:flex;justify-content:space-between;align-items:flex-start;gap:16px;width:100%;flex-wrap:wrap}
.card-head strong{font-size:18px;display:block}.card-head p{margin-top:4px;font-size:13px;color:#6b7f88}
.card-actions{display:flex;gap:8px;align-items:center}
.form-row{display:grid;grid-template-columns:1fr 1fr;gap:12px}
.created-box{display:grid;gap:12px;justify-items:center;padding:16px}
.created-icon{color:#15803d;background:#dcfce7;width:60px;height:60px;border-radius:50%;display:grid;place-items:center}
.copy-row{width:100%;display:flex;gap:8px;align-items:center;background:#f4f7f8;border:1px solid #e3eaed;border-radius:8px;padding:10px 12px}
.row-label{flex-shrink:0;font-size:12px;color:#6b7f88;background:#fff;border:1px solid #e3eaed;border-radius:4px;padding:2px 8px}
.row-val{flex:1;font-family:Consolas,Menlo,monospace;font-size:13px;word-break:break-all}
@media (max-width: 640px) {
  .form-row{grid-template-columns:1fr}
}
</style>
