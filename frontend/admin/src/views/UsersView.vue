<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import { RefreshCw, ShieldCheck, UserCheck, Users as UsersIcon } from '@lucide/vue';
import {
  NAlert,
  NAvatar,
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NIcon,
  NPopconfirm,
  NSelect,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui';
import { listUsers, updateUserRole, updateUserStatus, type UserItem } from '../api';
import { useAuthStore } from '../stores/auth';

const message = useMessage();
const auth = useAuthStore();

const items = ref<UserItem[]>([]);
const loading = ref(false);

const roleLabelMap: Record<string, string> = { super_admin: '超级管理员', admin: '管理员', user: '普通用户' };
const roleTypeMap: Record<string, 'error' | 'info' | 'default'> = { super_admin: 'error', admin: 'info', user: 'default' };
const statusLabelMap: Record<string, string> = { active: '正常', pending: '待授权', disabled: '已停用' };
const statusTypeMap: Record<string, 'success' | 'warning' | 'default'> = { active: 'success', pending: 'warning', disabled: 'default' };
const sourceLabelMap: Record<string, string> = { logto: 'Logto', local: '本地', development: '开发模式' };

const roleOptions = [
  { label: '普通用户', value: 'user' },
  { label: '管理员', value: 'admin' },
];

const columns: DataTableColumns<UserItem> = [
  {
    title: '账号',
    key: 'username',
    render: (row) =>
      h('div', { style: 'display:flex;align-items:center;gap:12px' }, [
        h(NAvatar, { round: true, size: 'small', style: 'background:var(--color-primary);color:#fff' }, () =>
          row.username.slice(0, 1).toUpperCase(),
        ),
        h('div', {}, [
          h('div', { style: 'font-weight:500' }, row.username),
          h('div', { class: 'muted', style: 'font-size:12px' }, row.email || '未提供邮箱'),
        ]),
      ]),
  },
  { title: '来源', key: 'auth_source', width: 110, render: (row) => sourceLabelMap[row.auth_source] || row.auth_source },
  {
    title: '角色',
    key: 'role',
    width: 120,
    render: (row) =>
      h(NTag, { type: roleTypeMap[row.role] || 'default', size: 'small' }, () => roleLabelMap[row.role] || row.role),
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row) =>
      h(NTag, { type: statusTypeMap[row.status] || 'default', size: 'small', round: true }, () => statusLabelMap[row.status] || row.status),
  },
  {
    title: '最近登录',
    key: 'last_login_at',
    width: 180,
    render: (row) => (row.last_login_at ? new Date(row.last_login_at).toLocaleString('zh-CN', { hour12: false }) : '—'),
  },
  {
    title: '操作',
    key: 'actions',
    width: 220,
    render: (row) => {
      if (!auth.isSuperAdmin || row.role === 'super_admin') {
        return h('span', { class: 'muted' }, '受保护');
      }
      return h('div', { style: 'display:flex;gap:8px;align-items:center' }, [
        h(NSelect, {
          value: row.role,
          options: roleOptions,
          size: 'small',
          style: 'width:120px',
          onUpdateValue: (value: 'admin' | 'user') => setRole(row, value),
        }),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => setStatus(row, row.status === 'active' ? 'disabled' : 'active'),
            positiveText: '确认',
            negativeText: '取消',
          },
          {
            trigger: () =>
              h(
                NButton,
                { size: 'small', tertiary: true, type: row.status === 'active' ? 'error' : 'success' },
                {
                  icon: () => h(NIcon, { size: 14 }, { default: () => h(UserCheck) }),
                  default: () => (row.status === 'active' ? '停用' : '启用'),
                },
              ),
            default: () => (row.status === 'active' ? `停用账号 ${row.username}？` : `启用账号 ${row.username}？`),
          },
        ),
      ]);
    },
  },
];

onMounted(refresh);

async function refresh() {
  loading.value = true;
  try {
    items.value = (await listUsers()).items;
  } catch (err) {
    message.error(err instanceof Error ? err.message : '加载账号失败');
  } finally {
    loading.value = false;
  }
}

async function setRole(item: UserItem, role: 'admin' | 'user') {
  try {
    await updateUserRole(item.id, role);
    message.success(`已将 ${item.username} 设为${roleLabelMap[role]}`);
    await refresh();
  } catch (err) {
    message.error(err instanceof Error ? err.message : '更新角色失败');
  }
}

async function setStatus(item: UserItem, status: 'active' | 'disabled') {
  try {
    await updateUserStatus(item.id, status);
    message.success(status === 'active' ? `已启用 ${item.username}` : `已停用 ${item.username}`);
    await refresh();
  } catch (err) {
    message.error(err instanceof Error ? err.message : '更新状态失败');
  }
}
</script>

<template>
  <div class="users-page">
    <NAlert type="info" :show-icon="true">
      <template #icon><ShieldCheck :size="16" /></template>
      <strong>后端强制执行权限</strong> · 超级管理员可授权管理员、停用账号；界面选择不会覆盖后端角色。
    </NAlert>

    <NCard>
      <template #header>
        <div class="card-head">
          <div>
            <strong>账号与权限</strong>
            <p class="muted">Logto 新账号首次访问后进入待授权状态</p>
          </div>
          <NButton :loading="loading" @click="refresh">
            <template #icon><RefreshCw :size="16" /></template>
            刷新
          </NButton>
        </div>
      </template>

      <NDataTable :columns="columns" :data="items" :loading="loading" :pagination="{ pageSize: 20 }" :bordered="false" size="small">
        <template #empty>
          <NEmpty description="暂无账号">
            <template #icon><UsersIcon :size="32" /></template>
          </NEmpty>
        </template>
      </NDataTable>
    </NCard>
  </div>
</template>

<style scoped>
.users-page {
  display: grid;
  gap: var(--space-4);
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  width: 100%;
}

.card-head strong {
  font-size: var(--font-size-lg);
  display: block;
}

.card-head p {
  margin-top: var(--space-1);
  font-size: var(--font-size-md);
}
</style>
