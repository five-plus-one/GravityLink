<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { RefreshCw, ShieldCheck, UserCheck, Users } from '@lucide/vue';
import { listUsers, updateUserRole, updateUserStatus, type UserItem } from '../api';
import { loadCurrentUser, type AuthUser } from '../auth';

const items = ref<UserItem[]>([]);
const current = ref<AuthUser | null>(null);
const loading = ref(false);
const error = ref('');
onMounted(async () => { current.value = await loadCurrentUser(); await refresh(); });
async function refresh() { loading.value = true; try { items.value = (await listUsers()).items; } catch (err) { error.value = messageOf(err); } finally { loading.value = false; } }
async function setRole(item: UserItem, role: 'admin' | 'user') { try { await updateUserRole(item.id, role); await refresh(); } catch (err) { error.value = messageOf(err); } }
async function setStatus(item: UserItem, status: 'active' | 'disabled') { try { await updateUserStatus(item.id, status); await refresh(); } catch (err) { error.value = messageOf(err); } }
function messageOf(err: unknown) { return err instanceof Error ? err.message : '操作失败'; }
</script>
<template><div class="view">
  <header class="view-header"><div><h1>账号与权限</h1><p>Logto 新账号首次访问后进入待授权状态。</p></div><button class="icon-text" @click="refresh"><RefreshCw :class="{ spin: loading }" :size="17" />刷新</button></header>
  <div class="notice"><ShieldCheck :size="18" /><div><strong>后端强制执行权限</strong><p>超级管理员可授权管理员、停用账号；界面选择不会覆盖后端角色。</p></div></div>
  <p v-if="error" class="error banner">{{ error }}</p>
  <section class="surface"><div class="table-wrap"><table><thead><tr><th>账号</th><th>来源</th><th>角色</th><th>状态</th><th>最近登录</th><th>授权操作</th></tr></thead><tbody>
    <tr v-for="item in items" :key="item.id"><td><div class="identity"><span><Users :size="16" /></span><div><strong>{{ item.username }}</strong><small>{{ item.email || '未提供邮箱' }}</small></div></div></td><td>{{ item.auth_source === 'logto' ? 'Logto' : '本地' }}</td><td><span class="role" :class="item.role">{{ item.role === 'super_admin' ? '超级管理员' : item.role === 'admin' ? '管理员' : '普通用户' }}</span></td><td><span class="status" :class="item.status">{{ item.status }}</span></td><td>{{ item.last_login_at ? new Date(item.last_login_at).toLocaleString() : '—' }}</td>
      <td><div v-if="current?.role === 'super_admin' && item.role !== 'super_admin'" class="row-controls"><select :value="item.role" @change="setRole(item, ($event.target as HTMLSelectElement).value as 'admin' | 'user')"><option value="user">普通用户</option><option value="admin">管理员</option></select><button class="icon-text compact" @click="setStatus(item, item.status === 'active' ? 'disabled' : 'active')"><UserCheck :size="15" />{{ item.status === 'active' ? '停用' : '启用' }}</button></div><span v-else class="muted">受保护</span></td>
    </tr><tr v-if="!loading && !items.length"><td colspan="6" class="empty">暂无账号</td></tr>
  </tbody></table></div></section>
</div></template>
