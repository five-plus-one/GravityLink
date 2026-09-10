<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Check, ChevronLeft, ChevronRight, Clipboard, Database, KeyRound, ShieldCheck } from '@lucide/vue';
import {
  NAlert,
  NButton,
  NCard,
  NCollapse,
  NCollapseItem,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NRadioButton,
  NRadioGroup,
  NSteps,
  NStep,
  useDialog,
  useMessage,
} from 'naive-ui';
import { login, recallSetupOIDC, rememberSetupOIDC } from '../auth';
import {
  checkSetupLogto,
  completeSetup,
  configureLocalOwner,
  setupAuthConfig,
  testSetupDatabase,
  type SetupPayload,
} from '../setup';
import { useSetupStore } from '../stores/setup';

const route = useRoute();
const router = useRouter();
const setup = useSetupStore();
const dialog = useDialog();
const message = useMessage();

const step = ref(1);
const busy = ref(false);
const error = ref('');
const copied = ref('');
const authVerified = ref(false);
const ownerName = ref('');
const dbTested = ref(false);

const status = computed(() => setup.status);
const form = reactive({
  authMode: 'logto' as 'logto' | 'local',
  issuer: '',
  appID: '',
  audience: '',
  jwksURL: '',
  scopes: 'openid profile email',
  adminBaseURL: window.location.origin,
  publicBaseURL: '',
  allowedRoles: 'admin',
  username: '',
  password: '',
  passwordConfirm: '',
  mysqlHost: 'mysql',
  mysqlPort: '3306',
  mysqlDatabase: 'gravitylink',
  mysqlUser: 'gravitylink',
  mysqlPassword: '',
  mysqlParams: 'charset=utf8mb4&parseTime=True&loc=Local',
  mysqlDSN: '',
  redisHost: 'redis',
  redisPort: '6379',
  redisAddr: '',
  redisPassword: '',
  redisDB: 0,
});

const localReady = computed(
  () => form.username.trim().length >= 3 && form.password.length >= 10 && form.password === form.passwordConfirm,
);
const databaseReady = computed(
  () => Boolean(form.mysqlDSN.trim()) || Boolean(form.mysqlHost && form.mysqlPort && form.mysqlDatabase && form.mysqlUser),
);
const adminOrigin = computed(() => form.adminBaseURL.trim().replace(/\/+$/, ''));
const setupRedirectURI = computed(() => `${adminOrigin.value}/setup/auth/callback`);
const loginRedirectURI = computed(() => `${adminOrigin.value}/auth/callback`);
const postLogoutRedirectURI = computed(() => `${adminOrigin.value}/`);
const suggestedAudience = computed(() => `${adminOrigin.value}/api`);

onMounted(async () => {
  if (!status.value) {
    try {
      await setup.refresh();
    } catch (err) {
      error.value = messageOf(err);
      return;
    }
  }
  hydrateFromStatus();

  // OIDC 回调成功返回
  if (route.query.verified === '1' && status.value?.auth.verified) {
    authVerified.value = true;
    ownerName.value = status.value.auth.owner_username;
    step.value = 3;
    message.success(`已验证管理员身份：${ownerName.value}`);
  }
});

function hydrateFromStatus() {
  const s = status.value;
  if (!s) return;
  form.authMode = s.auth.mode || 'logto';
  form.issuer = s.auth.issuer;
  form.appID = s.auth.app_id;
  form.audience = s.auth.audience;
  form.jwksURL = s.auth.jwks_url;
  form.scopes = s.auth.scopes || 'openid profile email';
  form.adminBaseURL = s.auth.admin_base_url || window.location.origin;
  form.allowedRoles = s.auth.allowed_roles?.join(', ') || 'admin';
  form.username = s.auth.owner_username;
  form.mysqlHost = s.database.host || 'mysql';
  form.mysqlPort = s.database.port || '3306';
  form.mysqlDatabase = s.database.database || 'gravitylink';
  form.mysqlUser = s.database.user || 'gravitylink';
  form.mysqlParams = s.database.params || 'charset=utf8mb4&parseTime=True&loc=Local';
  form.redisHost = s.redis.host || 'redis';
  form.redisPort = s.redis.port || '6379';
  form.redisDB = s.redis.db || 0;
  authVerified.value = s.auth.verified;
  ownerName.value = s.auth.owner_username;
  // 后端 draft 已保存数据库配置时（Logto 回调后页面重载），标记为已测试
  if (s.database.dsn_configured) {
    dbTested.value = true;
  }
}

