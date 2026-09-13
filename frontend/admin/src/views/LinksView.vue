<script setup lang="ts">
import LinkStatsDrawer from '../components/LinkStatsDrawer.vue';
import ViewportTable from '../components/ViewportTable.vue';
import { computed, h, onMounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { BarChart3, Copy, ExternalLink, Pencil, Plus, RefreshCw, Search, Send, Trash2 } from '@lucide/vue';
import {
  NAlert,
  NCheckbox,
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
  NPagination,
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
  request,
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
import LabelManager from '../components/LabelManager.vue';
import OnlineSchedulePicker from '../components/OnlineSchedulePicker.vue';

const route = useRoute();
const statsLink=ref<LinkItem|null>(null),statsOpen=ref(false);
function openStats(link:LinkItem){statsLink.value=link;statsOpen.value=true}
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
const categoryFilter=ref<string|null>(null),tagFilter=ref<string|null>(null),minViews=ref<number|null>(null),sort=ref('new');
const catalog=ref<{categories:string[];tags:string[]}>({categories:[],tags:[]});
const categoryOptions=computed(()=>[...new Set([...catalog.value.categories,...items.value.map(i=>i.Category||'').filter(Boolean)])].sort().map(v=>({label:v,value:v})));
const tagOptions=computed(()=>[...new Set([...catalog.value.tags,...items.value.flatMap(i=>i.Tags||[])])].sort().map(v=>({label:v,value:v})));
const showLabels=ref(false),advanced=ref(false),checked=ref<number[]>([]),showBulk=ref(false),bulkBusy=ref(false),bulkError=ref('');
const bulk=reactive({changeCategory:false,category:'',tagMode:'',tags:[] as string[],status:null as string|null});
async function loadCatalog(){try{catalog.value=await request('/api/v1/link-labels')}catch(e){message.error(messageOf(e))}}
async function labelsSaved(){await Promise.all([loadCatalog(),refresh()])}
function openBulk(){Object.assign(bulk,{changeCategory:false,category:'',tagMode:'',tags:[],status:null});bulkError.value='';showBulk.value=true}
async function applyBulk(){
 bulkBusy.value=true;bulkError.value='';
 try{await request('/api/v1/links/bulk',{method:'POST',body:JSON.stringify({ids:checked.value,category:bulk.changeCategory?bulk.category:undefined,tag_mode:bulk.tagMode,tags:bulk.tags,status:bulk.status||undefined})});await labelsSaved();checked.value=[];showBulk.value=false;message.success('所选链接已更新')}
 catch(e){bulkError.value=messageOf(e)}finally{bulkBusy.value=false}
}
watch([query,typeFilter,statusFilter,categoryFilter,tagFilter,minViews,sort],()=>checked.value=[]);
onMounted(loadCatalog);
const kindOptions=[{label:'短链接',value:'short'},{label:'渠道链接',value:'channel'},{label:'群活码',value:'liveqr'},{label:'客服码',value:'kf'},{label:'卡密提取',value:'kami'}];
function kindLabel(item:LinkItem){return kindOptions.find(o=>o.value===(item.Kind||item.Type))?.label||'活码'}


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
  title: '', category:'', tags:[] as string[],
  code: '',
  entryDomainId: null as number | null,
  targetUrl: '',
  accessRule: 'none' as 'none' | 'wechat' | 'ios' | 'android' | 'mobile' | 'pc',
  onlineSchedule: '',
  strategyMode: 'round_robin' as 'round_robin' | 'weighted',
  targets: [{ label: '', url: '', weight: 1, scanLimit: null as number | null, wxRemark: '' }],
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
    if (typeFilter.value && (item.Kind||item.Type) !== typeFilter.value) return false;
    if(categoryFilter.value && (categoryFilter.value==='__none__'?!!item.Category:item.Category!==categoryFilter.value))return false;
 if(tagFilter.value && !item.Tags?.includes(tagFilter.value))return false;
 if(minViews.value!==null&&(item.Views||0)<minViews.value)return false;
 if (!value) return true;
    return `${item.Code} ${item.Title || ''} ${item.TargetURL || ''}`.toLowerCase().includes(value);
  }).sort((a,b)=>sort.value==='views-desc'?(b.Views||0)-(a.Views||0):sort.value==='views-asc'?(a.Views||0)-(b.Views||0):b.ID-a.ID);
});

