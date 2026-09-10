<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { NButton, NSpin } from 'naive-ui';
import { handleCallback, recallSetupOIDC } from '../auth';
import { useAuthStore } from '../stores/auth';
import { useSetupStore } from '../stores/setup';
import { claimSetupLogto } from '../setup';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const setup = useSetupStore();

const error = ref('');
const unauthorized = ref(false);

onMounted(async () => {
  try {
    if (route.name === 'setup-callback') {
      await handleSetupCallback();
    } else {
      await handleLoginCallback();
    }
  } catch (err) {
    const msg = err instanceof Error ? err.message : '登录回调处理失败';
    if (msg.includes('pending') || msg.includes('未授权') || msg.includes('forbidden') || msg.includes('403')) {
      unauthorized.value = true;
      error.value = '你的账号尚未获得管理端授权，请联系超级管理员在「账号与权限」中审批后再试。';
    } else {
      error.value = msg;
    }
  }
});

async function handleLoginCallback() {
  const config = await auth.loadConfig();
  const returnPath = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
  await handleCallback(config, '/auth/callback', returnPath);
  // 直接调 /api/v1/auth/me 获取具体错误（pending/disabled 等）
  const { loadCurrentUser } = await import('../auth');
  try {
    const user = await loadCurrentUser();
    auth.user = user;
    auth.loaded = true;
    await router.replace(returnPath);
  } catch (err) {
    const msg = err instanceof Error ? err.message : '';
    if (msg.includes('待审核') || msg.includes('停用') || msg.includes('pending') || msg.includes('forbidden')) {
      unauthorized.value = true;
      error.value = msg || '你的账号尚未获得管理端授权，请联系超级管理员审批。';
    } else {
      error.value = msg || '登录失败，请重试';
    }
  }
}

async function handleSetupCallback() {
  const config = recallSetupOIDC();
  if (!config) {
    throw new Error('登录会话已失效，请重新检查 Logto 配置');
  }
  await handleCallback(config, '/setup/auth/callback', '/setup');
  const result = await claimSetupLogto();
  if (setup.status) {
    setup.status.auth.verified = result.verified;
    setup.status.auth.owner_username = result.username;
    setup.status.auth.owner_email = result.email;
  }
  await router.replace({ name: 'setup', query: { verified: '1' } });
}

function goLogin() {
  router.replace('/login');
}
</script>

<template>
  <div class="state-screen">
    <template v-if="!error">
      <NSpin size="medium" />
      <span>正在完成登录…</span>
    </template>
    <template v-else>
      <div class="error-box">
        <p class="error-msg" :class="{ warn: unauthorized }">{{ error }}</p>
        <NButton type="primary" @click="goLogin">返回登录</NButton>
      </div>
    </template>
  </div>
</template>

<style scoped>
.state-screen {
  min-height: 100vh;
  display: grid;
  place-items: center;
  gap: 16px;
  color: var(--color-text-secondary, #48565e);
}

.error-box {
  display: grid;
  gap: 20px;
  justify-items: center;
  max-width: 420px;
  text-align: center;
  padding: 32px;
}

.error-msg {
  font-size: 15px;
  line-height: 1.7;
  color: var(--color-error, #dc2626);
}

.error-msg.warn {
  color: #b45309;
}
</style>
