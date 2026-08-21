<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Check, ChevronLeft, ChevronRight, Clipboard, Database, KeyRound, LoaderCircle, ShieldCheck } from '@lucide/vue';
import { handleCallback, login, recallSetupOIDC, rememberSetupOIDC } from './auth';
import {
  checkSetupLogto,
  claimSetupLogto,
  completeSetup,
  configureLocalOwner,
  setupAuthConfig,
  testSetupDatabase,
  type SetupPayload,
  type SetupStatus,
} from './setup';

const props = defineProps<{ status: SetupStatus }>();
const emit = defineEmits<{ completed: [] }>();

const step = ref(1);
const busy = ref(false);
const error = ref('');
const success = ref('');
const copied = ref('');
const authVerified = ref(props.status.auth.verified);
const ownerName = ref(props.status.auth.owner_username);
const dbTested = ref(false);

const form = reactive({
  authMode: (props.status.auth.mode || 'logto') as 'logto' | 'local',
  issuer: props.status.auth.issuer,
  appID: props.status.auth.app_id,
  audience: props.status.auth.audience,
  jwksURL: props.status.auth.jwks_url,
  scopes: props.status.auth.scopes || 'openid profile email',
  adminBaseURL: props.status.auth.admin_base_url || window.location.origin,
  allowedRoles: props.status.auth.allowed_roles?.join(', ') || 'admin',
  username: props.status.auth.owner_username,
  password: '',
  passwordConfirm: '',
  mysqlHost: props.status.database.host || 'mysql',
  mysqlPort: props.status.database.port || '3306',
  mysqlDatabase: props.status.database.database || 'gravitylink',
  mysqlUser: props.status.database.user || 'gravitylink',
  mysqlPassword: '',
  mysqlParams: props.status.database.params || 'charset=utf8mb4&parseTime=True&loc=Local',
  mysqlDSN: '',
  redisHost: props.status.redis.host || 'redis',
  redisPort: props.status.redis.port || '6379',
  redisAddr: '',
  redisPassword: '',
  redisDB: props.status.redis.db || 0,
});

const localReady = computed(
  () =>
    form.username.trim().length >= 3 &&
    form.password.length >= 10 &&
    form.password === form.passwordConfirm,
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
  if (window.location.pathname !== '/setup/auth/callback') return;
  const config = recallSetupOIDC();
  if (!config) {
    error.value = '登录会话已失效，请重新检查 Logto 配置';
    window.history.replaceState({}, '', '/');
    return;
  }
  busy.value = true;
  try {
    await handleCallback(config, '/setup/auth/callback', '/');
    const result = await claimSetupLogto();
    authVerified.value = result.verified;
    ownerName.value = result.username;
    step.value = 3;
    success.value = `已验证管理员身份：${result.username}`;
  } catch (err) {
    error.value = messageOf(err);
  } finally {
    busy.value = false;
  }
});

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
      disabled: false,
      issuer: form.issuer.trim(),
      app_id: form.appID.trim(),
      audience: form.audience.trim(),
      jwks_url: form.jwksURL.trim(),
      scopes: form.scopes.trim(),
      admin_base_url: form.adminBaseURL.trim(),
      allowed_roles: form.allowedRoles.split(',').map((role) => role.trim()).filter(Boolean),
      username: form.username.trim(),
      password: form.password,
    },
  };
}

async function verifyLogto() {
  busy.value = true;
  error.value = '';
  success.value = '';
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
    success.value = `本地超级管理员 ${result.username} 已就绪`;
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
  success.value = '';
  try {
    const result = await testSetupDatabase(payload());
    dbTested.value = true;
    success.value = result.schema_ready
      ? 'MySQL 与 Redis 连接正常，现有数据库结构可用'
      : '连接正常，完成初始化时将自动创建数据库结构';
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
    emit('completed');
  } catch (err) {
    error.value = messageOf(err);
  } finally {
    busy.value = false;
  }
}