const landingPageOptions = computed(() =>
  landingPages.value
    .filter((p) => p.Template === 'liveqr' || p.Template === 'kf' || p.Template === 'kami')
    .map((p) => ({ label: `${p.Title}（${landingTemplateLabel(p.Template)}）`, value: p.ID })),
);

function landingTemplateLabel(t: string): string {
  if (t === 'kf') return '客服码';
  if (t === 'kami') return '卡密提取';
  return '活码';
}

const selectedLandingTemplate = computed(() => {
  const page = landingPages.value.find((p) => p.ID === landingPageId.value);
  return page?.Template || '';
});

const isKfLanding = computed(() => selectedLandingTemplate.value === 'kf');
const isKamiLanding = computed(() => selectedLandingTemplate.value === 'kami');
const needsTargets = computed(() => form.type === 'liveqr' && !isKamiLanding.value);

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
  {type:'selection',fixed:'left',width:40,disabled:()=>!canWrite.value},
  { title: '链接', key: 'Title', width: 300,
    render: row => h('div', {class:'link-identity'}, [
      h('strong', {title:row.Title || row.Code}, row.Title || row.Code),
      h('a', {href:shortUrlOf(row) || undefined,target:'_blank',rel:'noopener noreferrer',title:shortUrlOf(row)}, shortUrlOf(row) || row.Code),
    ]),
  },
  {
    title: '类型',
    key: 'Type',
    width: 100,
    render: (row) => kindLabel(row) || row.Type,
  },
 {title:'分类 / 标签',key:'labels',width:160,render:row=>h('div',{style:'display:flex;gap:4px;flex-wrap:wrap'},[h('span',row.Category||'未分类'),...(row.Tags||[]).map(t=>h(NTag,{size:'small'},()=>t))])},
 {title:'访问量',key:'Views',width:85,render:row=>(row.Views||0).toLocaleString()},
  {
    title: '目标',
    key: 'TargetURL',
    width: 240,
    ellipsis: { tooltip: true },
    render: (row) => row.TargetURL || h('span', { class: 'muted' }, row.Kind==='kami'?'卡密领取页':'二维码轮换'),
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
    fixed: 'right',
    width: 70,
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
    fixed: 'right',
    width: 220,
    render: (row) =>
      h('div', { style: 'display:flex;gap:6px;align-items:center' }, [
        h(
          NButton,
          { text: true, size: 'small', title: '分享（链接+二维码+数据）', disabled: !shortUrlOf(row), onClick: () => { drawerLink.value = row; showDrawer.value = true; } },
          { icon: () => h(NIcon, { size: 15 }, { default: () => h(Send) }) },
        ),
        h(
          NButton,
          { text: true, size: 'small', title: '查看统计', onClick: () => openStats(row) },
          { icon: () => h(NIcon, { size: 15 }, { default: () => h(BarChart3) }) },
        ),
        h(EntryQRCode, {url:shortUrlOf(row),name:row.Code}),
        h('span', { style:'display:inline-flex;width:20px;justify-content:center' }, canWrite.value && row.Type === 'liveqr' && row.Kind !== 'kami' ? [h(TargetManager, { linkId: row.ID, origin: originOf(row) })] : []),
        h(
          NButton,
          { text: true, size: 'small', title: '打开访问地址', disabled: !shortUrlOf(row), tag: 'a', href: shortUrlOf(row) || undefined, target: '_blank', rel: 'noopener noreferrer' },
          { icon: () => h(NIcon, { size: 15 }, { default: () => h(ExternalLink) }) },
        ),
        canWrite.value
          ? h(
              NButton,
              { text: true, size: 'small', title: '编辑链接', onClick: () => openEdit(row) },
              { icon: () => h(NIcon, { size: 15 }, { default: () => h(Pencil) }) },
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
                    { text: true, size: 'small', title: '删除链接', type: 'error' },
                    { icon: () => h(NIcon, { size: 15 }, { default: () => h(Trash2) }) },
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
    title: '', category:'', tags:[] as string[],
    code: '',
    entryDomainId: null,
    targetUrl: '',
    accessRule: 'none',
    onlineSchedule: '',
    strategyMode: 'round_robin',
    targets: [{ label: '', url: '', weight: 1, scanLimit: null, wxRemark: '' }],
    expireAt: null,
    status: 'active',
  });
  modalError.value = '';
  showModal.value = true;
}

function openEdit(row: LinkItem) {
  landingPageId.value=row.LandingPageID||null;
  modalMode.value = 'edit';
  editingId.value = row.ID;
  Object.assign(form, {
    type: row.Type as typeof form.type,
    title: row.Title || '', category:row.Category||'',tags:[...(row.Tags||[])],
    code: row.Code,
    entryDomainId: row.EntryDomainID,
    targetUrl: row.TargetURL || '',
    strategyMode: 'round_robin',
    targets: [{ label: '', url: '', weight: 1, scanLimit: null, wxRemark: '' }],
    expireAt: row.ExpireAt?Date.parse(row.ExpireAt):null,
    onlineSchedule:row.OnlineSchedule||'',
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
      const page = landingPages.value.find(p => p.ID === landingPageId.value);
      if (form.type === 'liveqr' && !page) throw new Error('请选择活码落地页；没有可选项时请先创建落地页');
      if (form.type === 'liveqr' && page && !['liveqr', 'kf', 'kami'].includes(page.Template)) throw new Error('所选落地页模板不支持活码链接');
      const created = await createLink({
        landing_page_id: form.type === 'liveqr' ? page?.ID : undefined,
        landing_domain_id: form.type === 'liveqr' ? page?.DomainID : undefined,
        type: form.type,
        code: form.code || undefined,
        title: form.title || undefined, category:form.category,tags:form.tags,
        entry_domain_id: form.entryDomainId!,
        target_url: form.type === 'liveqr' ? '' : form.targetUrl,
        access_rule: form.accessRule !== 'none' ? form.accessRule : undefined,
        online_schedule: isKfLanding.value && form.onlineSchedule ? form.onlineSchedule : undefined,
        expire_at: form.expireAt ? new Date(form.expireAt).toISOString() : undefined,
        strategy:
          form.type === 'liveqr' && needsTargets.value
            ? {
                mode: form.strategyMode,
                targets: form.targets.map((item) => ({
                  label: item.label.trim() || undefined,
                  target_url: item.url.trim(),
                  weight: form.strategyMode === 'weighted' ? item.weight : 1,
                  scan_limit: item.scanLimit ?? undefined,
                  wx_remark: item.wxRemark.trim() || undefined,
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
      const origRow = items.value.find((i) => i.ID === editingId.value);
      const codeChanged = origRow && form.code.trim() && form.code.trim() !== origRow.Code;
      const domainChanged = origRow && form.entryDomainId && form.entryDomainId !== origRow.EntryDomainID;
      await updateLink(editingId.value, {
        title: form.title, category:form.category,tags:form.tags,
        target_url: form.type === 'liveqr' ? undefined : form.targetUrl,
        expire_at: form.expireAt ? new Date(form.expireAt).toISOString() : null,
        status: form.status,
        online_schedule: isKfLanding.value ? form.onlineSchedule : undefined,
        code: codeChanged ? form.code.trim() : undefined,
        entry_domain_id: domainChanged ? form.entryDomainId! : undefined,
      });
      message.success(codeChanged ? `链接已更新，旧码 ${origRow!.Code} 将自动跳转到新码` : '链接已更新');
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
  form.targets.push({ label: '', url: '', weight: 1, scanLimit: null, wxRemark: '' });
}

function removeTarget(index: number) {
  if (form.targets.length === 1) return;
  form.targets.splice(index, 1);
}
function originOf(link:LinkItem) { const url = shortUrlOf(link); return url ? new URL(url).origin : ''; }
const mobilePage = ref(1);
const mobileLinks = computed(() => filtered.value.slice((mobilePage.value-1)*20,mobilePage.value*20));
watch([query,typeFilter,statusFilter,categoryFilter,tagFilter,minViews,sort], () => { mobilePage.value = 1; });

</script>

<template>
  <div class="page-view">
    <NCard>
    <template #header>
      <div class="card-head">
        <div>
          <strong>链接 <span class="muted" style="font-size:12px;font-weight:400">{{filtered.length}} 条</span></strong>
          
        </div>
        <div class="toolbar-actions"><NButton v-if="canWrite" size="small" @click="showLabels=true">分类与标签</NButton><BatchAdd v-if="canWrite" :domains="domains.items" @saved="refresh"/><NButton v-if="canWrite" type="primary" size="small" @click="openCreate">创建链接</NButton></div>
      </div>
    </template>

    <!-- 工具栏独立成行，窄屏自动换行（Docs/ui-design-system.md 2026-09-11） -->
    <div class="toolbar">
      <NInput v-model:value="query" class="tb-search" placeholder="搜索短码、名称或目标" clearable>
        <template #prefix><Search :size="14" /></template>
      </NInput>
      <NSelect v-model:value="typeFilter" class="tb-select" :options="kindOptions" placeholder="类型" clearable />
      <NSelect v-model:value="categoryFilter" class="tb-select" :options="[{label:'未分类',value:'__none__'},...categoryOptions]" placeholder="分类" clearable filterable />
      <NButton @click="advanced=!advanced">{{advanced?'收起筛选':'更多筛选'}}</NButton>
      <NButton :loading="loading" title="刷新列表" @click="refresh"><RefreshCw :size="16" /></NButton>
    </div>
    <div v-if="advanced" class="toolbar">
      <NSelect v-model:value="statusFilter" class="tb-select" :options="statusOptions" placeholder="状态" clearable />
      <NSelect v-model:value="tagFilter" class="tb-select" :options="tagOptions" placeholder="标签" clearable filterable />
      <NInputNumber v-model:value="minViews" :min="0" placeholder="最低访问量" style="width:140px" clearable />
      <NSelect v-model:value="sort" class="tb-select" :options="[{label:'最新创建',value:'new'},{label:'访问量从高到低',value:'views-desc'},{label:'访问量从低到高',value:'views-asc'}]" />
      <NButton @click="query='';typeFilter=null;categoryFilter=null;statusFilter=null;tagFilter=null;minViews=null;sort='new'">重置</NButton>
    </div>
    <div v-if="checked.length" class="selection-bar"><span>已选 {{checked.length}} 条</span><NButton size="small" @click="checked=filtered.map(i=>i.ID)">选择全部 {{filtered.length}} 条结果</NButton><NButton type="primary" size="small" @click="openBulk">批量设置</NButton><NButton size="small" @click="checked=[]">取消选择</NButton></div>

    <ViewportTable class="desktop-links"
      :columns="columns" :row-key="(row:LinkItem)=>row.ID" v-model:checked-row-keys="checked"
      :data="filtered"
      :loading="loading"
      :pagination="{ pageSize: 20, showSizePicker: true, pageSizes: [10, 20, 50, 100] }"
      :scroll-x="1485"
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
    </ViewportTable>
    <div class="mobile-links">
      <NEmpty v-if="!mobileLinks.length" :description="loading ? '正在加载' : '没有匹配的链接'" />
      <NPagination v-model:page="mobilePage" :page-size="20" :item-count="filtered.length" :page-slot="3" />
      <article v-for="link in mobileLinks" :key="link.ID" class="mobile-link">
        <div class="mobile-link-heading"><NCheckbox v-if="canWrite" :checked="checked.includes(link.ID)" @update:checked="v=>checked=v?[...checked,link.ID]:checked.filter(id=>id!==link.ID)"/><strong>{{ link.Title || link.Code }}</strong><NSwitch v-if="canWrite" size="small" :value="link.Status==='active'" @update:value="v=>toggleStatus(link,v)"/><NTag v-else size="small">{{statusLabelMap[link.Status]}}</NTag></div>
        <a :href="shortUrlOf(link)" target="_blank" rel="noopener noreferrer">{{ shortUrlOf(link) || link.Code }}</a>
        <p class="muted">{{ kindLabel(link) || link.Type }} · {{ (link.Views||0).toLocaleString() }} 次访问</p><div class="mobile-labels"><NTag size="small">{{link.Category||'未分类'}}</NTag><NTag v-for="tag in link.Tags" :key="tag" size="small">{{tag}}</NTag></div>
        <div class="mobile-link-actions">
          <NButton size="small" :disabled="!shortUrlOf(link)" @click="copyShortUrl(link)">复制</NButton>
          <NButton size="small" @click="drawerLink=link;showDrawer=true">分享</NButton>
          <NButton size="small" @click="openStats(link)">统计</NButton>
          <NButton v-if="canWrite" size="small" @click="openEdit(link)">编辑</NButton>
          <TargetManager v-if="canWrite && link.Type==='liveqr' && link.Kind !== 'kami'" :link-id="link.ID" :origin="originOf(link)" />
        </div>
      </article>

    </div>
  </NCard>

  <LabelManager v-model:show="showLabels" :categories="categoryOptions.map(o=>o.value)" :tags="tagOptions.map(o=>o.value)" @saved="labelsSaved" />
  <NModal v-model:show="showBulk" preset="card" :title="`批量设置 · ${checked.length} 条链接`" style="width:min(500px,94vw)" :mask-closable="false">
   <NForm label-placement="top">
    <NFormItem label="分类"><div style="width:100%"><NCheckbox v-model:checked="bulk.changeCategory">转移分类</NCheckbox><NSelect v-if="bulk.changeCategory" v-model:value="bulk.category" :options="[{label:'未分类',value:''},...categoryOptions]" filterable tag placeholder="选择或输入分类" style="margin-top:8px" /></div></NFormItem>
    <NFormItem label="标签"><NSelect v-model:value="bulk.tagMode" :options="[{label:'保持现有标签',value:''},{label:'追加标签',value:'append'},{label:'移除标签',value:'remove'},{label:'替换全部标签',value:'replace'}]"/></NFormItem>
    <NFormItem v-if="bulk.tagMode" label="选择标签"><NSelect v-model:value="bulk.tags" :options="tagOptions" multiple filterable tag placeholder="选择或输入标签"/></NFormItem>
    <NFormItem label="启用状态"><NSelect v-model:value="bulk.status" :options="statusOptions" clearable placeholder="保持当前状态"/></NFormItem>
    <NAlert v-if="bulkError" type="error">{{bulkError}}</NAlert>
   </NForm>
   <template #footer><NButton type="primary" :disabled="!bulk.changeCategory&&!bulk.tagMode&&!bulk.status" :loading="bulkBusy" @click="applyBulk">保存</NButton></template>
  </NModal>

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

      <NFormItem label="分类"><NSelect v-model:value="form.category" :options="categoryOptions" filterable tag clearable placeholder="选择或输入分类" @update:value="v=>form.category=v||''" /></NFormItem>
      <NFormItem label="标签"><NSelect v-model:value="form.tags" :options="tagOptions" multiple filterable tag placeholder="选择或输入标签" /></NFormItem>
      <NFormItem label="名称">
        <NInput v-model:value="form.title" placeholder="便于管理端识别（可选）" />
      </NFormItem>

      <div class="form-row">
        <NFormItem label="短码">
          <NInput v-model:value="form.code" :placeholder="modalMode === 'create' ? '留空自动生成' : '修改后旧码自动跳转新码'" />
        </NFormItem>
        <NFormItem label="入口域名" :path="modalMode === 'create' ? 'entryDomainId' : undefined">
          <NSelect v-model:value="form.entryDomainId" :options="entryDomainOptions" placeholder="选择入口域名" :loading="domains.loading" />
        </NFormItem>
      </div>
      <NFormItem v-if="modalMode === 'edit'" label="状态">
        <NSelect v-model:value="form.status" :options="statusOptions" />
      </NFormItem>
      <NFormItem v-if="form.type !== 'liveqr'" label="目标 URL" path="targetUrl">
        <NInput v-model:value="form.targetUrl" placeholder="https://example.com/path" />
      </NFormItem>
      <template v-else-if="modalMode === 'create'">
        <NFormItem label="活码落地页（必选）">
          <NSelect v-model:value="landingPageId" :options="landingPageOptions" placeholder="先在落地页管理中创建活码/客服/卡密页面" />
        </NFormItem>
        <template v-if="isKamiLanding">
          <NAlert type="info" :show-icon="true">卡密提取页无需配置二维码目标；访客打开后点击按钮即可领取卡密。</NAlert>
        </template>
        <template v-else>
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
              <div v-for="(target, index) in form.targets" :key="index" class="target-row" :class="{ kf: isKfLanding }">
                <div class="target-index">{{ index + 1 }}</div>
                <NInput v-model:value="target.label" placeholder="名称（可选）" />
                <div><NInput v-model:value="target.url" placeholder="https://example.com/qr.png 或从素材库选择" /><MaterialPicker relative @select="target.url=$event" /></div>
                <NInput v-if="isKfLanding" v-model:value="target.wxRemark" placeholder="微信号（可复制）" />
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

        <NFormItem v-if="isKfLanding" label="在线时段（可选）">
          <OnlineSchedulePicker v-model="form.onlineSchedule" />
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
<LinkStatsDrawer v-model:show="statsOpen" :link="statsLink" />
</template>

<style scoped>
.selection-bar{display:flex;align-items:center;gap:8px;flex-wrap:wrap;padding:8px;background:#eff6ff;margin-bottom:10px;border-radius:6px}

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

.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-4);
}

.tb-search {
  width: 240px;
}

.tb-select {
  width: 120px;
}

.tb-spacer {
  flex: 1 1 var(--space-3);
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

.target-row.kf {
  grid-template-columns: 28px minmax(100px, .5fr) minmax(180px, 1.2fr) minmax(110px, .7fr) minmax(100px, .45fr) 32px;
}

.target-row:has(.n-input-number:nth-of-type(2)) {
  grid-template-columns: 28px minmax(100px, .5fr) minmax(190px, 1.25fr) 90px 110px 32px;
}

.target-row.kf:has(.n-input-number:nth-of-type(2)) {
  grid-template-columns: 28px minmax(90px, .45fr) minmax(160px, 1.1fr) minmax(100px, .65fr) 90px 110px 32px;
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
  .tb-search {
    width: 100%;
  }
  .tb-select {
    width: calc(50% - var(--space-1));
  }
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
.mobile-links{display:none}
.mobile-links :deep(.n-pagination){position:sticky;top:0;z-index:2;background:white;padding:8px 0}
:deep(.link-identity){display:grid;gap:4px;min-width:0}
:deep(.link-identity strong),:deep(.link-identity a){overflow:hidden;text-overflow:ellipsis;white-space:nowrap;display:block}
:deep(.link-identity a){font-size:12px;color:var(--color-text-tertiary)}
@media(max-width:768px){.desktop-links{display:none}.mobile-links{display:grid;gap:14px}.mobile-link{border:1px solid var(--color-border);border-radius:12px;padding:14px;display:grid;gap:10px;min-width:0}.mobile-link>a{overflow-wrap:anywhere;font-size:13px}.mobile-link-heading{display:flex;justify-content:space-between;gap:12px}.mobile-link-heading strong{overflow-wrap:anywhere;min-width:0}.mobile-link-actions{display:flex;gap:8px;flex-wrap:wrap}}
.toolbar-actions{display:flex;gap:8px;margin-left:auto;flex-wrap:wrap}.mobile-labels{display:flex;gap:5px;flex-wrap:wrap}
</style>
