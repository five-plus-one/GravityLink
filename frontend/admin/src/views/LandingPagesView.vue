<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue';
import { ExternalLink, Plus, RefreshCw } from '@lucide/vue';
import {
  NAlert,
  NButton,
  NCard,
  NColorPicker,
  NDataTable,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
  NSelect,
  NTag,
  useMessage,
  type DataTableColumns,
  type FormInst,
  type FormRules,
} from 'naive-ui';
import { request, createLandingPage, listLandingPages, type LandingPageItem } from '../api';
import { useAuthStore } from '../stores/auth';
import { useDomainStore } from '../stores/domain';

const message = useMessage();
const auth = useAuthStore();
const domains = useDomainStore();

const items = ref<LandingPageItem[]>([]);
const loading = ref(false);
const showModal = ref(false);
const editing=ref<LandingPageItem|null>(null),preview=ref(''),previewShow=ref(false);
async function showPreview(row:LandingPageItem){try{preview.value=(await request<{html:string}>(`/api/v1/landing-pages/${row.ID}/preview`)).html;previewShow.value=true;}catch(e){message.error(String(e));}}
function openEdit(row:LandingPageItem){const c=row.Content||{};editing.value=row;Object.assign(form,{title:row.Title,domainId:row.DomainID,headline:String(c.headline||row.Title),subtext:String(c.subtext||''),footer:String(c.footer_text||''),color:String(c.theme_color||'#0f766e')});modalError.value='';showModal.value=true;}
const saving = ref(false);
const modalError = ref('');
const formRef = ref<FormInst | null>(null);

const form = reactive({
  title: '',
  domainId: null as number | null,
  headline: '',
  subtext: '',
  footer: '',
  color: '#0f766e',
});

const canWrite = computed(() => auth.isSuperAdmin || auth.user?.role === 'admin');
const landingDomainOptions = computed(() => domains.landingDomains.map((d) => ({ label: d.Host, value: d.ID })));

const templateLabelMap: Record<string, string> = {
  liveqr: '活码',
  redirect_notice: '跳转提示',
  custom: '自定义',
};
const templateTagMap: Record<string, 'success' | 'warning' | 'info'> = {
  liveqr: 'success',
  redirect_notice: 'warning',
  custom: 'info',
};

const rules: FormRules = {
  title: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  domainId: [{ required: true, type: 'number', message: '请选择落地域名', trigger: 'change' }],
  headline: [{ required: true, message: '请输入主标题', trigger: 'blur' }],
};