function payload(): SetupPayload {
  return {
    database: {
      host: form.mysqlHost.trim(),
      port: form.mysqlPort.trim(),
      database: form.mysqlDatabase.trim(),
      user: form.mysqlUser.trim(),
      password: form.mysqlPassword,
      params: form.mysqlParams.trim(),
      dsn: form.mysqlDSN.trim(),
    },
    redis: {
      host: form.redisHost.trim(),
      port: form.redisPort.trim(),
      addr: form.redisAddr.trim(),
      password: form.redisPassword,
      db: Number(form.redisDB) || 0,
    },
    auth: {
      mode: form.authMode,
      issuer: form.issuer.trim(),
      app_id: form.appID.trim(),
      audience: form.audience.trim(),
      jwks_url: form.jwksURL.trim(),
      scopes: form.scopes.trim(),
      admin_base_url: form.adminBaseURL.trim(),
      allowed_roles: form.allowedRoles.split(',').map((r) => r.trim()).filter(Boolean),
      username: form.username.trim(),
      password: form.password,
    },
    public_base_url: form.publicBaseURL.trim(),
  };
}

async function verifyLogto() {
  busy.value = true;
  error.value = '';
  try {
    const current = payload();
    const result = await checkSetupLogto(current);
    const config = setupAuthConfig(current, result);
    rememberSetupOIDC(config);
    await login(config);
  } catch (err) {
    error.value = messageOf(err);
    busy.value = false;
  }
}

async function verifyLocal() {
  if (!localReady.value) {
    error.value = form.password !== form.passwordConfirm ? '两次输入的密码不一致' : '用户名至少 3 位，密码至少 10 位';
    return;
  }
  busy.value = true;
  error.value = '';
  try {
    const result = await configureLocalOwner(payload());
    authVerified.value = result.verified;
    ownerName.value = result.username;
    message.success(`本地超级管理员 ${result.username} 已就绪`);
    step.value = 3;
  } catch (err) {
    error.value = messageOf(err);
  } finally {
    busy.value = false;
  }
}

async function testConnections() {
  busy.value = true;
  error.value = '';
  try {
    const result = await testSetupDatabase(payload());
    dbTested.value = true;
    message.success(
      result.schema_ready ? 'MySQL 与 Redis 连接正常，现有数据库结构可用' : '连接正常，完成初始化时将自动创建数据库结构',
    );
  } catch (err) {
    error.value = messageOf(err);
  } finally {
    busy.value = false;
  }
}

async function finish() {
  busy.value = true;
  error.value = '';
  try {
    await completeSetup(payload());
    setup.invalidate();
    message.success('初始化完成，正在进入管理端…');
    await router.replace('/');
  } catch (err) {
    error.value = messageOf(err);
  } finally {
    busy.value = false;
  }
}

function chooseMode(mode: 'logto' | 'local') {
  if (mode === form.authMode) return;
  if (!authVerified.value) {
    form.authMode = mode;
    return;
  }
  dialog.warning({
    title: '切换认证方式',
    content: '切换认证方式将清除已验证的管理员身份，需要重新验证。确定切换吗？',
    positiveText: '切换',
    negativeText: '取消',
    onPositiveClick: () => {
      form.authMode = mode;
      authVerified.value = false;
    },
  });
}

function editDatabase() {
  dbTested.value = false;
  step.value = 1;
}

function editAuth() {
  authVerified.value = false;
  step.value = 2;
}

function useCurrentOrigin() {
  form.adminBaseURL = window.location.origin;
}

async function copyValue(label: string, value: string) {
  if (!value) return;
  try {
    await navigator.clipboard.writeText(value);
    copied.value = label;
    window.setTimeout(() => {
      if (copied.value === label) copied.value = '';
    }, 1600);
  } catch {
    message.error('浏览器未允许读取剪贴板，请手动复制该值');
  }
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : '操作失败，请检查配置后重试';
}
</script>