function chooseMode(mode: 'logto' | 'local') {
  if (mode === form.authMode) return;
  if (authVerified.value && !window.confirm('切换认证方式将清除已验证的管理员身份，需要重新验证。确定切换吗？')) {
    return;
  }
  form.authMode = mode;
  authVerified.value = false;
  error.value = '';
  success.value = '';
}

// 从确认页返回修改某一步时，清除对应步骤的已完成标记
function editDatabase() {
  dbTested.value = false;
  success.value = '';
  step.value = 1;
}

function editAuth() {
  authVerified.value = false;
  success.value = '';
  step.value = 2;
}

function useCurrentOrigin() {
  form.adminBaseURL = window.location.origin;
}

function useSuggestedAudience() {
  form.audience = suggestedAudience.value;
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
    error.value = '浏览器未允许读取剪贴板，请手动复制该值';
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
        <div><strong>GravityLink</strong><small>系统初始化</small></div>
      </div>
      <ol class="setup-steps">
        <li :class="{ active: step === 1, done: step > 1 }">
          <span><Database :size="16" /></span>
          <div><strong>数据服务</strong><small>MySQL 与 Redis</small></div>
        </li>
        <li :class="{ active: step === 2, done: step > 2 }">
          <span><ShieldCheck :size="16" /></span>
          <div><strong>管理员身份</strong><small>Logto 或本地账号</small></div>
        </li>
        <li :class="{ active: step === 3 }">
          <span><Check :size="16" /></span>
          <div><strong>确认启用</strong><small>创建结构并启动服务</small></div>
        </li>
      </ol>
      <div class="setup-environment"><span>运行环境</span><strong>{{ status.environment }}</strong></div>
    </aside>

    <section class="setup-main">
      <header class="setup-header">
        <div>
          <p class="eyebrow">首次配置</p>
          <h1>{{ step === 1 ? '连接数据服务' : step === 2 ? '建立管理员身份' : '确认并启用' }}</h1>
          <p class="page-description">
            {{ step === 1 ? 'GravityLink 依赖 MySQL 与 Redis，先确认它们可以连通。' : step === 2 ? '首位完成验证的账号将成为超级管理员。' : '检查配置摘要，完成后立即进入管理端。' }}
          </p>
        </div>
        <span class="setup-progress">步骤 {{ step }} / 3</span>
      </header>

      <div v-if="status.reason && step === 1" class="setup-notice">
        <strong>系统正在等待初始化</strong><span>{{ status.reason }}</span>
      </div>

      <div v-if="step === 1" class="setup-form">
        <section class="setup-section">
          <div class="setup-section-head"><h2>MySQL</h2><p>保存短链、域名、落地页、用户与统计数据。</p></div>
          <div class="form-grid">
            <label class="span-2">Host<input v-model="form.mysqlHost" /></label>
            <label>Port<input v-model="form.mysqlPort" inputmode="numeric" /></label>
            <label>Database<input v-model="form.mysqlDatabase" /></label>
            <label>User<input v-model="form.mysqlUser" autocomplete="username" /></label>
            <label>Password<input v-model="form.mysqlPassword" type="password" autocomplete="new-password" /></label>
            <label class="span-4">连接参数<input v-model="form.mysqlParams" /></label>
            <details class="advanced span-4"><summary>使用完整 DSN</summary><label>MYSQL_DSN<input v-model="form.mysqlDSN" /></label></details>
          </div>
        </section>
        <section class="setup-section">
          <div class="setup-section-head"><h2>Redis</h2><p>用于访问缓存、计数和异步队列。</p></div>
          <div class="form-grid">
            <label class="span-2">Host<input v-model="form.redisHost" /></label>
            <label>Port<input v-model="form.redisPort" inputmode="numeric" /></label>
            <label>DB<input v-model.number="form.redisDB" min="0" type="number" /></label>
            <label class="span-2">Password<input v-model="form.redisPassword" type="password" autocomplete="new-password" /></label>
            <label class="span-2">完整地址（可选）<input v-model="form.redisAddr" placeholder="redis:6379" /></label>
          </div>
        </section>
        <p class="field-hint">建议先点击「测试连接」确认可达，再进入下一步。</p>
      </div>

      <div v-else-if="step === 2" class="setup-form">
        <div class="segmented auth-selector">
          <button :class="{ active: form.authMode === 'logto' }" type="button" @click="chooseMode('logto')">
            <ShieldCheck :size="17" /> Logto
          </button>
          <button :class="{ active: form.authMode === 'local' }" type="button" @click="chooseMode('local')">
            <KeyRound :size="17" /> 本地账号
          </button>
        </div>

        <section v-if="form.authMode === 'logto'" class="setup-section">
          <div class="setup-section-head"><h2>Logto OIDC</h2><p>先检查 Discovery 配置，再跳转到 Logto 完成身份验证。</p></div>
          <div class="form-grid">
            <label class="span-2">Issuer<input v-model="form.issuer" placeholder="https://logto.example.com/oidc" /></label>
            <label class="span-2">应用 ID<input v-model="form.appID" autocomplete="off" /></label>
            <div class="field-control span-2">
              <div class="label-with-action"><label for="setup-audience">API Audience</label><button type="button" @click="useSuggestedAudience">使用建议值</button></div>
              <input id="setup-audience" v-model="form.audience" placeholder="https://admin.example.com/api" />
              <small>必须与 Logto“API 资源”的 API 标识符完全一致。</small>
            </div>
            <div class="field-control span-2">
              <div class="label-with-action"><label for="setup-admin-url">管理端公开 URL</label><button type="button" @click="useCurrentOrigin">使用当前地址</button></div>
              <input id="setup-admin-url" v-model="form.adminBaseURL" placeholder="https://admin.example.com" />
              <small>填写用户实际访问管理端的协议和域名，不包含路径。</small>
            </div>
            <label class="span-2">Scopes<input v-model="form.scopes" /></label>
            <label class="span-2">后续可授权角色<input v-model="form.allowedRoles" placeholder="admin" /></label>
            <details class="advanced span-4">
              <summary>高级配置</summary>
              <label>JWKS URL<input v-model="form.jwksURL" placeholder="留空后从 Discovery 获取" /></label>
            </details>
          </div>
          <div class="logto-guide">
            <div class="logto-guide-head">
              <div><h3>同步填写 Logto 应用设置</h3><p>在 Logto 的 GravityLink 单页应用中逐项添加，URI 必须完全一致。</p></div>
              <span>SPA + PKCE</span>
            </div>
            <div class="logto-guide-list">
              <div>
                <strong>重定向 URIs</strong>
                <div class="copy-values">
                  <code>{{ setupRedirectURI }}</code>
                  <button class="icon-button" type="button" title="复制初始化回调 URI" @click="copyValue('setup', setupRedirectURI)">
                    <Check v-if="copied === 'setup'" :size="16" /><Clipboard v-else :size="16" />
                  </button>
                  <code>{{ loginRedirectURI }}</code>
                  <button class="icon-button" type="button" title="复制登录回调 URI" @click="copyValue('login', loginRedirectURI)">
                    <Check v-if="copied === 'login'" :size="16" /><Clipboard v-else :size="16" />
                  </button>
                </div>
              </div>
              <div>
                <strong>退出登录后重定向 URI</strong>
                <div class="copy-value">
                  <code>{{ postLogoutRedirectURI }}</code>
                  <button class="icon-button" type="button" title="复制退出回调 URI" @click="copyValue('logout', postLogoutRedirectURI)">
                    <Check v-if="copied === 'logout'" :size="16" /><Clipboard v-else :size="16" />
                  </button>
                </div>
              </div>
              <div>
                <strong>CORS 允许的来源</strong>
                <div class="copy-value">
                  <code>{{ adminOrigin }}</code>
                  <button class="icon-button" type="button" title="复制 CORS 来源" @click="copyValue('cors', adminOrigin)">
                    <Check v-if="copied === 'cors'" :size="16" /><Clipboard v-else :size="16" />
                  </button>
                </div>
              </div>
              <div>
                <strong>API 资源</strong>
                <div class="copy-value">
                  <code>{{ form.audience || '请先填写上方 API Audience' }}</code>
                  <button class="icon-button" type="button" title="复制 API 标识符" :disabled="!form.audience" @click="copyValue('audience', form.audience)">
                    <Check v-if="copied === 'audience'" :size="16" /><Clipboard v-else :size="16" />
                  </button>
                </div>
                <small>在 Logto“API 资源”中新建 GravityLink API，并将 API 标识符设置为此值。</small>
              </div>
            </div>
            <div class="logto-options">
              <span><strong>刷新令牌：</strong>保持关闭</span>
              <span><strong>后端通道注销 URI：</strong>留空</span>
              <span><strong>同时设备限制：</strong>按需设置或留空</span>
            </div>
          </div>
          <button class="primary verify-button" type="button" :disabled="busy" @click="verifyLogto">
            <LoaderCircle v-if="busy" class="spin" :size="17" />
            <ShieldCheck v-else :size="17" />
            检查配置并使用 Logto 验证
          </button>
        </section>

        <section v-else class="setup-section">
          <div class="setup-section-head"><h2>本地超级管理员</h2><p>适合内网或不接入统一身份平台的部署。</p></div>
          <div class="form-grid">
            <label class="span-2">用户名<input v-model="form.username" autocomplete="username" placeholder="admin" /></label>
            <label class="span-2">密码<input v-model="form.password" type="password" autocomplete="new-password" /></label>
            <label class="span-2">确认密码<input v-model="form.passwordConfirm" type="password" autocomplete="new-password" /></label>
          </div>
          <p class="field-hint">密码至少 10 位。系统仅保存 bcrypt 哈希。</p>
          <button class="primary verify-button" type="button" :disabled="busy || !localReady" @click="verifyLocal">
            <LoaderCircle v-if="busy" class="spin" :size="17" /><KeyRound v-else :size="17" />创建管理员身份
          </button>
        </section>
      </div>

      <div v-else class="setup-review">
        <section><span>MySQL</span><strong>{{ form.mysqlHost }}:{{ form.mysqlPort }} / {{ form.mysqlDatabase }}</strong><button type="button" @click="editDatabase">修改</button></section>
        <section><span>Redis</span><strong>{{ form.redisAddr || `${form.redisHost}:${form.redisPort}` }} / DB {{ form.redisDB }}</strong><button type="button" @click="editDatabase">修改</button></section>
        <section><span>超级管理员</span><strong>{{ ownerName }} · {{ form.authMode === 'logto' ? 'Logto' : '本地账号' }}</strong><button type="button" @click="editAuth">修改</button></section>
        <div class="setup-final-note">完成后会自动创建或升级数据库结构，并锁定初始化接口。业务数据不会因重新配置而被清空。</div>
      </div>

      <p v-if="success" class="success-message">{{ success }}</p>
      <p v-if="error" class="error setup-error">{{ error }}</p>

      <footer class="setup-actions">
        <button v-if="step > 1" class="ghost icon-text" type="button" :disabled="busy" @click="step -= 1"><ChevronLeft :size="16" />上一步</button>
        <span v-else></span>
        <div>
          <button v-if="step === 1" class="ghost" type="button" :disabled="busy || !databaseReady" @click="testConnections">{{ busy ? '正在测试...' : dbTested ? '重新测试连接' : '测试连接' }}</button>
          <button v-if="step === 1" class="primary icon-text" type="button" :disabled="busy || !databaseReady" @click="step = 2">继续<ChevronRight :size="16" /></button>
          <button v-if="step === 2" class="primary icon-text" type="button" :disabled="busy || !authVerified" @click="step = 3">
            {{ authVerified ? '继续' : '请先完成管理员验证' }}<ChevronRight :size="16" />
          </button>
          <button v-if="step === 3" class="primary icon-text" type="button" :disabled="busy || !authVerified" @click="finish">
            <LoaderCircle v-if="busy" class="spin" :size="16" /><Check v-else :size="16" />{{ busy ? '正在启用...' : '完成初始化' }}
          </button>
        </div>
      </footer>
    </section>
  </main>
</template>
