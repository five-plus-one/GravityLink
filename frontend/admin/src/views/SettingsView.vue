<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { AlertTriangle, ExternalLink, RotateCcw, Save } from '@lucide/vue';
import {
  NAlert,
  NButton,
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NSkeleton,
  NSwitch,
  NTabPane,
  NTabs,
  useMessage,
} from 'naive-ui';
import { getSystemConfigs, resetSystem, updateSystemConfigs, type AuthConfigStatus } from '../api';
import { clearSession } from '../auth';
import { useAuthStore } from '../stores/auth';

const message = useMessage();
const router = useRouter();
const auth = useAuthStore();

const loading = ref(true);
const saving = ref(false);
const authInfo = ref<AuthConfigStatus | null>(null);

const showResetDialog = ref(false);
const resetConfirmation = ref('');
const resetPassword = ref('');
const resetBusy = ref(false);

const form = reactive({
  siteName: 'GravityLink',
  publicBaseURL: '',
  homeTitle: '链接服务正在运行',
  homeMessage: '这是短链接访问入口，请使用完整短链接访问目标内容。',
  notFoundTitle: '链接不存在或已失效',
  notFoundMessage: '请检查链接是否完整，或联系链接提供方确认当前状态。',
  goneTitle: '链接已过期',
  goneMessage: '该链接已超过有效期，无法继续访问。',
  footer: 'GravityLink',
  // 通知与检测
  notifyWebhookUrl: '',
  notifyHttpUrl: '',
  domainCheckEnabled: false,
});
const notifySaving = ref(false);

const authModeLabel = computed(() => {
  const source = auth.user?.auth_source;
  if (source === 'local') return '本地账号';
  if (source === 'development') return '开发模式';
  return 'Logto OIDC';
});

const canReset = computed(
  () => resetConfirmation.value === 'RESET GRAVITYLINK' && (auth.user?.auth_source !== 'local' || resetPassword.value.length > 0),
);

onMounted(async () => {
  try {
    const data = await getSystemConfigs();
    authInfo.value = data.auth;
    form.siteName = data.configs['public.site_name'] || form.siteName;
    form.publicBaseURL = data.configs['public.base_url'] || '';
    form.homeTitle = data.configs['public.home.title'] || form.homeTitle;
    form.homeMessage = data.configs['public.home.message'] || form.homeMessage;
    form.notFoundTitle = data.configs['public.not_found.title'] || form.notFoundTitle;
    form.notFoundMessage = data.configs['public.not_found.message'] || form.notFoundMessage;
    form.goneTitle = data.configs['public.gone.title'] || form.goneTitle;
    form.goneMessage = data.configs['public.gone.message'] || form.goneMessage;
    form.footer = data.configs['public.footer'] || form.footer;
    form.notifyWebhookUrl = data.configs['notify_webhook_url'] || '';
    form.notifyHttpUrl = data.configs['notify_http_url'] || '';
    form.domainCheckEnabled = data.configs['domain_check_enabled'] === '1';
  } catch (err) {
    message.error(err instanceof Error ? err.message : '加载配置失败');
  } finally {
    loading.value = false;
  }
});

async function save() {
  saving.value = true;
  try {
    await updateSystemConfigs({
      'public.site_name': form.siteName,
      'public.base_url': form.publicBaseURL.trim(),
      'public.home.title': form.homeTitle,
      'public.home.message': form.homeMessage,
      'public.not_found.title': form.notFoundTitle,
      'public.not_found.message': form.notFoundMessage,
      'public.gone.title': form.goneTitle,
      'public.gone.message': form.goneMessage,
      'public.footer': form.footer,
    });
    message.success('公开页面提示已更新');
  } catch (err) {
    message.error(err instanceof Error ? err.message : '保存失败');
  } finally {
    saving.value = false;
  }
}

