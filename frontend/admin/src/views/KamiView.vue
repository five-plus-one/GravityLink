<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue';
import { Key, Plus, RefreshCw, Trash2 } from '@lucide/vue';
import {
  NAlert, NButton, NCard, NDataTable, NEmpty, NForm, NFormItem, NGrid, NGridItem,
  NInput, NInputNumber, NModal, NPopconfirm, NSelect, NSwitch, NTabPane, NTabs, NTag,
  useMessage, type DataTableColumns, type FormInst, type FormRules,
} from 'naive-ui';
import { request } from '../api';
import { useAuthStore } from '../stores/auth';

interface KamiProject { id: number; title: string; type: string; status: string; remaining: number; issued: number }
interface KamiItem { ID: number; Content: string; Note: string | null; ExpiresText: string | null; Status: string; IssuedAt: string | null; IssuedIP: string | null; CreatedAt: string }
interface KamiIssuance { ID: number; ProjectID: number; KamiItemID: number; VisitorIP: string | null; VisitorUA: string | null; VisitorDevice: string | null; CreatedAt: string }

const message = useMessage();
const auth = useAuthStore();
const canWrite = computed(() => auth.isSuperAdmin || auth.user?.role === 'admin');
const projects = ref<KamiProject[]>([]);
const loading = ref(false);

// 项目创建表单
const showCreate = ref(false);
const saving = ref(false);
const modalError = ref('');
const createFormRef = ref<FormInst | null>(null);
const createForm = reactive({
  title: '', type: '卡密', password: '', repeatPolicy: 'never' as 'never' | 'allow', repeatInterval: 60,
});

function resetCreateForm() {
  Object.assign(createForm, { title: '', type: '卡密', password: '', repeatPolicy: 'never', repeatInterval: 60 });
}

// 项目详情
const detailProject = ref<KamiProject | null>(null);
const items = ref<KamiItem[]>([]);
const itemsTotal = ref(0);
const itemsLoading = ref(false);
const itemPage = ref(0);

// 提取记录
const issuances = ref<KamiIssuance[]>([]);
const issuTotal = ref(0);
const issLoading = ref(false);

// 导入
const showImport = ref(false);
const importText = ref('');
const importNote = ref('');
const importBusy = ref(false);

const projectItemCols: DataTableColumns<KamiItem> = [
  { title: '卡密内容', key: 'Content', render: (r) => h('code', { style: 'font-size:12px' }, r.Content) },
  { title: '备注', key: 'Note', render: (r) => r.Note || '—' },
  { title: '有效期', key: 'ExpiresText', render: (r) => r.ExpiresText || '—' },
  { title: '状态', key: 'Status', width: 80, render: (r) => h(NTag, { type: r.Status === 'issued' ? 'success' : 'default', size: 'small', round: true }, () => r.Status === 'issued' ? '已发' : '待发') },
  { title: '发出时间', key: 'IssuedAt', render: (r) => r.IssuedAt ? new Date(r.IssuedAt).toLocaleString('zh-CN', { hour12: false }) : '—' },
  { title: '发出IP', key: 'IssuedIP', render: (r) => r.IssuedIP || '—' },
  { title: '', key: 'act', width: 60, render: (r) => h(NPopconfirm, { onPositiveClick: () => deleteItem(r.ID) }, { trigger: () => h(NButton, { text: true, type: 'error', size: 'small' }, () => '删除'), default: () => '确认删除此卡密？' }) },
];

const issuanceCols: DataTableColumns<KamiIssuance> = [
  { title: 'IP', key: 'VisitorIP', render: (r) => r.VisitorIP || '—' },
  { title: '设备', key: 'VisitorDevice', render: (r) => r.VisitorDevice || '—' },
  { title: 'UA', key: 'VisitorUA', ellipsis: { tooltip: true }, render: (r) => r.VisitorUA || '—' },
  { title: '提取时间', key: 'CreatedAt', render: (r) => new Date(r.CreatedAt).toLocaleString('zh-CN', { hour12: false }) },
];

onMounted(loadProjects);

async function loadProjects() {
  loading.value = true;
  try { projects.value = (await request<{ items: KamiProject[] }>('/api/admin/kami/projects')).items; }
  catch (e) { message.error(String(e)); }
  finally { loading.value = false; }
}

