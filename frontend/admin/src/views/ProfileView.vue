<script setup lang="ts">
import { computed } from 'vue';
import { KeyRound, Mail, ShieldCheck, UserRound } from '@lucide/vue';
import { NAvatar, NButton, NCard, NDescriptions, NDescriptionsItem, NTag } from 'naive-ui';
import { useAuthStore } from '../stores/auth';

const auth = useAuthStore();

const roleLabel = computed(() => ({ super_admin: '超级管理员', admin: '管理员', user: '普通用户' })[auth.user?.role || 'user']);
const sourceLabel = computed(() => ({ logto: 'Logto OIDC', local: '本地账号', development: '开发模式' })[auth.user?.auth_source || 'local']);
const statusLabel = computed(() => ({ active: '正常', pending: '待授权', disabled: '已停用' })[auth.user?.status || 'pending']);
</script>

<template>
  <div class="profile-page">
    <NCard class="identity-card">
      <div class="identity-main">
        <NAvatar round :size="76" class="profile-avatar">{{ auth.user?.username?.slice(0, 1).toUpperCase() }}</NAvatar>
        <div>
          <span class="eyebrow">当前登录账号</span>
          <h2>{{ auth.user?.username }}</h2>
          <div class="identity-tags">
            <NTag type="info" round>{{ roleLabel }}</NTag>
            <NTag :type="auth.user?.status === 'active' ? 'success' : 'warning'" round>{{ statusLabel }}</NTag>
          </div>
        </div>
      </div>
      <div class="security-summary">
        <span class="security-icon"><ShieldCheck :size="24" /></span>
        <div><strong>账号安全</strong><p>身份由 {{ sourceLabel }} 验证</p></div>
      </div>
    </NCard>

    <div class="profile-grid">
      <NCard title="基础资料" class="surface-card">
        <NDescriptions :column="1" label-placement="left">
          <NDescriptionsItem label="用户 ID"><span class="field-value"><UserRound :size="16" />{{ auth.user?.id }}</span></NDescriptionsItem>
          <NDescriptionsItem label="用户名"><span class="field-value"><UserRound :size="16" />{{ auth.user?.username }}</span></NDescriptionsItem>
          <NDescriptionsItem label="邮箱"><span class="field-value"><Mail :size="16" />{{ auth.user?.email || '未提供' }}</span></NDescriptionsItem>
          <NDescriptionsItem label="认证来源"><span class="field-value"><KeyRound :size="16" />{{ sourceLabel }}</span></NDescriptionsItem>
        </NDescriptions>
      </NCard>

      <NCard title="权限概览" class="permission-card">
        <div class="permission-hero">
          <span><ShieldCheck :size="26" /></span>
          <div><small>当前角色</small><strong>{{ roleLabel }}</strong></div>
        </div>
        <p>{{ auth.isSuperAdmin ? '拥有全部管理权限，可维护账号授权和系统运行配置。' : '权限由超级管理员分配，具体操作仍由后端进行强制校验。' }}</p>
        <NButton secondary type="primary" block @click="$router.push('/settings')">查看系统设置</NButton>
      </NCard>
    </div>
  </div>
</template>

<style scoped>
.profile-page { display: grid; gap: var(--space-5); }
.identity-card { background: rgba(255,255,255,.9); overflow: hidden; }
.identity-card :deep(.n-card__content) { display: flex; align-items: center; justify-content: space-between; gap: var(--space-6); padding: var(--space-8); }
.identity-main { display: flex; align-items: center; gap: var(--space-5); }
.profile-avatar { background: linear-gradient(135deg, var(--color-primary), #7067f0); color: #fff; font-size: 28px; font-weight: 700; box-shadow: 0 12px 28px rgba(39,135,245,.24); }
.eyebrow { color: var(--color-text-tertiary); font-size: var(--font-size-sm); }
.identity-main h2 { margin: var(--space-1) 0 var(--space-3); font-size: var(--font-size-3xl); }
.identity-tags { display: flex; gap: var(--space-2); }
.security-summary { min-width: 300px; display: flex; align-items: center; gap: var(--space-3); padding: var(--space-4); border: 1px solid var(--color-border); border-radius: var(--radius-lg); background: var(--color-bg-subtle); }
.security-icon { width: 48px; height: 48px; display: grid; place-items: center; border-radius: var(--radius-md); color: var(--color-primary); background: var(--color-primary-soft); }
.security-summary p { margin-top: var(--space-1); color: var(--color-text-tertiary); font-size: var(--font-size-md); }
.profile-grid { display: grid; grid-template-columns: minmax(0,1.35fr) minmax(300px,.65fr); gap: var(--space-5); }
.surface-card { background: rgba(255,255,255,.9); }
.field-value { display: inline-flex; align-items: center; gap: var(--space-2); color: var(--color-text-secondary); }
.field-value svg { color: var(--color-primary); }
.permission-card { color: #fff; background: linear-gradient(145deg, #1b6fd1, #2787f5 58%, #7067f0); }
.permission-card :deep(.n-card-header__main) { color: #fff; }
.permission-card :deep(.n-button) { color: #fff; background: rgba(255,255,255,.14); }
.permission-hero { display: flex; align-items: center; gap: var(--space-3); padding: var(--space-4); border-radius: var(--radius-lg); background: rgba(10,45,110,.22); }
.permission-hero > span { width: 48px; height: 48px; display: grid; place-items: center; border-radius: var(--radius-md); background: rgba(255,255,255,.16); }
.permission-hero small, .permission-hero strong { display: block; }
.permission-hero small { color: rgba(255,255,255,.68); }
.permission-hero strong { margin-top: 2px; font-size: var(--font-size-xl); }
.permission-card p { margin: var(--space-5) 0; color: rgba(255,255,255,.78); line-height: 1.7; }
@media (max-width: 900px) {
  .identity-card :deep(.n-card__content) { align-items: stretch; flex-direction: column; }
  .security-summary { min-width: 0; }
  .profile-grid { grid-template-columns: 1fr; }
}
</style>
