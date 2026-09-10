<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue';
import { ExternalLink, Pencil, Plus, RefreshCw, Trash2 } from '@lucide/vue';
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
  NInputNumber,
  NModal,
  NPopconfirm,
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
  createLandingPage,
  deleteLandingPage,
  previewLandingDraft,
  listLandingPages,
  type CreateLandingPagePayload,
  type LandingPageItem,
} from '../api';
import { useAuthStore } from '../stores/auth';
import { useDomainStore } from '../stores/domain';
import MaterialPicker from '../components/MaterialPicker.vue';

const message = useMessage();
const auth = useAuthStore();
const domains = useDomainStore();

const items = ref<LandingPageItem[]>([]);
const loading = ref(false);
const showModal = ref(false);
const editing = ref<LandingPageItem | null>(null);
const preview = ref('');
const previewShow = ref(false);
const previewBusy = ref(false);
const saving = ref(false);
const modalError = ref('');
const formRef = ref<FormInst | null>(null);

type TemplateKind = 'liveqr' | 'redirect_notice' | 'custom' | 'kf' | 'kami';

const form = reactive({
  template: 'liveqr' as TemplateKind,
  title: '',
  domainId: null as number | null,
  // liveqr
  headline: '',
  subtext: '',
  footer: '',
  color: '#0f766e',
  showLogo: false,
  logoUrl: '',
  // redirect_notice
  message: '',
  buttonText: '继续访问',
  countdown: 5,
  showTargetUrl: true,
  // custom
  html: '',
  // kf
  kfSubtext: '长按识别二维码，添加客服微信',
  kfFooter: '工作时间内回复更快',
  safetyTip: '',
  // kami
  announcement: '',
  kamiButtonText: '立即领取',
  kamiProjectId: null as number | null,
});

const templateMeta: Record<TemplateKind, { label: string; desc: string; tag: 'success' | 'warning' | 'info' | 'error' }> = {
  liveqr: { label: '群活码页', desc: '展示群二维码，配合活码链接使用', tag: 'success' },
  kf: { label: '客服码页', desc: '客服二维码 + 微信号复制 + 在线状态', tag: 'info' },
  redirect_notice: { label: '跳转提示页', desc: '「即将前往外站」倒计时中转页', tag: 'warning' },
  kami: { label: '卡密提取页', desc: '访客一键领取卡密，支持提取口令', tag: 'error' },
  custom: { label: '自定义页', desc: '自由 HTML 托管（已做 XSS 防护）', tag: 'info' },
};

const canWrite = computed(() => auth.isSuperAdmin || auth.user?.role === 'admin');
const landingDomainOptions = computed(() => domains.landingDomains.map((d) => ({ label: d.Host, value: d.ID })));

const rules: FormRules = {
  title: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  domainId: [{ required: true, type: 'number', message: '请选择落地域名', trigger: 'change' }],
  headline: [
    {
      validator: (_r, _v: string) => (form.template !== 'liveqr' || form.headline.trim() ? true : new Error('请输入主标题')),
      trigger: 'blur',
    },
  ],
  html: [
    {
      validator: (_r, _v: string) => (form.template !== 'custom' || form.html.trim() ? true : new Error('请输入页面 HTML 内容')),
      trigger: 'blur',
    },
  ],
  kamiProjectId: [
    {
      validator: (_r, _v: number | null) => (form.template !== 'kami' || form.kamiProjectId ? true : new Error('请填写卡密项目 ID')),
      trigger: ['blur', 'change'],
    },
  ],
};

function buildPayload(): CreateLandingPagePayload {
  const base = { title: form.title, domain_id: form.domainId!, template: form.template };
  if (form.template === 'liveqr') {
    return { ...base, content: { headline: form.headline, subtext: form.subtext, footer_text: form.footer, theme_color: form.color, show_logo: form.showLogo, logo_url: form.showLogo ? form.logoUrl : '' } };
  }
  if (form.template === 'kf') {
    return { ...base, content: { headline: form.headline, subtext: form.kfSubtext, footer_text: form.kfFooter, theme_color: form.color, safety_tip: form.safetyTip } };
  }
  if (form.template === 'redirect_notice') {
    return { ...base, content: { message: form.message, button_text: form.buttonText, countdown: form.countdown, show_target_url: form.showTargetUrl, theme_color: form.color } };
  }
  if (form.template === 'kami') {
    return { ...base, content: { announcement: form.announcement, button_text: form.kamiButtonText, theme_color: form.color, project_id: form.kamiProjectId } };
  }
  return { ...base, content: { html: form.html, theme_color: form.color } };
}