<template>
  <main class="setup-shell">
    <aside class="setup-aside">
      <div class="setup-brand">
        <span class="brand-mark">G</span>
        <div>
          <strong>GravityLink</strong>
          <small>系统初始化</small>
        </div>
      </div>

      <NSteps :current="step" vertical class="setup-steps">
        <NStep title="数据服务" description="MySQL 与 Redis">
          <template #icon><Database :size="16" /></template>
        </NStep>
        <NStep title="管理员身份" description="Logto 或本地账号">
          <template #icon><ShieldCheck :size="16" /></template>
        </NStep>
        <NStep title="确认启用" description="创建结构并启动服务">
          <template #icon><Check :size="16" /></template>
        </NStep>
      </NSteps>

      <div class="setup-environment">
        <span>运行环境</span>
        <strong>{{ status?.environment || '—' }}</strong>
      </div>
    </aside>

    <section class="setup-main">
      <header class="setup-header">
        <div>
          <p class="eyebrow">首次配置</p>
          <h1>{{ step === 1 ? '连接数据服务' : step === 2 ? '建立管理员身份' : '确认并启用' }}</h1>
          <p class="muted">
            {{ step === 1 ? 'GravityLink 依赖 MySQL 与 Redis，先确认它们可以连通。' : step === 2 ? '首位完成验证的账号将成为超级管理员。' : '检查配置摘要，完成后立即进入管理端。' }}
          </p>
        </div>
        <span class="muted">步骤 {{ step }} / 3</span>
      </header>

      <NAlert v-if="status?.reason && step === 1" type="warning" :show-icon="true" style="margin-bottom: var(--space-4)">
        <strong>系统正在等待初始化</strong>
        <div>{{ status.reason }}</div>
      </NAlert>

      <!-- 步骤 1：数据服务 -->
      <div v-if="step === 1" class="setup-step">
        <NCard title="MySQL" size="small" class="setup-card">
          <template #header-extra><span class="muted">保存短链、域名、落地页、用户与统计数据</span></template>
          <NForm label-placement="top">
            <div class="form-grid">
              <NFormItem label="Host" class="span-2"><NInput v-model:value="form.mysqlHost" /></NFormItem>
              <NFormItem label="Port"><NInput v-model:value="form.mysqlPort" /></NFormItem>
              <NFormItem label="Database"><NInput v-model:value="form.mysqlDatabase" /></NFormItem>
              <NFormItem label="User"><NInput v-model:value="form.mysqlUser" autocomplete="username" /></NFormItem>
              <NFormItem label="Password">
                <NInput v-model:value="form.mysqlPassword" type="password" show-password-on="click" autocomplete="new-password" />
              </NFormItem>
              <NFormItem label="连接参数" class="span-4"><NInput v-model:value="form.mysqlParams" /></NFormItem>
            </div>
            <NCollapse class="span-4">
              <NCollapseItem title="使用完整 DSN" name="dsn">
                <NFormItem label="MYSQL_DSN"><NInput v-model:value="form.mysqlDSN" /></NFormItem>
              </NCollapseItem>
            </NCollapse>
          </NForm>
        </NCard>

        <NCard title="Redis" size="small" class="setup-card">
          <template #header-extra><span class="muted">用于访问缓存、计数和异步队列</span></template>
          <NForm label-placement="top">
            <div class="form-grid">
              <NFormItem label="Host" class="span-2"><NInput v-model:value="form.redisHost" /></NFormItem>
              <NFormItem label="Port"><NInput v-model:value="form.redisPort" /></NFormItem>
              <NFormItem label="DB"><NInputNumber v-model:value="form.redisDB" :min="0" style="width: 100%" /></NFormItem>
              <NFormItem label="Password" class="span-2">
                <NInput v-model:value="form.redisPassword" type="password" show-password-on="click" autocomplete="new-password" />
              </NFormItem>
              <NFormItem label="完整地址（可选）" class="span-2"><NInput v-model:value="form.redisAddr" placeholder="redis:6379" /></NFormItem>
            </div>
          </NForm>
        </NCard>

        <NAlert type="info" :show-icon="true">必须先点击「测试连接」确认可达，才能进入下一步。测试通过后连接配置会临时保存，Logto 验证返回后无需重新填写。</NAlert>
      </div>

      <!-- 步骤 2：管理员身份 -->
      <div v-else-if="step === 2" class="setup-step">
        <NRadioGroup :value="form.authMode" @update:value="chooseMode">
          <NRadioButton value="logto">
            <span style="display: inline-flex; align-items: center; gap: 6px"><ShieldCheck :size="14" />Logto</span>
          </NRadioButton>
          <NRadioButton value="local">
            <span style="display: inline-flex; align-items: center; gap: 6px"><KeyRound :size="14" />本地账号</span>
          </NRadioButton>
        </NRadioGroup>

        <NCard v-if="form.authMode === 'logto'" title="Logto OIDC" size="small" class="setup-card">
          <template #header-extra><span class="muted">先检查 Discovery 配置，再跳转到 Logto 完成身份验证</span></template>
          <NForm label-placement="top">
            <div class="form-grid">
              <NFormItem label="Issuer" class="span-2"><NInput v-model:value="form.issuer" placeholder="https://logto.example.com/oidc" /></NFormItem>
              <NFormItem label="应用 ID" class="span-2"><NInput v-model:value="form.appID" autocomplete="off" /></NFormItem>
              <NFormItem class="span-2">
                <template #label>
                  <div style="display: flex; justify-content: space-between; align-items: center; width: 100%">
                    <span>API Audience</span>
                    <NButton text type="primary" size="tiny" @click="form.audience = suggestedAudience">使用建议值</NButton>
                  </div>
                </template>
                <NInput v-model:value="form.audience" placeholder="https://admin.example.com/api" />
                <template #feedback>必须与 Logto "API 资源" 的 API 标识符完全一致。</template>
              </NFormItem>
              <NFormItem class="span-2">
                <template #label>
                  <div style="display: flex; justify-content: space-between; align-items: center; width: 100%">
                    <span>管理端公开 URL</span>
                    <NButton text type="primary" size="tiny" @click="useCurrentOrigin">使用当前地址</NButton>
                  </div>
                </template>
                <NInput v-model:value="form.adminBaseURL" placeholder="https://admin.example.com" />
                <template #feedback>填写用户实际访问管理端的协议和域名，不包含路径。</template>
              </NFormItem>
              <NFormItem label="Scopes" class="span-2"><NInput v-model:value="form.scopes" /></NFormItem>
              <NFormItem label="后续可授权角色" class="span-2"><NInput v-model:value="form.allowedRoles" placeholder="admin" /></NFormItem>
            </div>
            <NCollapse>
              <NCollapseItem title="高级配置" name="advanced">
                <NFormItem label="JWKS URL"><NInput v-model:value="form.jwksURL" placeholder="留空后从 Discovery 获取" /></NFormItem>
              </NCollapseItem>
            </NCollapse>
          </NForm>

          <NCard title="同步填写 Logto 应用设置" size="small" embedded class="logto-guide">
            <div class="logto-grid">
              <div>
                <strong>重定向 URIs</strong>
                <div class="copy-row">
                  <code>{{ setupRedirectURI }}</code>
                  <NButton text size="tiny" @click="copyValue('setup', setupRedirectURI)">
                    <Check v-if="copied === 'setup'" :size="14" /><Clipboard v-else :size="14" />
                  </NButton>
                </div>
                <div class="copy-row">
                  <code>{{ loginRedirectURI }}</code>
                  <NButton text size="tiny" @click="copyValue('login', loginRedirectURI)">
                    <Check v-if="copied === 'login'" :size="14" /><Clipboard v-else :size="14" />
                  </NButton>
                </div>
              </div>
              <div>
                <strong>退出登录后重定向 URI</strong>
                <div class="copy-row">
                  <code>{{ postLogoutRedirectURI }}</code>
                  <NButton text size="tiny" @click="copyValue('logout', postLogoutRedirectURI)">
                    <Check v-if="copied === 'logout'" :size="14" /><Clipboard v-else :size="14" />
                  </NButton>
                </div>
              </div>
              <div>
                <strong>CORS 允许的来源</strong>
                <div class="copy-row">
                  <code>{{ adminOrigin }}</code>
                  <NButton text size="tiny" @click="copyValue('cors', adminOrigin)">
                    <Check v-if="copied === 'cors'" :size="14" /><Clipboard v-else :size="14" />
                  </NButton>
                </div>
              </div>
              <div>
                <strong>API 资源</strong>
                <div class="copy-row">
                  <code>{{ form.audience || '请先填写上方 API Audience' }}</code>
                  <NButton text size="tiny" :disabled="!form.audience" @click="copyValue('audience', form.audience)">
                    <Check v-if="copied === 'audience'" :size="14" /><Clipboard v-else :size="14" />
                  </NButton>
                </div>
                <small class="muted">在 Logto "API 资源" 中新建 GravityLink API，并将 API 标识符设置为此值。</small>
              </div>
            </div>
          </NCard>

          <NButton type="primary" :loading="busy" @click="verifyLogto">
            <template #icon><ShieldCheck :size="16" /></template>
            检查配置并使用 Logto 验证
          </NButton>
        </NCard>

        <NCard v-else title="本地超级管理员" size="small" class="setup-card">
          <template #header-extra><span class="muted">适合内网或不接入统一身份平台的部署</span></template>
          <NForm label-placement="top">
            <div class="form-grid">
              <NFormItem label="用户名" class="span-2"><NInput v-model:value="form.username" autocomplete="username" placeholder="admin" /></NFormItem>
              <NFormItem label="密码" class="span-2">
                <NInput v-model:value="form.password" type="password" show-password-on="click" autocomplete="new-password" />
              </NFormItem>
              <NFormItem label="确认密码" class="span-2">
                <NInput v-model:value="form.passwordConfirm" type="password" show-password-on="click" autocomplete="new-password" />
              </NFormItem>
            </div>
          </NForm>
          <NAlert type="info" :show-icon="true" style="margin-bottom: var(--space-4)">密码至少 10 位。系统仅保存 bcrypt 哈希。</NAlert>
          <NButton type="primary" :loading="busy" :disabled="!localReady" @click="verifyLocal">
            <template #icon><KeyRound :size="16" /></template>
            创建管理员身份
          </NButton>
        </NCard>
      </div>

      <!-- 步骤 3：确认启用 -->
      <div v-else class="setup-step">
        <NCard size="small" class="setup-card">
          <div class="review-row">
            <span>MySQL</span>
            <strong>{{ form.mysqlHost }}:{{ form.mysqlPort }} / {{ form.mysqlDatabase }}</strong>
            <NButton text type="primary" size="small" @click="editDatabase">修改</NButton>
          </div>
          <div class="review-row">
            <span>Redis</span>
            <strong>{{ form.redisAddr || `${form.redisHost}:${form.redisPort}` }} / DB {{ form.redisDB }}</strong>
            <NButton text type="primary" size="small" @click="editDatabase">修改</NButton>
          </div>
          <div class="review-row">
            <span>超级管理员</span>
            <strong>{{ ownerName }} · {{ form.authMode === 'logto' ? 'Logto' : '本地账号' }}</strong>
            <NButton text type="primary" size="small" @click="editAuth">修改</NButton>
          </div>
        </NCard>
        <NCard size="small" class="setup-card" style="margin-top: var(--space-4)">
          <NForm label-placement="top">
            <NFormItem label="公开访问地址">
              <NInput v-model:value="form.publicBaseURL" placeholder="如 https://s.example.com 或 http://localhost:18080" />
            </NFormItem>
          </NForm>
          <p class="muted" style="font-size: 12px; margin: 0">短链接对外访问的入口地址。用于侧栏「查看访问地址」和设置页「预览公开入口」。留空则不显示这两个入口。</p>
        </NCard>
        <NAlert type="success" :show-icon="true" style="margin-top: var(--space-4)">
          完成后会自动创建或升级数据库结构，并锁定初始化接口。业务数据不会因重新配置而被清空。
        </NAlert>
      </div>

      <NAlert v-if="error" type="error" :show-icon="true" style="margin-top: var(--space-4)">{{ error }}</NAlert>

      <footer class="setup-actions">
        <NButton v-if="step > 1" :disabled="busy" @click="step -= 1">
          <template #icon><ChevronLeft :size="16" /></template>
          上一步
        </NButton>
        <span v-else></span>
        <div style="display: flex; gap: var(--space-2)">
          <template v-if="step === 1">
            <NButton :disabled="busy || !databaseReady" :loading="busy" @click="testConnections">
              {{ dbTested ? '重新测试连接' : '测试连接' }}
            </NButton>
            <NButton type="primary" :disabled="busy || !databaseReady || !dbTested" @click="step = 2">
              {{ dbTested ? '继续' : '请先测试连接' }}
              <template #icon><ChevronRight :size="16" /></template>
            </NButton>
          </template>
          <NButton v-else-if="step === 2" type="primary" :disabled="busy || !authVerified" @click="step = 3">
            {{ authVerified ? '继续' : '请先完成管理员验证' }}
            <template #icon><ChevronRight :size="16" /></template>
          </NButton>
          <NButton v-else type="primary" :loading="busy" :disabled="!authVerified" @click="finish">
            <template #icon><Check :size="16" /></template>
            {{ busy ? '正在启用…' : '完成初始化' }}
          </NButton>
        </div>
      </footer>
    </section>
  </main>
