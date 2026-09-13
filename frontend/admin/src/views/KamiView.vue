<script setup lang="ts">
import ViewportTable from '../components/ViewportTable.vue';
import { computed, h, onMounted, reactive, ref } from 'vue';
import { Key, Plus, RefreshCw, Trash2 } from '@lucide/vue';
import {
  NAlert, NButton, NCard, NDataTable, NEmpty, NForm, NFormItem, NGrid, NGridItem,
  NInput, NInputNumber, NModal, NPopconfirm, NSelect, NSwitch, NTabPane, NTabs, NTag,
  useMessage, type DataTableColumns, type FormInst, type FormRules,
} from 'naive-ui';
import { request, listLinks, listLandingPages, createLandingPage, createLink, type LinkItem } from '../api';
import { useDomainStore } from '../stores/domain';
import EntryQRCode from '../components/EntryQRCode.vue';
import { useAuthStore } from '../stores/auth';

interface KamiProject { password_configured:boolean; repeat_policy:string; repeat_interval_sec:number; id: number; title: string; type: string; status: string; remaining: number; issued: number }
interface KamiItem { ID: number; Content: string; Note: string | null; ExpiresText: string | null; Status: string; IssuedAt: string | null; IssuedIP: string | null; CreatedAt: string }
interface KamiIssuance { ID: number; ProjectID: number; KamiItemID: number; VisitorIP: string | null; VisitorUA: string | null; VisitorDevice: string | null; CreatedAt: string }

const message = useMessage();
const auth = useAuthStore();
const domains = useDomainStore();
const publishBusy = ref(false);
const entryId = ref<number|null>(null);
const landingId = ref<number|null>(null);
const published = ref<LinkItem[]>([]);
const pendingPage = ref<number|null>(null);
const issuancePage = ref(1);
async function loadPublished() {
  const [pages, links] = await Promise.all([listLandingPages(), listLinks()]);
  const ids = pages.items.filter(p => p.Template === 'kami' && Number(p.Content?.project_id) === detailProject.value?.id).map(p => p.ID);
  published.value = links.items.filter(l => ids.includes((l as LinkItem & {LandingPageID:number}).LandingPageID));
}
async function publish() {
  if (!detailProject.value || !entryId.value || !landingId.value) return;
  publishBusy.value = true;
  try {
    if (!pendingPage.value) pendingPage.value = (await createLandingPage({template:'kami',title:detailProject.value.title,domain_id:landingId.value,content:{project_id:detailProject.value.id,button_text:'立即领取',announcement:'点击下方按钮领取',theme_color:'#2563eb'}})).ID;
    await createLink({type:'liveqr',title:detailProject.value.title,entry_domain_id:entryId.value,landing_domain_id:landingId.value,landing_page_id:pendingPage.value,target_url:''});
    pendingPage.value = null;
    await loadPublished(); message.success('领取链接已生成');
  } catch (e) { message.error(String(e)); } finally { publishBusy.value = false; }
}
async function copyURL(url:string) { try { await navigator.clipboard.writeText(url); message.success('领取地址已复制'); } catch { message.error('复制失败，请手动复制地址'); } }

const canWrite = computed(() => auth.isSuperAdmin || auth.user?.role === 'admin');
const projects = ref<KamiProject[]>([]);
const loading = ref(false);

const editProject=ref<KamiProject|null>(null), editBusy=ref(false),editError=ref('');
const projectSettings=reactive({title:'',passwordMode:'keep',password:'',repeatPolicy:'never',repeatInterval:60,status:'active'});
function openSettings(p:KamiProject){editProject.value=p;editError.value='';Object.assign(projectSettings,{title:p.title,passwordMode:'keep',password:'',repeatPolicy:p.repeat_policy,repeatInterval:p.repeat_interval_sec,status:p.status})}
async function saveSettings(){
 if(!editProject.value)return;
 if(!projectSettings.title.trim()){editError.value='请输入项目标题';return}
 if(projectSettings.passwordMode==='change'&&!projectSettings.password){editError.value='请输入新口令';return}
 editBusy.value=true;editError.value='';
 try{await request(`/api/admin/kami/projects/${editProject.value.id}`,{method:'PUT',body:JSON.stringify({title:projectSettings.title.trim(),password:projectSettings.passwordMode==='keep'?undefined:projectSettings.passwordMode==='clear'?'':projectSettings.password,repeat_policy:projectSettings.repeatPolicy,repeat_interval_sec:projectSettings.repeatInterval,status:projectSettings.status})});await refreshProjectStats();editProject.value=null;message.success('项目设置已保存')}
 catch(e){editError.value=e instanceof Error?e.message:'保存失败'}finally{editBusy.value=false}
}

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

