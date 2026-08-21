import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import {
  clearSession,
  loadAuthConfig,
  loadCurrentUser,
  loginLocal,
  logout,
  logoutSession,
  type AuthConfig,
  type AuthUser,
} from '../auth';

/**
 * 全局认证状态：当前用户、AuthConfig、登录/登出动作。
 * 整个应用共享一份，避免在 App/UsersView/SettingsView 重复请求。
 */
export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null);
  const config = ref<AuthConfig | null>(null);
  const loaded = ref(false);

  const isLoggedIn = computed(() => Boolean(user.value));
  const isSuperAdmin = computed(() => user.value?.role === 'super_admin');

  async function loadConfig(): Promise<AuthConfig> {
    if (config.value) return config.value;
    config.value = await loadAuthConfig();
    return config.value;
  }

  async function fetchUser(): Promise<AuthUser | null> {
    try {
      user.value = await loadCurrentUser();
    } catch {
      user.value = null;
    }
    loaded.value = true;
    return user.value;
  }

  async function signInLocal(username: string, password: string): Promise<AuthUser> {
    user.value = await loginLocal(username, password);
    loaded.value = true;
    return user.value;
  }

  async function signOut(): Promise<void> {
    const current = config.value;
    if (current?.mode === 'logto') {
      logout(current);
      return;
    }
    try {
      await logoutSession();
    } finally {
      clearSession();
      user.value = null;
    }
  }

  function reset(): void {
    user.value = null;
    config.value = null;
    loaded.value = false;
  }

  return { user, config, loaded, isLoggedIn, isSuperAdmin, loadConfig, fetchUser, signInLocal, signOut, reset };
});