</template>

<style scoped>
.setup-shell {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  background: var(--color-bg-page);
}

.setup-aside {
  display: flex;
  flex-direction: column;
  padding: var(--space-8) var(--space-6);
  background: var(--color-bg-sidebar);
  color: var(--color-text-inverse);
}

.setup-brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-10);
}

.setup-brand strong,
.setup-brand small {
  display: block;
}

.setup-brand small {
  margin-top: var(--space-1);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-sm);
}

.setup-steps {
  flex: 1;
}

.setup-steps :deep(.n-step-content__description) {
  color: rgba(255, 255, 255, 0.45);
}

.setup-steps :deep(.n-step) {
  --n-indicator-text-color: rgba(255, 255, 255, 0.65);
  --n-indicator-border-color: rgba(255, 255, 255, 0.25);
  --n-title-text-color: rgba(255, 255, 255, 0.85);
}

.setup-environment {
  margin-top: auto;
  padding-top: var(--space-4);
  display: flex;
  justify-content: space-between;
  border-top: 1px solid rgba(255, 255, 255, 0.12);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-sm);
}

.setup-environment strong {
  color: var(--color-text-inverse);
}

.setup-main {
  width: min(920px, calc(100vw - 320px));
  margin: 0 auto;
  padding: var(--space-10) 0 var(--space-10);
}