onMounted(async () => { await loadProjects(); try { await domains.refresh(); entryId.value = domains.entryDomains.find(d => d.Status === 'active')?.ID ?? null; landingId.value = domains.landingDomains.find(d => d.Status === 'active')?.ID ?? null; } catch { message.error('加载域名失败'); } });

async function loadProjects() {
  loading.value = true;
  try { projects.value = (await request<{ items: KamiProject[] }>('/api/admin/kami/projects')).items; }
  catch (e) { message.error(String(e)); }
  finally { loading.value = false; }
}

async function openDetail(p: KamiProject) { detailProject.value = p; itemPage.value = 0; issuancePage.value = 1; pendingPage.value = null; published.value = []; await Promise.all([loadItems(), loadIssuances(), loadPublished().catch(() => message.error('加载领取链接失败'))]); }
async function refreshProjectStats() {
  await loadProjects();
  detailProject.value = projects.value.find(p => p.id === detailProject.value?.id) ?? detailProject.value;
}
async function loadItems() {
  if (!detailProject.value) return;
  itemsLoading.value = true;
  try { const r = await request<{ items: KamiItem[]; total: number }>(`/api/admin/kami/projects/${detailProject.value!.id}/items?limit=50&offset=${itemPage.value * 50}`); items.value = r.items ?? []; itemsTotal.value = r.total; await refreshProjectStats(); } catch (e) { message.error(String(e)); } finally { itemsLoading.value = false; }
}
async function loadIssuances() {
  if (!detailProject.value) return;
  issLoading.value = true;
  try { const r = await request<{ items: KamiIssuance[]; total: number }>(`/api/admin/kami/projects/${detailProject.value!.id}/issuances?limit=50&offset=${(issuancePage.value-1)*50}`); issuances.value = r.items ?? []; issuTotal.value = r.total; await refreshProjectStats(); } catch (e) { message.error(String(e)); } finally { issLoading.value = false; }
}

