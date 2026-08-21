<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { NSpin } from 'naive-ui';
import { handleCallback, recallSetupOIDC } from '../auth';
import { useAuthStore } from '../stores/auth';
import { useSetupStore } from '../stores/setup';
import { claimSetupLogto } from '../setup';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const setup = useSetupStore();

const error = ref('');

onMounted(async () => {
  try {
    if (route.name === 'setup-callback') {
      await handleSetupCallback();
    } else {
      await handleLoginCallback();
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '登录回调处理失败';
  }
});

async function handleLoginCallback() {
  const config = await auth.loadConfig();
  const returnPath = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
  await handleCallback(config, '/auth/callback', returnPath);
  await auth.fetchUser();
  await router.replace(returnPath);
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
</script>

<template>
  <div class="state-screen">
    <NSpin v-if="!error" size="medium" />
    <span v-if="!error">正在完成登录…</span>
    <span v-else style="color: var(--color-error)">{{ error }}</span>
  </div>
</template>
