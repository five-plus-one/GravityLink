<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { KeyRound, LogIn } from '@lucide/vue';
import { NAlert, NButton, NForm, NFormItem, NInput } from 'naive-ui';
import { login } from '../auth';
import { useAuthStore } from '../stores/auth';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();

const form = reactive({ username: '', password: '' });
const submitting = ref(false);
const error = ref('');

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
    <section class="login-panel">
      <div class="login-brand">
        <span class="brand-mark">G</span>
        <div>
          <strong>GravityLink</strong>
          <small>管理控制台</small>
        </div>
      </div>

      <div class="login-head">
        <h1>管理员登录</h1>
        <p class="muted">
          {{ auth.config?.mode === 'local' ? '使用初始化时创建的本地账号继续。' : '使用已获授权的 Logto 账号继续。' }}
        </p>
      </div>

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
    </section>
  </main>
</template>

<style scoped>
.login-shell {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: var(--space-6);
  background: var(--color-bg-page);
}

.login-panel {
  width: min(420px, 100%);
  padding: var(--space-8);
  display: grid;
  gap: var(--space-6);
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
}

.login-brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.login-brand strong,
.login-brand small {
  display: block;
}

.login-brand small {
  margin-top: var(--space-1);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-sm);
}

.login-head h1 {
  font-size: var(--font-size-2xl);
  font-weight: 600;
  margin-bottom: var(--space-2);
}
</style>