async function openDetail(p: KamiProject) { detailProject.value = p; await loadItems(); }
async function loadItems() {
  if (!detailProject.value) return;
  itemsLoading.value = true;
  try { const r = await request<{ items: KamiItem[]; total: number }>(`/api/admin/kami/projects/${detailProject.value!.id}/items?limit=50&offset=${itemPage.value * 50}`); items.value = r.items; itemsTotal.value = r.total; } catch (e) { message.error(String(e)); } finally { itemsLoading.value = false; }
}
async function loadIssuances() {
  if (!detailProject.value) return;
  issLoading.value = true;
  try { const r = await request<{ items: KamiIssuance[]; total: number }>(`/api/admin/kami/projects/${detailProject.value!.id}/issuances?limit=50`); issuances.value = r.items; issuTotal.value = r.total; } catch (e) { message.error(String(e)); } finally { issLoading.value = false; }
}

function openImport() { importText.value = ''; importNote.value = ''; showImport.value = true; }
async function doImport() {
  importBusy.value = true;
  try {
    const lines = importText.value.split('\n').filter(Boolean);
    const items = lines.map(l => ({ content: l.trim(), note: importNote.value || undefined }));
    const r = await request<{ imported: number; dup: number; failed: number }>(`/api/admin/kami/projects/${detailProject.value!.id}/items/import`, { method: 'POST', body: JSON.stringify({ items }) });
    message.success(`已导入 ${r.imported} 条${r.dup ? `，${r.dup} 条重复跳过` : ''}`);
    showImport.value = false;
    await loadItems();
  } catch (e) { message.error(String(e)); } finally { importBusy.value = false; }
}

async function deleteItem(id: number) {
  try { await request(`/api/admin/kami/items/${id}`, { method: 'DELETE' }); await loadItems(); } catch (e) { message.error(String(e)); }
}

async function createProject() {
  modalError.value = '';
  try { await createFormRef.value?.validate(); } catch { return; }
  saving.value = true;
  try {
    await request('/api/admin/kami/projects', {
      method: 'POST',
      body: JSON.stringify({
        title: createForm.title, type: createForm.type,
        password: createForm.password || undefined,
        repeat_policy: createForm.repeatPolicy,
        repeat_interval_sec: createForm.repeatPolicy === 'allow' ? createForm.repeatInterval : 0,
      }),
    });
    message.success('项目已创建');
    showCreate.value = false;
    await loadProjects();
  } catch (e) { modalError.value = String(e); }
  finally { saving.value = false; }
}
</script>

