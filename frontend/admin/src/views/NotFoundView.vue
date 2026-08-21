<script setup lang="ts">
import { useRouter } from 'vue-router';
import { NButton, NResult } from 'naive-ui';

const router = useRouter();
</script>

<template>
  <div class="state-screen">
    <NResult status="404" title="页面不存在" description="您访问的地址不存在或已被移除。">
      <template #footer>
        <NButton type="primary" @click="router.replace('/')">返回概览</NButton>
      </template>
    </NResult>
  </div>
</template>