const columns: DataTableColumns<LandingPageItem> = [
  { title: 'ID', key: 'ID', width: 70, render: (row) => h('code', {}, row.ID) },
  { title: '名称', key: 'Title' },
  {
    title: '模板',
    key: 'Template',
    width: 110,
    render: (row) =>
      h(NTag, { type: templateTagMap[row.Template] || 'default', size: 'small' }, () => templateLabelMap[row.Template] || row.Template),
  },
  {
    title: '落地域名',
    key: 'DomainID',
    width: 200,
    render: (row) => {
      const domain = domains.items.find((d) => d.ID === row.DomainID);
      return domain ? h('code', {}, `${domain.Scheme}://${domain.Host}`) : h('span', { class: 'muted' }, `ID ${row.DomainID}`);
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 100,
    render: (row) => {
      return h('div',{style:'display:flex;gap:12px'},[h(NButton,{text:true,onClick:()=>showPreview(row)},()=> '布局预览'),canWrite.value&&row.Template==='liveqr'?h(NButton,{text:true,onClick:()=>openEdit(row)},()=> '编辑'):null]);
    },
  },
];

onMounted(async () => {
  await Promise.all([refresh(), domains.refresh()]);
});

async function refresh() {
  loading.value = true;
  try {
    items.value = (await listLandingPages()).items;
  } catch (err) {
    message.error(messageOf(err));
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editing.value=null;
  Object.assign(form, {
    title: '',
    domainId: domains.landingDomains[0]?.ID ?? null,
    headline: '',
    subtext: '',
    footer: '',
    color: '#0f766e',
  });
  modalError.value = '';
  showModal.value = true;
}

async function submit() {
  if(saving.value)return;
  modalError.value = '';
  try {
    await formRef.value?.validate();
  } catch {
    return;
  }
  saving.value = true;
  try {
    const payload = {
      template: 'liveqr' as const,
      title: form.title,
      domain_id: form.domainId!,
      content: {
        ...(editing.value?.Content||{}),
        headline: form.headline,
        subtext: form.subtext,
        footer_text: form.footer,
        theme_color: form.color,
      },
    };
    if(editing.value) await request(`/api/v1/landing-pages/${editing.value.ID}`,{method:'PUT',body:JSON.stringify(payload)});else await createLandingPage(payload);
    message.success(editing.value?'落地页已更新':'落地页已创建');
    showModal.value = false;
    await refresh();
  } catch (err) {
    modalError.value = messageOf(err);
  } finally {
    saving.value = false;
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
          <strong>落地页</strong>
          <p class="muted">配置活码访问时展示的公开页面</p>
        </div>
        <div class="card-actions">
          <NButton :loading="loading" @click="refresh">
            <template #icon><RefreshCw :size="16" /></template>
          </NButton>
          <NButton v-if="canWrite" type="primary" :disabled="!landingDomainOptions.length" @click="openCreate">
            <template #icon><Plus :size="16" /></template>
            创建落地页
          </NButton>
        </div>
      </div>
    </template>

    <NAlert v-if="!landingDomainOptions.length && !loading" type="warning" :show-icon="true" style="margin-bottom: var(--space-4)">
      尚未配置落地域名。请先到「域名」页面添加类型为「落地页」的域名。
    </NAlert>

    <NDataTable :columns="columns" :data="items" :loading="loading" :pagination="{ pageSize: 20 }" :bordered="false" size="small">
      <template #empty>
        <NEmpty description="尚未创建落地页">
          <template v-if="canWrite && landingDomainOptions.length" #extra>
            <NButton type="primary" size="small" @click="openCreate">创建第一个落地页</NButton>
          </template>
        </NEmpty>
      </template>
    </NDataTable>
  </NCard>

  <NModal v-model:show="previewShow" preset="card" title="布局预览" style="width:min(700px,94vw)"><NAlert>仅预览布局，不分配群二维码，不产生访问统计；实际内容请通过活码入口查看。</NAlert><iframe title="落地页布局预览" sandbox="" :srcdoc="preview" style="width:100%;height:520px;border:0"/></NModal>
  <NModal v-model:show="showModal" preset="card" :title="editing?'编辑落地页':'创建落地页'" style="width:min(620px,94vw)" :mask-closable="false" :closable="!saving" :close-on-esc="!saving">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="top">
      <div class="form-row">
        <NFormItem label="名称" path="title">
          <NInput v-model:value="form.title" placeholder="便于管理端识别" />
        </NFormItem>
        <NFormItem label="落地域名" path="domainId">
          <NSelect :disabled="!!editing" v-model:value="form.domainId" :options="landingDomainOptions" placeholder="选择落地域名" :loading="domains.loading" />
        </NFormItem>
      </div>
      <NFormItem label="主标题" path="headline">
        <NInput v-model:value="form.headline" placeholder="扫码后看到的标题" />
      </NFormItem>
      <NFormItem label="说明">
        <NInput v-model:value="form.subtext" type="textarea" :rows="3" placeholder="标题下方的补充说明" />
      </NFormItem>
      <div class="form-row">
        <NFormItem label="页脚">
          <NInput v-model:value="form.footer" placeholder="默认：长按识别二维码" />
        </NFormItem>
        <NFormItem label="主题色">
          <NColorPicker v-model:value="form.color" :show-alpha="false" :modes="['hex']" style="width: 100%" />
        </NFormItem>
      </div>
      <NAlert v-if="modalError" type="error" :show-icon="true" style="margin-bottom: var(--space-3)">{{ modalError }}</NAlert>
    </NForm>

    <template #footer>
      <div style="display: flex; justify-content: flex-end; gap: var(--space-2)">
        <NButton :disabled="saving" @click="showModal = false">取消</NButton>
        <NButton type="primary" :loading="saving" @click="submit">{{editing?'保存修改':'创建'}}</NButton>
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