.setup-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--space-5);
  margin-bottom: var(--space-6);
}

.setup-header h1 {
  font-size: var(--font-size-2xl);
  font-weight: 600;
  margin-bottom: var(--space-2);
}

.eyebrow {
  margin-bottom: var(--space-2);
  color: var(--color-primary);
  font-size: var(--font-size-sm);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.setup-step {
  display: grid;
  gap: var(--space-4);
}

.setup-card :deep(.n-card-header) {
  font-weight: 600;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0 var(--space-4);
}

.span-2 {
  grid-column: span 2;
}
.span-4 {
  grid-column: span 4;
}

.logto-guide {
  margin: var(--space-4) 0;
}

.logto-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.logto-grid strong {
  display: block;
  margin-bottom: var(--space-2);
  font-size: var(--font-size-md);
  color: var(--color-text-secondary);
}

.copy-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 24px;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-2);
}

.copy-row code {
  padding: var(--space-1) var(--space-2);
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.review-row {
  display: grid;
  grid-template-columns: 140px minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--color-border);
}

.review-row:last-child {
  border-bottom: 0;
}

.review-row span {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-md);
}

.setup-actions {
  margin-top: var(--space-6);
  padding-top: var(--space-5);
  display: flex;
  justify-content: space-between;
  gap: var(--space-3);
  border-top: 1px solid var(--color-border);
}

@media (max-width: 980px) {
  .setup-shell {
    grid-template-columns: 1fr;
  }
  .setup-aside {
    padding: var(--space-5);
  }
  .setup-main {
    width: auto;
    padding: var(--space-6) var(--space-5);
  }
  .form-grid {
    grid-template-columns: 1fr 1fr;
  }
  .span-4 {
    grid-column: span 2;
  }
  .logto-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
  .span-2, .span-4 {
    grid-column: span 1;
  }
}
</style>