const columns: DataTableColumns<LandingPageItem> = [
  { title: 'ID', key: 'ID', width: 70, render: (row) => h('code', {}, row.ID) },
  { title: '名称', key: 'Title' },
  {
    title: '模板',
    key: 'Template',
    width: 110,
    render: (row) => h(NTag, { type: templateMeta[row.Template as TemplateKind]?.tag || 'default', size: 'small' }, () => templateMeta[row.Template as TemplateKind]?.label || row.Template),
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
    width: 210,
    render: (row) =>
      h('div', { style: 'display:flex;gap:10px;align-items:center' }, [
        h(NButton, { text: true, type: 'primary', onClick: () => showPreview(row) }, { icon: () => h(NIcon, { size: 14 }, { default: () => h(ExternalLink) }), default: () => '预览' }),
        canWrite.value ? h(NButton, { text: true, onClick: () => openEdit(row) }, { icon: () => h(NIcon, { size: 14 }, { default: () => h(Pencil) }), default: () => '编辑' }) : null,
        canWrite.value
          ? h(
              NPopconfirm,
              { onPositiveClick: () => removePage(row) },
              {
                trigger: () => h(NButton, { text: true, type: 'error' }, { icon: () => h(NIcon, { size: 14 }, { default: () => h(Trash2) }) }),
                default: () => `确认删除「${row.Title}」？被活码引用时将无法删除。`,
              },
            )
          : null,
      ]),
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
  editing.value = null;
  Object.assign(form, {
    template: 'liveqr',
    title: '',
    domainId: domains.landingDomains[0]?.ID ?? null,
    headline: '',
    subtext: '',
    footer: '长按识别二维码',
    color: '#0f766e',
    showLogo: false,
    logoUrl: '',
    message: '您即将离开本页面，前往外部网站',
    buttonText: '继续访问',
    countdown: 5,
    showTargetUrl: true,
    html: '',
    kfSubtext: '长按识别二维码，添加客服微信',
    kfFooter: '工作时间内回复更快',
    safetyTip: '',
    announcement: '',
    kamiButtonText: '立即领取',
    kamiProjectId: null,
  });
  modalError.value = '';
  showModal.value = true;
}

function openEdit(row: LandingPageItem) {
  const c = row.Content || {};
  editing.value = row;
  Object.assign(form, {
    template: row.Template as TemplateKind,
    title: row.Title,
    domainId: row.DomainID,
    headline: String(c.headline || row.Title),
    subtext: String(c.subtext || ''),
    footer: String(c.footer_text || ''),
    color: String(c.theme_color || '#0f766e'),
    showLogo: Boolean(c.show_logo),
    logoUrl: String(c.logo_url || ''),
    message: String(c.message || ''),
    buttonText: String(c.button_text || '继续访问'),
    countdown: Number(c.countdown ?? 5),
    showTargetUrl: c.show_target_url !== false,
    html: String(c.html || ''),
    kfSubtext: String(c.subtext || '长按识别二维码，添加客服微信'),
    kfFooter: String(c.footer_text || '工作时间内回复更快'),
    safetyTip: String(c.safety_tip || ''),
    announcement: String(c.announcement || ''),
    kamiButtonText: String(c.button_text || '立即领取'),
    kamiProjectId: c.project_id != null ? Number(c.project_id) : null,
  });
  modalError.value = '';
  showModal.value = true;
}

async function submit() {
  if (saving.value) return;
  modalError.value = '';
  try {
    await formRef.value?.validate();
  } catch {
    return;
  }
  saving.value = true;
  try {
    const payload = buildPayload();
    if (editing.value) await request(`/api/v1/landing-pages/${editing.value.ID}`, { method: 'PUT', body: JSON.stringify(payload) });
    else await createLandingPage(payload);
    message.success(editing.value ? '落地页已更新' : '落地页已创建');
    showModal.value = false;
    await refresh();
  } catch (err) {
    modalError.value = messageOf(err);
  } finally {
    saving.value = false;
  }
}

async function showPreview(row: LandingPageItem) {
  try {
    preview.value = (await request<{ html: string }>(`/api/v1/landing-pages/${row.ID}/preview`)).html;
    previewShow.value = true;
  } catch (e) {
    message.error(String(e));
  }
}

// P1：编辑中草稿实时预览（手机宽度容器）
async function previewDraft() {
  if (previewBusy.value) return;
  previewBusy.value = true;
  try {
    preview.value = await previewLandingDraft(buildPayload());
    previewShow.value = true;
  } catch (e) {
    message.error(String(e));
  } finally {
    previewBusy.value = false;
  }
}

async function removePage(row: LandingPageItem) {
  try {
    await deleteLandingPage(row.ID);
    message.success(`已删除「${row.Title}」`);
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
    <NCard>
    <template #header>
      <div class="card-head">
        <div>
          <strong>落地页</strong>
          <p class="muted">活码展示页、跳转中转页与自定义页面的公开模板</p>
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

    <NDataTable :columns="columns" :data="items" :loading="loading" :pagination="{ pageSize: 20 }" :scroll-x="800" :bordered="false" size="small">
      <template #empty>
        <NEmpty description="尚未创建落地页">
          <template v-if="canWrite && landingDomainOptions.length" #extra>
            <NButton type="primary" size="small" @click="openCreate">创建第一个落地页</NButton>
          </template>
        </NEmpty>
      </template>
    </NDataTable>
  </NCard>

  <!-- 手机宽度预览容器 -->
  <NModal v-model:show="previewShow" preset="card" title="页面预览（手机效果）" style="width:min(480px,94vw)">
    <NAlert type="info" :show-icon="true" style="margin-bottom:12px">预览不产生访问统计；活码页在预览中不分配真实群二维码。</NAlert>
    <div class="phone-frame"><iframe title="落地页预览" sandbox="" :srcdoc="preview" style="width:100%;height:100%;border:0"/></div>
  </NModal>

  <NModal v-model:show="showModal" preset="card" :title="editing?'编辑落地页':'创建落地页'" style="width:min(680px,94vw)" :mask-closable="false" :closable="!saving" :close-on-esc="!saving">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="top">
      <NFormItem v-if="!editing" label="页面模板">
        <div class="tpl-picker">
          <button
            v-for="(meta, kind) in templateMeta"
            :key="kind"
            type="button"
            class="tpl-card"
            :class="{ sel: form.template === kind }"
            @click="form.template = kind"
          >
            <strong>{{ meta.label }}</strong>
            <span>{{ meta.desc }}</span>
          </button>
        </div>
      </NFormItem>
      <NFormItem v-else label="页面模板">
        <NTag :type="templateMeta[form.template]?.tag || 'default'">{{ templateMeta[form.template]?.label }}（创建后不可更换）</NTag>
      </NFormItem>

      <div class="form-row">
        <NFormItem label="名称" path="title">
          <NInput v-model:value="form.title" placeholder="便于管理端识别" />
        </NFormItem>
        <NFormItem label="落地域名" path="domainId">
          <NSelect :disabled="!!editing" v-model:value="form.domainId" :options="landingDomainOptions" placeholder="选择落地域名" :loading="domains.loading" />
        </NFormItem>
      </div>

      <!-- 群活码页字段 -->
      <template v-if="form.template === 'liveqr'">
        <NFormItem label="主标题" path="headline">
          <NInput v-model:value="form.headline" placeholder="扫码后看到的标题" />
        </NFormItem>
        <NFormItem label="说明">
          <NInput v-model:value="form.subtext" type="textarea" :rows="2" placeholder="标题下方的补充说明" />
        </NFormItem>
        <div class="form-row">
          <NFormItem label="页脚引导语">
            <NInput v-model:value="form.footer" placeholder="默认：长按识别二维码" />
          </NFormItem>
          <NFormItem label="主题色">
            <NColorPicker v-model:value="form.color" :show-alpha="false" :modes="['hex']" style="width: 100%" />
          </NFormItem>
        </div>
        <NFormItem label="显示 Logo">
          <div class="logo-row">
            <NSwitch v-model:value="form.showLogo" />
            <NInput v-if="form.showLogo" v-model:value="form.logoUrl" placeholder="Logo 图片地址，或从素材库选择" />
            <MaterialPicker v-if="form.showLogo" relative @select="form.logoUrl = $event" />
          </div>
        </NFormItem>
      </template>

      <!-- 客服码页字段 -->
      <template v-else-if="form.template === 'kf'">
        <NFormItem label="主标题" path="headline">
          <NInput v-model:value="form.headline" placeholder="扫码后看到的标题，如「添加专属客服」" />
        </NFormItem>
        <NFormItem label="说明">
          <NInput v-model:value="form.kfSubtext" type="textarea" :rows="2" placeholder="标题下方的引导文案" />
        </NFormItem>
        <NFormItem label="微信号备注（在二维码配置中按客服设置）">
          <NAlert type="info" :show-icon="true">每个客服的微信号存在「二维码配置 → 微信号」字段；展示页会自动提供一键复制。</NAlert>
        </NFormItem>
        <div class="form-row">
          <NFormItem label="页脚提示">
            <NInput v-model:value="form.kfFooter" placeholder="工作时间内回复更快" />
          </NFormItem>
          <NFormItem label="主题色">
            <NColorPicker v-model:value="form.color" :show-alpha="false" :modes="['hex']" style="width: 100%" />
          </NFormItem>
        </div>
        <NFormItem label="安全提示（可选）">
          <NInput v-model:value="form.safetyTip" type="textarea" :rows="2" placeholder="如：谨防冒充客服的诈骗行为，平台不会索要验证码" />
        </NFormItem>
      </template>

      <!-- 跳转提示页字段 -->
      <template v-else-if="form.template === 'redirect_notice'">
        <NFormItem label="提示文案">
          <NInput v-model:value="form.message" type="textarea" :rows="2" placeholder="默认：您即将离开本页面，前往外部网站" />
        </NFormItem>
        <div class="form-row">
          <NFormItem label="按钮文案">
            <NInput v-model:value="form.buttonText" placeholder="继续访问" />
          </NFormItem>
          <NFormItem label="倒计时（秒）">
            <NInputNumber v-model:value="form.countdown" :min="0" :max="60" style="width:100%" />
          </NFormItem>
        </div>
        <NFormItem label="展示目标地址">
          <NSwitch v-model:value="form.showTargetUrl" />
        </NFormItem>
        <NFormItem label="主题色">
          <NColorPicker v-model:value="form.color" :show-alpha="false" :modes="['hex']" style="width: 100%" />
        </NFormItem>
      </template>

      <!-- 卡密提取页字段 -->
      <template v-else-if="form.template === 'kami'">
        <NFormItem label="公告文案">
          <NInput v-model:value="form.announcement" type="textarea" :rows="2" placeholder="点击下方按钮领取卡密，领取后请妥善保存。" />
        </NFormItem>
        <div class="form-row">
          <NFormItem label="按钮文案">
            <NInput v-model:value="form.kamiButtonText" placeholder="立即领取" />
          </NFormItem>
          <NFormItem label="主题色">
            <NColorPicker v-model:value="form.color" :show-alpha="false" :modes="['hex']" style="width: 100%" />
          </NFormItem>
        </div>
        <NFormItem label="卡密项目 ID" path="kamiProjectId">
          <NInputNumber v-model:value="form.kamiProjectId" :min="1" :precision="0" style="width:100%" placeholder="在「卡密分发」页查看项目 ID" />
        </NFormItem>
        <NAlert type="info" :show-icon="true">绑定项目后，访客打开此页点击按钮即可原子领取一张未发放卡密；口令与频率限制由项目配置控制。</NAlert>
      </template>

      <!-- 自定义页字段 -->
      <template v-else>
        <NFormItem label="页面 HTML" path="html">
          <NInput
            v-model:value="form.html"
            type="textarea"
            :rows="10"
            placeholder="<div style='font-family:sans-serif'>...</div> ｜ script 标签、javascript: 协议与 on* 事件会被自动过滤"
            class="html-input"
          />
        </NFormItem>
        <NFormItem label="主题色">
          <NColorPicker v-model:value="form.color" :show-alpha="false" :modes="['hex']" style="width: 100%" />
        </NFormItem>
      </template>

      <NAlert v-if="modalError" type="error" :show-icon="true" style="margin-bottom: var(--space-3)">{{ modalError }}</NAlert>
    </NForm>

    <template #footer>
      <div style="display: flex; justify-content: space-between; gap: var(--space-2)">
        <NButton :loading="previewBusy" @click="previewDraft">预览（手机效果）</NButton>
        <div style="display:flex;gap:var(--space-2)">
          <NButton :disabled="saving" @click="showModal = false">取消</NButton>
          <NButton type="primary" :loading="saving" @click="submit">{{editing?'保存修改':'创建'}}</NButton>
        </div>
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

.tpl-picker {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: var(--space-3);
  width: 100%;
}

.tpl-card {
  border: 1.5px solid var(--color-border, #e3eaed);
  border-radius: 10px;
  background: #fff;
  padding: 12px;
  text-align: left;
  cursor: pointer;
  display: grid;
  gap: 4px;
}

.tpl-card strong { font-size: 13px; }
.tpl-card span { font-size: 12px; color: var(--color-text-secondary, #6b7f88); line-height: 1.6; }

.tpl-card.sel {
  border-color: var(--color-primary, #0f766e);
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.12);
}

.logo-row { display: flex; gap: 10px; align-items: center; width: 100%; }
.logo-row > .n-input { flex: 1; }

.html-input :deep(textarea) { font-family: Consolas, Menlo, monospace; font-size: 12.5px; }

.phone-frame {
  width: 375px;
  max-width: 100%;
  height: 640px;
  margin: 0 auto;
  border: 10px solid #16262d;
  border-radius: 26px;
  overflow: hidden;
  background: #fff;
}

@media (max-width: 640px) {
  .form-row { grid-template-columns: 1fr; }
  .tpl-picker { grid-template-columns: 1fr; }
}
</style>
