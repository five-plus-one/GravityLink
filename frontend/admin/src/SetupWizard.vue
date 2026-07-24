<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { completeSetup, testSetupDatabase, type SetupPayload, type SetupStatus } from './setup';

const props = defineProps<{ status: SetupStatus }>();
const emit = defineEmits<{ completed: [] }>();

const step = ref(1);
const busy = ref(false);
const error = ref('');
const testMessage = ref('');

const form = reactive({
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
  authDisabled: props.status.auth.disabled,
  issuer: props.status.auth.issuer || '',
  appID: props.status.auth.app_id || '',
  audience: props.status.auth.audience || '',
  jwksURL: props.status.auth.jwks_url || '',
  scopes: props.status.auth.scopes || 'openid profile email',
  adminBaseURL: props.status.auth.admin_base_url || window.location.origin,
  allowedRoles: props.status.auth.allowed_roles?.join(', ') || 'admin',
});

const isProduction = computed(() => props.status.environment === 'production');
const canContinueDatabase = computed(
  () =>
    Boolean(form.mysqlDSN) ||
    Boolean(form.mysqlHost && form.mysqlPort && form.mysqlDatabase && form.mysqlUser),
);
const canComplete = computed(
  () =>
    form.authDisabled ||
    Boolean(form.issuer && form.appID && form.audience && form.adminBaseURL),
);

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
      disabled: form.authDisabled,
      issuer: form.issuer.trim(),
      app_id: form.appID.trim(),
      audience: form.audience.trim(),
      jwks_url: form.jwksURL.trim(),
      scopes: form.scopes.trim(),
      admin_base_url: form.adminBaseURL.trim(),
      allowed_roles: form.allowedRoles.split(',').map((role) => role.trim()).filter(Boolean),
    },
  };
}