<template>
  <div class="page-view">
    <!-- 项目列表 -->
    <NCard v-if="!detailProject">
    <template #header>
      <div class="card-head">
        <div><strong>卡密分发</strong><p class="muted">管理提取项目、导入卡密与查看提取记录</p></div>
        <div class="card-actions">
          <NButton :loading="loading" @click="loadProjects"><template #icon><RefreshCw :size="16" /></template></NButton>
          <NButton v-if="canWrite" type="primary" @click="resetCreateForm(); modalError=''; showCreate = true"><template #icon><Plus :size="16" /></template>创建项目</NButton>
        </div>
      </div>
    </template>
    <div class="project-grid">
      <NCard v-for="p in projects" :key="p.id" hoverable class="project-card" @click="openDetail(p)">
        <div class="pc-head">
          <strong>{{ p.title }}</strong>
          <NTag :type="p.status==='active'?'success':'default'" size="small" round>{{ p.status==='active'?'启用':'停用' }}</NTag>
        </div>
        <p class="muted" style="font-size:12px">{{ p.type }}</p>
        <div style="margin:12px 0 8px;display:flex;justify-content:space-between;font-size:12.5px">
          <span>剩余 <b>{{ p.remaining }}</b></span>
          <span>已发 <b>{{ p.issued }}</b></span>
        </div>
        <div class="bar-track"><div class="bar-fill" :style="{width: (p.issued+p.remaining? Math.round(p.issued/(p.issued+p.remaining)*100):0)+'%'}"></div></div>
      </NCard>
      <div v-if="canWrite" class="project-card add-card" @click="resetCreateForm(); showCreate = true">
        + 创建项目
      </div>
    </div>
    <NEmpty v-if="!projects.length && !loading" description="尚无卡密项目" />
    </NCard>

    <!-- 项目详情 -->
    <NCard v-else>
      <template #header>
        <div class="card-head">
          <div style="display:flex;gap:12px;align-items:center">
            <NButton text @click="detailProject = null">← 返回</NButton>
            <strong>{{ detailProject.title }}</strong>
          </div>
          <div class="card-actions">
            <NButton size="small" ghost @click="openImport">导入卡密</NButton>
            <NButton size="small" @click="loadItems"><template #icon><RefreshCw :size="14" /></template></NButton>
          </div>
        </div>
      </template>
      <NTabs type="line">
        <NTabPane name="items" tab="卡密列表">
          <NDataTable :columns="projectItemCols" :data="items" :loading="itemsLoading" :pagination="itemsTotal>50?{pageSize:50,onChange:(p)=>{itemPage=p-1;loadItems()}}:false" :scroll-x="800" :bordered="false" size="small" />
          <p v-if="itemsTotal" style="font-size:12px;color:#6b7f88;margin-top:8px">共 {{ itemsTotal }} 条</p>
        </NTabPane>
        <NTabPane name="issuances" tab="提取记录">
          <NButton size="small" @click="loadIssuances" style="margin-bottom:12px">刷新记录</NButton>
          <NDataTable :columns="issuanceCols" :data="issuances" :loading="issLoading" :scroll-x="600" :bordered="false" size="small" />
          <p v-if="issuTotal" style="font-size:12px;color:#6b7f88;margin-top:8px">共 {{ issuTotal }} 条</p>
        </NTabPane>
      </NTabs>
    </NCard>

    <!-- 创建项目弹窗 -->
    <NModal v-model:show="showCreate" preset="card" title="创建卡密项目" style="width:min(480px,94vw)" :mask-closable="false">
      <NForm ref="createFormRef" :model="createForm" label-placement="top">
        <NFormItem label="项目标题" :rule="{required:true,trigger:'blur'}"><NInput v-model:value="createForm.title" placeholder="如：2026 会员兑换码" /></NFormItem>
        <NFormItem label="类型"><NSelect v-model:value="createForm.type" :options="['卡密','激活码','兑换码','序列号','链接','验证码'].map(v=>({label:v,value:v}))" /></NFormItem>
        <NFormItem label="提取口令（留空表示无需口令）"><NInput v-model:value="createForm.password" placeholder="可选，访客需输入此口令才能提取" /></NFormItem>
        <NFormItem label="允许重复提取"><div style="display:flex;gap:12px;align-items:center"><NSwitch v-model:value="createForm.repeatPolicy" active-value="allow" inactive-value="never" /><span v-if="createForm.repeatPolicy==='allow'" class="muted">间隔 <NInputNumber v-model:value="createForm.repeatInterval" :min="1" size="small" style="width:80px;display:inline-block" /> 秒</span></div></NFormItem>
        <NAlert v-if="modalError" type="error">{{ modalError }}</NAlert>
      </NForm>
      <template #footer><div style="display:flex;gap:8px;justify-content:flex-end"><NButton @click="showCreate=false">取消</NButton><NButton type="primary" :loading="saving" @click="createProject">创建</NButton></div></template>
    </NModal>

    <!-- 导入弹窗 -->
    <NModal v-model:show="showImport" preset="card" title="批量导入卡密" style="width:min(560px,94vw)" :mask-closable="false">
      <NAlert type="info" style="margin-bottom:12px">每行一条卡密内容，最多 1000 条，自动去重。</NAlert>
      <NFormItem label="备注（可选，应用到全部）"><NInput v-model:value="importNote" placeholder="如：7天卡" /></NFormItem>
      <NInput v-model:value="importText" type="textarea" :rows="8" placeholder="VIP-001&#10;VIP-002&#10;VIP-003" />
      <p style="font-size:12px;color:#6b7f88;margin:8px 0">共 {{ importText.split('\n').filter(Boolean).length }} 条</p>
      <template #footer><NButton type="primary" :loading="importBusy" :disabled="!importText.trim()" @click="doImport">确认导入</NButton><NButton @click="showImport=false">取消</NButton></template>
    </NModal>
  </div>
</template>

<style scoped>
.page-view{display:grid;gap:16px}
.card-head{display:flex;justify-content:space-between;align-items:flex-start;gap:16px;width:100%;flex-wrap:wrap}
.card-head strong{font-size:18px;display:block}.card-head p{margin-top:4px;font-size:13px;color:#6b7f88}
.card-actions{display:flex;gap:8px;align-items:center}
.project-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:16px;margin-bottom:16px}
.project-card{cursor:pointer}.project-card:hover{transform:translateY(-2px);box-shadow:0 4px 16px rgba(16,42,51,.1)}
.pc-head{display:flex;justify-content:space-between;align-items:center;margin-bottom:4px}
.bar-track{height:6px;border-radius:99px;background:#e8edf0;overflow:hidden}.bar-fill{height:100%;border-radius:99px;background:#0f766e;display:block}
.add-card{border:1.5px dashed #c2cdd2;background:#f8fafb;display:flex;align-items:center;justify-content:center;min-height:140px;font-size:14px;color:#6b7f88;cursor:pointer}
</style>
