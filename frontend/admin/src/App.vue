<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { KeyRound, LoaderCircle, LogIn } from '@lucide/vue';
import SetupWizard from './SetupWizard.vue';
import {
  handleCallback,
  loadAuthConfig,
  loadCurrentUser,
  login,
  loginLocal,
  logout,
  logoutSession,
  type AuthConfig,
  type AuthUser,
} from './auth';
import { loadSetupStatus, type SetupStatus } from './setup';

const loading = ref(true);
const error = ref('');
const setupStatus = ref<SetupStatus | null>(null);
const authConfig = ref<AuthConfig | null>(null);
const user = ref<AuthUser | null>(null);
const username = ref('');
const password = ref('');
const submitting = ref(false);

onMounted(boot);

async function boot() {
  loading.value = true;
  error.value = '';
  try {
    const status = await loadSetupStatus();
    setupStatus.value = status;
    if (status.setup_required) return;

    const config = await loadAuthConfig();
    authConfig.value = config;
    await handleCallback(config);
    try {
      user.value = await loadCurrentUser();
    } catch {
      user.value = null;
    }
  } catch (err) {
    error.value = messageOf(err);
  } finally {
    loading.value = false;
  }
}

async function localSignIn() {
  submitting.value = true;
  error.value = '';
  try {
    user.value = await loginLocal(username.value.trim(), password.value);
    password.value = '';
  } catch (err) {
    error.value = messageOf(err);
  } finally {
    submitting.value = false;
  }
}

async function signOut() {
  if (authConfig.value?.mode === 'logto') {
    logout(authConfig.value);
    return;
  }
  await logoutSession();
  user.value = null;
}

async function setupCompleted() {
  setupStatus.value = null;
  await boot();
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : '操作失败';
}
</script>

<template>
  <div v-if="loading" class="state-screen">
    <LoaderCircle class="spin" :size="24" />
    <span>正在检查系统状态</span>
  </div>

  <SetupWizard
    v-else-if="setupStatus?.setup_required"
    :status="setupStatus"
    @completed="setupCompleted"
  />

  <main v-else-if="!user" class="login-shell">
    <section class="login-panel">
      <div class="login-brand"><span class="brand-mark">G</span><div><strong>GravityLink</strong><small>管理控制台</small></div></div>
      <div>
        <h1>管理员登录</h1>
        <p>{{ authConfig?.mode === 'local' ? '使用初始化时创建的本地账号继续。' : '使用已获授权的 Logto 账号继续。' }}</p>
      </div>
      <form v-if="authConfig?.mode === 'local'" class="login-form" @submit.prevent="localSignIn">
        <label>用户名<input v-model="username" autocomplete="username" autofocus /></label>
        <label>密码<input v-model="password" type="password" autocomplete="current-password" /></label>
        <button class="primary icon-text" type="submit" :disabled="submitting || !username || !password">
          <LoaderCircle v-if="submitting" class="spin" :size="17" /><KeyRound v-else :size="17" />登录
        </button>
      </form>
      <button v-else class="primary login-button icon-text" type="button" :disabled="!authConfig" @click="authConfig && login(authConfig)">
        <LogIn :size="17" />使用 Logto 登录
      </button>
      <p v-if="error" class="error">{{ error }}</p>
    </section>
  </main>

  <RouterView v-else v-slot="{ Component }">
    <component :is="Component" :user="user" :auth-config="authConfig" @logout="signOut" />
  </RouterView>
</template>