async function testConnections() {
  busy.value = true;
  error.value = '';
  testMessage.value = '';
  try {
    const result = await testSetupDatabase(payload());
    testMessage.value = result.schema_ready
      ? 'MySQL、Redis 连接正常，数据库表结构已就绪。'
      : '连接正常，但数据库尚未导入 GravityLink 表结构。';
  } catch (err) {
    error.value = err instanceof Error ? err.message : '连接测试失败';
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
    error.value = err instanceof Error ? err.message : '初始化失败';
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <main class="setup-shell">
    <aside class="setup-aside">
      <div class="setup-brand">
        <span class="brand-mark">G</span>
        <div>
          <strong>GravityLink</strong>
          <small>首次初始化</small>
        </div>
      </div>
      <ol class="setup-steps">
        <li :class="{ active: step === 1, done: step > 1 }">
          <span>1</span>
          <div><strong>数据服务</strong><small>MySQL 与 Redis</small></div>
        </li>
        <li :class="{ active: step === 2, done: step > 2 }">
          <span>2</span>
          <div><strong>身份认证</strong><small>Logto 与管理权限</small></div>
        </li>
        <li :class="{ active: step === 3 }">
          <span>3</span>
          <div><strong>确认启用</strong><small>保存并启动服务</small></div>
        </li>
      </ol>
      <div class="setup-environment">
        <span>运行环境</span>
        <strong>{{ status.environment }}</strong>
      </div>
    </aside>

    <section class="setup-main">
      <header class="setup-header">
        <div>
          <p class="eyebrow">系统配置</p>
          <h1 v-if="step === 1">连接数据服务</h1>
          <h1 v-else-if="step === 2">配置管理员登录</h1>
          <h1 v-else>检查并启用 GravityLink</h1>
        </div>
        <span class="setup-progress">步骤 {{ step }} / 3</span>
      </header>

      <div v-if="status.reason && step === 1" class="setup-notice">
        <strong>当前无法启动业务服务</strong>
        <span>{{ status.reason }}</span>
      </div>

      <form v-if="step === 1" class="setup-form" @submit.prevent>
        <section class="setup-section">
          <div class="setup-section-head">
            <div><h2>MySQL</h2><p>用于保存链接、域名、页面和统计数据</p></div>
          </div>
          <div class="form-grid">
            <label class="span-2">Host<input v-model="form.mysqlHost" autocomplete="off" placeholder="mysql" /></label>
            <label>Port<input v-model="form.mysqlPort" inputmode="numeric" placeholder="3306" /></label>
            <label>Database<input v-model="form.mysqlDatabase" autocomplete="off" placeholder="gravitylink" /></label>
            <label>User<input v-model="form.mysqlUser" autocomplete="username" placeholder="gravitylink" /></label>
            <label>Password<input v-model="form.mysqlPassword" autocomplete="new-password" type="password" /></label>
            <label class="span-4">连接参数<input v-model="form.mysqlParams" autocomplete="off" /></label>
            <details class="advanced span-4">
              <summary>使用完整 DSN</summary>
              <label>MYSQL_DSN<input v-model="form.mysqlDSN" autocomplete="off" placeholder="留空使用上方字段" /></label>
            </details>
          </div>
        </section>

        <section class="setup-section">
          <div class="setup-section-head">
            <div><h2>Redis</h2><p>用于短链缓存、访问计数与异步队列</p></div>
          </div>
          <div class="form-grid">
            <label class="span-2">Host<input v-model="form.redisHost" autocomplete="off" placeholder="redis" /></label>
            <label>Port<input v-model="form.redisPort" inputmode="numeric" placeholder="6379" /></label>
            <label>DB<input v-model.number="form.redisDB" min="0" type="number" /></label>
            <label class="span-2">Password<input v-model="form.redisPassword" autocomplete="new-password" type="password" /></label>
            <label class="span-2">完整地址（可选）<input v-model="form.redisAddr" autocomplete="off" placeholder="redis:6379" /></label>
          </div>
        </section>
      </form>

      <form v-else-if="step === 2" class="setup-form" @submit.prevent>
        <section class="setup-section">
          <div class="auth-mode">
            <div>
              <h2>Logto 身份认证</h2>
              <p>管理端使用 OIDC Authorization Code + PKCE 登录。</p>
            </div>
            <label class="switch-row">
              <input v-model="form.authDisabled" :disabled="isProduction" type="checkbox" />
              <span>开发环境关闭鉴权</span>
            </label>
          </div>
          <p v-if="isProduction" class="field-hint">production 环境强制启用身份认证。</p>
          <div v-if="!form.authDisabled" class="form-grid auth-fields">
            <label class="span-2">Issuer<input v-model="form.issuer" placeholder="https://logto.example.com/oidc" /></label>
            <label class="span-2">SPA App ID<input v-model="form.appID" autocomplete="off" /></label>
            <label class="span-2">API Audience<input v-model="form.audience" placeholder="https://gravitylink.example.com/api" /></label>
            <label class="span-2">管理端公开 URL<input v-model="form.adminBaseURL" placeholder="https://admin.example.com" /></label>
            <label class="span-2">Scopes<input v-model="form.scopes" /></label>
            <label class="span-2">允许管理的角色<input v-model="form.allowedRoles" placeholder="admin" /></label>
            <details class="advanced span-4">
              <summary>高级 Logto 配置</summary>
              <label>JWKS URL<input v-model="form.jwksURL" placeholder="留空自动推导" /></label>
            </details>
          </div>
          <div v-else class="dev-mode-note">
            当前管理端将以本地 `admin` 身份进入。此选项只适用于开发和内网调试。
          </div>
        </section>
      </form>

      <div v-else class="setup-review">
        <section>
          <span>MySQL</span>
          <strong>{{ form.mysqlHost }}:{{ form.mysqlPort }} / {{ form.mysqlDatabase }}</strong>
          <button type="button" @click="step = 1">修改</button>
        </section>
        <section>
          <span>Redis</span>
          <strong>{{ form.redisAddr || `${form.redisHost}:${form.redisPort}` }} / DB {{ form.redisDB }}</strong>
          <button type="button" @click="step = 1">修改</button>
        </section>
        <section>
          <span>管理端认证</span>
          <strong>{{ form.authDisabled ? '开发模式（未启用 Logto）' : form.issuer }}</strong>
          <button type="button" @click="step = 2">修改</button>
        </section>
        <div class="setup-final-note">
          配置将写入服务器的持久化配置文件。完成后匿名初始化接口会立即锁定。
        </div>
      </div>

      <p v-if="testMessage" class="success-message">{{ testMessage }}</p>
      <p v-if="error" class="error setup-error">{{ error }}</p>

      <footer class="setup-actions">
        <button v-if="step > 1" class="ghost" type="button" :disabled="busy" @click="step -= 1">上一步</button>
        <span v-else></span>
        <div>
          <button v-if="step === 1" class="ghost" type="button" :disabled="busy || !canContinueDatabase" @click="testConnections">
            {{ busy ? '正在测试...' : '测试连接' }}
          </button>
          <button v-if="step < 3" class="primary" type="button" :disabled="busy || (step === 1 ? !canContinueDatabase : !canComplete)" @click="step += 1">
            继续
          </button>
          <button v-else class="primary" type="button" :disabled="busy || !canComplete" @click="finish">
            {{ busy ? '正在启用...' : '完成初始化' }}
          </button>
        </div>
      </footer>
    </section>
  </main>
</template>