function openImport() { importText.value = ''; importNote.value = ''; showImport.value = true; }
async function doImport() {
  importBusy.value = true;
  try {
    const lines = importText.value.split('\n').map(l => l.trim()).filter(Boolean);
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
        <NButton v-if="canWrite" size="small" style="margin-bottom:10px" @click.stop="openSettings(p)">项目设置</NButton>
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
            <NButton v-if="canWrite" size="small" @click="openSettings(detailProject)">项目设置</NButton>
            <NButton size="small" ghost @click="openImport">导入卡密</NButton>
            <NButton size="small" @click="loadItems"><template #icon><RefreshCw :size="14" /></template></NButton>
          </div>
        </div>
      </template>
      <NCard title="领取链接" size="small" embedded style="margin-bottom:16px">
        <div v-for="link in published" :key="link.ID" class="publish-row">
          <a :href="link.PublicURL" target="_blank" rel="noopener noreferrer" class="claim-url">{{ link.PublicURL || link.Code }}</a>
          <NButton :disabled="!link.PublicURL" @click="copyURL(link.PublicURL!)">复制地址</NButton>
          <EntryQRCode :url="link.PublicURL || ''" :name="link.Title || link.Code" />
        </div>
        <p v-if="!published.length" class="muted">导入卡密后，选择域名生成领取链接，即可发给访客。</p>
        <div v-if="!published.length" class="publish-row">
          <NSelect v-model:value="entryId" :disabled="publishBusy" :options="domains.entryDomains.filter(d=>d.Status==='active').map(d=>({label:d.Host,value:d.ID}))" placeholder="选择入口域名" />
          <NSelect v-model:value="landingId" :disabled="publishBusy || !!pendingPage" :options="domains.landingDomains.filter(d=>d.Status==='active').map(d=>({label:d.Host,value:d.ID}))" placeholder="选择领取页域名" />
          <NButton type="primary" :loading="publishBusy" :disabled="!entryId || !landingId" @click="publish">生成领取链接</NButton>
        </div>
        <p class="muted">剩余 {{ detailProject.remaining }} 条 · 已发 {{ detailProject.issued }} 条</p>
      </NCard>
      <NTabs type="line" @update:value="v => { if(v==='issuances') loadIssuances() }">
        <NTabPane name="items" tab="卡密列表">
          <ViewportTable :max-height="440" :columns="projectItemCols" :data="items" :loading="itemsLoading" remote :pagination="{page:itemPage+1,pageSize:50,itemCount:itemsTotal,pageSlot:5,onUpdatePage:(p:number)=>{itemPage=p-1;loadItems()}}" :scroll-x="800" :bordered="false" size="small" />
          <p v-if="itemsTotal" style="font-size:12px;color:#6b7f88;margin-top:8px">共 {{ itemsTotal }} 条</p>
        </NTabPane>
        <NTabPane name="issuances" tab="提取记录">
          <NButton size="small" @click="loadIssuances" style="margin-bottom:12px">刷新记录</NButton>
          <ViewportTable :max-height="440" remote :pagination="{page:issuancePage,pageSize:50,itemCount:issuTotal,pageSlot:5,onUpdatePage:(p:number)=>{issuancePage=p;loadIssuances()}}" :columns="issuanceCols" :data="issuances" :loading="issLoading" :scroll-x="600" :bordered="false" size="small" />
          <p v-if="issuTotal" style="font-size:12px;color:#6b7f88;margin-top:8px">共 {{ issuTotal }} 条</p>
        </NTabPane>
      </NTabs>
    </NCard>

    <NModal :show="!!editProject" @update:show="v=>{if(!v)editProject=null}" preset="card" title="项目设置" style="width:min(480px,94vw)" :mask-closable="false">
      <NForm label-placement="top">
        <NFormItem label="项目标题"><NInput v-model:value="projectSettings.title" :maxlength="128" /></NFormItem>
        <NFormItem :label="editProject?.password_configured?'提取口令 · 已设置':'提取口令 · 未设置'"><NSelect v-model:value="projectSettings.passwordMode" :options="[{label:'保持当前口令',value:'keep'},{label:'设置新口令',value:'change'},{label:'取消口令',value:'clear'}]" /></NFormItem>
        <NFormItem v-if="projectSettings.passwordMode==='change'" label="新口令"><NInput v-model:value="projectSettings.password" type="password" show-password-on="click" :maxlength="128" /></NFormItem>
        <NFormItem label="允许重复提取"><NSwitch v-model:value="projectSettings.repeatPolicy" active-value="allow" inactive-value="never" /></NFormItem>
        <NFormItem v-if="projectSettings.repeatPolicy==='allow'" label="提取间隔（秒）"><NInputNumber v-model:value="projectSettings.repeatInterval" :min="0" /></NFormItem>
        <NFormItem label="启用"><NSwitch v-model:value="projectSettings.status" active-value="active" inactive-value="disabled" /></NFormItem>
        <NAlert v-if="editError" type="error">{{editError}}</NAlert>
      </NForm>
      <template #footer><div class="card-actions"><NButton @click="editProject=null">取消</NButton><NButton type="primary" :loading="editBusy" @click="saveSettings">保存</NButton></div></template>
    </NModal>

    <!-- 创建项目弹窗 -->
    <NModal v-model:show="showCreate" preset="card" title="创建卡密项目" style="width:min(480px,94vw)" :mask-closable="false">
      <NForm ref="createFormRef" :model="createForm" label-placement="top">
        <NFormItem label="项目标题" path="title" :rule="{required:true,trigger:'blur'}"><NInput v-model:value="createForm.title" placeholder="如：2026 会员兑换码" /></NFormItem>
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
.project-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(min(100%,280px),1fr));gap:16px;margin-bottom:16px}
.project-card{cursor:pointer}.project-card:hover{transform:translateY(-2px);box-shadow:0 4px 16px rgba(16,42,51,.1)}
.pc-head{display:flex;justify-content:space-between;align-items:center;margin-bottom:4px}
.bar-track{height:6px;border-radius:99px;background:#e8edf0;overflow:hidden}.bar-fill{height:100%;border-radius:99px;background:#0f766e;display:block}
.add-card{border:1.5px dashed #c2cdd2;background:#f8fafb;display:flex;align-items:center;justify-content:center;min-height:140px;font-size:14px;color:#6b7f88;cursor:pointer}
.publish-row{display:flex;flex-wrap:wrap;gap:10px;margin:12px 0;align-items:center}.publish-row .n-select{flex:1;min-width:180px}.claim-url{flex:1;min-width:0;overflow-wrap:anywhere}
@media(max-width:768px){.publish-row .n-select{flex-basis:100%}.card-actions{flex-wrap:wrap}}
</style>
