<script setup lang="ts">
import { computed } from 'vue';

import { NCard, NDescriptions, NDescriptionsItem, NTag } from 'naive-ui';
import { useAuthStore } from '../stores/auth';

const auth = useAuthStore();

const roleLabel = computed(() => ({ super_admin: '超级管理员', admin: '管理员', user: '普通用户' })[auth.user?.role || 'user']);
const sourceLabel = computed(() => ({ logto: 'Logto OIDC', local: '本地账号', development: '开发模式' })[auth.user?.auth_source || 'local']);
const statusLabel = computed(() => ({ active: '正常', pending: '待授权', disabled: '已停用' })[auth.user?.status || 'pending']);
</script>

<template>
 <NCard title="个人资料" class="profile-card">
  <NDescriptions :column="1" label-placement="left" :label-style="{width:'90px'}">
   <NDescriptionsItem label="用户名">{{ auth.user?.username }}</NDescriptionsItem>
   <NDescriptionsItem label="邮箱">{{ auth.user?.email || '未提供' }}</NDescriptionsItem>
   <NDescriptionsItem label="登录方式">{{ sourceLabel }}</NDescriptionsItem>
   <NDescriptionsItem label="角色">{{ roleLabel }}</NDescriptionsItem>
   <NDescriptionsItem label="账号状态"><NTag :type="auth.user?.status==='active'?'success':'warning'" size="small">{{ statusLabel }}</NTag></NDescriptionsItem>
  </NDescriptions>
 </NCard>
</template>
<style scoped>.profile-card{max-width:760px;margin-left:0}.profile-card :deep(.n-descriptions-table-content){padding-bottom:20px}</style>