async function saveNotify() {
  notifySaving.value = true;
  try {
    await updateSystemConfigs({
      notify_webhook_url: form.notifyWebhookUrl.trim(),
      notify_http_url: form.notifyHttpUrl.trim(),
      domain_check_enabled: form.domainCheckEnabled ? '1' : '0',
    });
    message.success('通知与检测配置已更新，立即生效');
  } catch (err) {
    message.error(err instanceof Error ? err.message : '保存失败');
  } finally {
    notifySaving.value = false;
  }
}

async function executeReset() {
  resetBusy.value = true;
  try {
    await resetSystem(resetConfirmation.value, resetPassword.value);
    clearSession();
    auth.reset();
    message.success('已恢复到未初始化状态');
    await router.replace({ name: 'setup' });
  } catch (err) {
    message.error(err instanceof Error ? err.message : '重置失败');
    resetBusy.value = false;
    showResetDialog.value = false;
  }
}
</script>

<template>
  <div class="settings-page">
    <NSkeleton v-if="loading" :repeat="4" height="120px" :sharp="false" style="display:grid;gap:16px" />

    <template v-else>
      <NCard class="settings-shell">
        <NTabs type="line" animated pane-style="padding-top: var(--space-5)">
          <NTabPane name="public" tab="网站配置">
      <section class="settings-section">
        <div class="section-head">
          <div>
            <strong>公开访问提示</strong>
            <p class="muted">首页、未知短码和过期链接都会使用这些文案</p>
          </div>
          <div class="card-actions">
            <NButton v-if="form.publicBaseURL" text tag="a" :href="form.publicBaseURL" target="_blank">
              <template #icon><ExternalLink :size="14" /></template>
              预览公开入口
            </NButton>
            <NButton type="primary" :loading="saving" @click="save">
              <template #icon><Save :size="16" /></template>
              保存更改
            </NButton>
          </div>
        </div>

        <NForm label-placement="top">
          <div class="form-grid">
            <NFormItem label="站点名称"><NInput v-model:value="form.siteName" /></NFormItem>
            <NFormItem label="页脚"><NInput v-model:value="form.footer" /></NFormItem>
            <NFormItem label="公开访问地址" class="span-2">
              <NInput v-model:value="form.publicBaseURL" placeholder="如 https://s.example.com 或 http://localhost:18080" />
            </NFormItem>
            <NFormItem label="首页标题" class="span-2"><NInput v-model:value="form.homeTitle" /></NFormItem>
            <NFormItem label="首页说明" class="span-2"><NInput v-model:value="form.homeMessage" type="textarea" :rows="3" /></NFormItem>
            <NFormItem label="链接不存在标题" class="span-2"><NInput v-model:value="form.notFoundTitle" /></NFormItem>
            <NFormItem label="链接不存在说明" class="span-2"><NInput v-model:value="form.notFoundMessage" type="textarea" :rows="3" /></NFormItem>
            <NFormItem label="链接过期标题" class="span-2"><NInput v-model:value="form.goneTitle" /></NFormItem>
            <NFormItem label="链接过期说明" class="span-2"><NInput v-model:value="form.goneMessage" type="textarea" :rows="3" /></NFormItem>
          </div>
        </NForm>
      </section>
          </NTabPane>

          <NTabPane name="auth" tab="认证配置">
      <section class="settings-section">
        <div class="section-head">
          <div><strong>身份认证</strong><p class="muted">认证提供方本身只能通过重新初始化修改</p></div>
        </div>
        <NDescriptions :column="2" label-placement="left" bordered>
          <NDescriptionsItem label="模式">{{ authModeLabel }}</NDescriptionsItem>
          <NDescriptionsItem label="当前账号">{{ auth.user?.username }} · {{ auth.user?.role }}</NDescriptionsItem>
          <NDescriptionsItem label="Issuer">{{ authInfo?.issuer || '不适用' }}</NDescriptionsItem>
          <NDescriptionsItem label="Audience">{{ authInfo?.audience || '不适用' }}</NDescriptionsItem>
        </NDescriptions>
      </section>
          </NTabPane>

          <NTabPane name="notify" tab="通知与检测">
      <section class="settings-section">
        <div class="section-head">
          <div><strong>通知渠道</strong><p class="muted">活码二维码耗尽、域名被封等事件会自动推送；两个渠道都留空则不通知</p></div>
          <NButton type="primary" :loading="notifySaving" @click="saveNotify">
            <template #icon><Save :size="16" /></template>
            保存配置
          </NButton>
        </div>
        <NForm label-placement="top">
          <NFormItem label="企业微信机器人 Webhook">
            <NInput v-model:value="form.notifyWebhookUrl" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx" />
          </NFormItem>
          <NFormItem label="自定义通知接口">
            <NInput v-model:value="form.notifyHttpUrl" placeholder="POST JSON {title, content, time}；兼容 Bark/Server酱 等自建转发" />
          </NFormItem>
          <NFormItem label="域名封禁检测">
            <div class="check-row">
              <NSwitch v-model:value="form.domainCheckEnabled" />
              <span class="muted">每小时借微信官方桥接接口检测全部活跃域名，发现被微信封禁立即通知（默认关闭）</span>
            </div>
          </NFormItem>
        </NForm>
      </section>
          </NTabPane>

          <NTabPane v-if="auth.isSuperAdmin" name="danger" tab="危险区">
      <section class="settings-section danger-card">
          <div class="danger-head">
            <AlertTriangle :size="18" />
            <strong>危险区</strong>
          </div>
        <div class="danger-body">
          <div>
            <strong>恢复到未初始化状态</strong>
            <p class="muted">保留短链、域名、落地页和访问统计；清除账号授权、会话、公开页配置与运行配置文件。</p>
          </div>
          <NButton type="error" @click="showResetDialog = true">
            <template #icon><RotateCcw :size="16" /></template>
            清除所有配置
          </NButton>
        </div>
      </section>
          </NTabPane>
        </NTabs>
      </NCard>
    </template>

    <NModal v-model:show="showResetDialog" preset="card" title="确认清除所有配置" style="width: 520px" :mask-closable="false">
      <NAlert type="warning" :show-icon="true" style="margin-bottom: var(--space-4)">
        业务数据会保留，但所有用户授权和登录会话将失效。此操作会让管理端立即退出并进入首次配置。
      </NAlert>
      <NForm label-placement="top">
        <NFormItem label="输入 RESET GRAVITYLINK 确认">
          <NInput v-model:value="resetConfirmation" autocomplete="off" placeholder="RESET GRAVITYLINK" />
        </NFormItem>
        <NFormItem v-if="auth.user?.auth_source === 'local'" label="再次输入当前密码">
          <NInput v-model:value="resetPassword" type="password" show-password-on="click" autocomplete="current-password" />
        </NFormItem>
      </NForm>

      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: var(--space-2)">
          <NButton @click="showResetDialog = false">取消</NButton>
          <NButton type="error" :loading="resetBusy" :disabled="!canReset" @click="executeReset">
            <template #icon><RotateCcw :size="16" /></template>
            确认清除
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.settings-page {
  display: grid;
  gap: var(--space-5);
}

.settings-shell {
  background: rgba(255, 255, 255, 0.9);
}

.settings-shell :deep(.n-tabs-nav) {
  padding: 0 var(--space-2);
}

.settings-section {
  padding: var(--space-2) var(--space-2) var(--space-3);
}

.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
  margin-bottom: var(--space-5);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.check-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
}

.section-head strong { font-size: var(--font-size-lg); }
.section-head p { margin-top: var(--space-1); }

.card-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 var(--space-4);
}

.span-2 {
  grid-column: span 2;
}

.danger-card {
  padding: var(--space-5);
  border: 1px solid rgba(179, 38, 30, .24);
  border-radius: var(--radius-lg);
  background: var(--color-error-bg);
}

.danger-head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--color-error);
}

.danger-body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-5);
}

.danger-body p {
  margin-top: var(--space-1);
  font-size: var(--font-size-md);
}

@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
  .span-2 {
    grid-column: span 1;
  }
  .danger-body {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
