<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { KeyRound, Link2, LayoutTemplate, LogIn, BarChart3 } from '@lucide/vue';
import { NAlert, NButton, NForm, NFormItem, NInput } from 'naive-ui';
import { login } from '../auth';
import { useAuthStore } from '../stores/auth';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();

const form = reactive({ username: '', password: '' });
const submitting = ref(false);
const error = ref('');

const isDevMode = computed(() => Boolean(auth.config?.auth_disabled));

onMounted(async () => {
  try {
    await auth.loadConfig();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '无法加载认证配置';
  }
});

async function submitLocal() {
  if (!form.username || !form.password) return;
  submitting.value = true;
  error.value = '';
  try {
    await auth.signInLocal(form.username.trim(), form.password);
    await redirectBack();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '登录失败，请检查账号密码';
  } finally {
    submitting.value = false;
  }
}

async function signInLogto() {
  if (!auth.config) return;
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
  sessionStorage.setItem('gravitylink_login_redirect', redirect);
  await login(auth.config);
}

async function redirectBack() {
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
  await router.replace(redirect);
}
</script>

<template>
  <main class="login-shell">
    <!-- 左侧品牌区 -->
    <section class="login-hero">
      <div class="hero-brand">
        <span class="brand-mark">G</span>
        <strong>GravityLink</strong>
      </div>
      <h1>短链接与活码，一个控制台全部搞定</h1>
      <p class="hero-sub">管理短链接、渠道码、群活码与落地页，跟踪每一次访问。</p>
      <ul class="hero-features">
        <li><Link2 :size="18" />短链接 / 渠道码 / 群活码统一管理</li>
        <li><LayoutTemplate :size="18" />落地页模板与主题定制</li>
        <li><BarChart3 :size="18" />多维度访问统计</li>
      </ul>
      <footer class="hero-footer">© GravityLink</footer>
    </section>

    <!-- 右侧登录表单 -->
    <section class="login-form-side">
      <div class="login-panel">
        <header>
          <h2>管理员登录</h2>
          <p class="muted">
            {{ isDevMode ? '当前为开发模式（认证已关闭）。' : auth.config?.mode === 'local' ? '使用初始化时创建的本地账号继续。' : '使用已获授权的 Logto 账号继续。' }}
          </p>
        </header>

        <NAlert v-if="error" type="error" :show-icon="true">{{ error }}</NAlert>

        <NForm v-if="auth.config?.mode === 'local'" @submit.prevent="submitLocal">
          <NFormItem label="用户名">
            <NInput v-model:value="form.username" placeholder="请输入用户名" autocomplete="username" autofocus />
          </NFormItem>
          <NFormItem label="密码">
            <NInput
              v-model:value="form.password"
              type="password"
              show-password-on="click"
              placeholder="请输入密码"
              autocomplete="current-password"
              @keyup.enter="submitLocal"
            />
          </NFormItem>
          <NButton
            type="primary"
            block
            size="large"
            :loading="submitting"
            :disabled="!form.username || !form.password"
            attr-type="submit"
          >
            <template #icon><KeyRound :size="16" /></template>
            登录
          </NButton>
        </NForm>

        <NButton v-else type="primary" block size="large" :disabled="!auth.config" @click="signInLogto">
          <template #icon><LogIn :size="16" /></template>
          使用 Logto 登录
        </NButton>
      </div>
    </section>
  </main>
</template>

<style scoped>
.login-shell {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(400px, 1fr) minmax(420px, 1fr);
}

/* 左侧品牌区：铺满、深色渐变 */
.login-hero {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: var(--space-4);
  padding: var(--space-10);
  background:
    radial-gradient(ellipse at 20% 0%, rgba(20, 184, 166, 0.25), transparent 55%),
    radial-gradient(ellipse at 80% 100%, rgba(15, 118, 110, 0.35), transparent 50%),
    var(--color-bg-sidebar);
  color: var(--color-text-inverse);
  overflow: hidden;
}

.hero-brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-6);
}

.hero-brand strong {
  font-size: var(--font-size-xl);
}

.login-hero h1 {
  font-size: 34px;
  line-height: 1.25;
  font-weight: 600;
  max-width: 480px;
}

.hero-sub {
  color: rgba(255, 255, 255, 0.65);
  font-size: var(--font-size-lg);
  max-width: 440px;
  line-height: 1.6;
}

.hero-features {
  margin: var(--space-6) 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: var(--space-4);
}

.hero-features li {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: rgba(255, 255, 255, 0.85);
  font-size: var(--font-size-base);
}

.hero-features svg {
  color: var(--color-accent);
  flex: 0 0 auto;
}

.hero-footer {
  position: absolute;
  bottom: var(--space-6);
  left: var(--space-10);
  color: rgba(255, 255, 255, 0.35);
  font-size: var(--font-size-sm);
}

/* 右侧表单区：铺满、居中 */
.login-form-side {
  display: grid;
  place-items: center;
  padding: var(--space-8);
  background: var(--color-bg-page);
}

.login-panel {
  width: min(400px, 100%);
  display: grid;
  gap: var(--space-5);
}

.login-panel h2 {
  font-size: var(--font-size-2xl);
  font-weight: 600;
  margin-bottom: var(--space-2);
}

@media (max-width: 900px) {
  .login-shell {
    grid-template-columns: 1fr;
  }
  .login-hero {
    display: none;
  }
  .login-form-side {
    min-height: 100vh;
  }
}
</style>
