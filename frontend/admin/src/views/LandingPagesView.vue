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
import { createLandingPage, listLandingPages, type LandingPageItem } from '../api';
import { useAuthStore } from '../stores/auth';
import { useDomainStore } from '../stores/domain';

const message = useMessage();
const auth = useAuthStore();
const domains = useDomainStore();

const items = ref<LandingPageItem[]>([]);
const loading = ref(false);
const showModal = ref(false);
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
      const domain = domains.items.find((d) => d.ID === row.DomainID);
      if (!domain) return h('span', { class: 'muted' }, '—');
      return h(
        NButton,
        {
          text: true,
          size: 'small',
          tag: 'a',
          href: `${domain.Scheme}://${domain.Host}/`,
          target: '_blank',
          title: '访问落地页域名',
        },
        { icon: () => h(NIcon, { size: 14 }, { default: () => h(ExternalLink) }) },
      );
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
  modalError.value = '';
  try {
    await formRef.value?.validate();
  } catch {
    return;
  }
  saving.value = true;
  try {
    await createLandingPage({
      template: 'liveqr',
      title: form.title,
      domain_id: form.domainId!,
      content: {
        headline: form.headline,
        subtext: form.subtext,
        footer_text: form.footer,
        theme_color: form.color,
      },
    });
    message.success('落地页已创建');
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

  <NModal v-model:show="showModal" preset="card" title="创建落地页" style="width: 620px" :mask-closable="false">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="top">
      <div class="form-row">
        <NFormItem label="名称" path="title">
          <NInput v-model:value="form.title" placeholder="便于管理端识别" />
        </NFormItem>
        <NFormItem label="落地域名" path="domainId">
          <NSelect v-model:value="form.domainId" :options="landingDomainOptions" placeholder="选择落地域名" :loading="domains.loading" />
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
        <NButton @click="showModal = false">取消</NButton>
        <NButton type="primary" :loading="saving" @click="submit">创建</NButton>
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

@media (max-width: 640px) {
  .form-row {
    grid-template-columns: 1fr;
  }
}
</style>
