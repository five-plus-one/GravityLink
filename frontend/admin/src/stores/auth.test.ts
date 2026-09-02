import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useAuthStore } from './auth';
import type { AuthConfig, AuthUser } from '../auth';

vi.mock('../auth', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../auth')>();
  return {
    ...actual,
    loadAuthConfig: vi.fn(),
    loadCurrentUser: vi.fn(),
    loginLocal: vi.fn(),
    logoutSession: vi.fn(),
    logout: vi.fn(),
    clearSession: vi.fn(),
  };
});

import {
  clearSession,
  loadAuthConfig,
  loadCurrentUser,
  loginLocal,
  logout,
  logoutSession,
} from '../auth';

const mockLoadCurrentUser = vi.mocked(loadCurrentUser);
const mockLoadAuthConfig = vi.mocked(loadAuthConfig);
const mockLoginLocal = vi.mocked(loginLocal);
const mockLogoutSession = vi.mocked(logoutSession);
const mockLogout = vi.mocked(logout);
const mockClearSession = vi.mocked(clearSession);

const sampleUser: AuthUser = {
  id: 1,
  subject: 'sub-1',
  email: 'admin@example.com',
  username: 'admin',
  role: 'super_admin',
  status: 'active',
  auth_source: 'local',
};

const logtoConfig: AuthConfig = {
  mode: 'logto',
  local_enabled: false,
  issuer: 'https://issuer.example.com',
  client_id: 'client',
  audience: 'https://api.example.com',
  scopes: 'openid profile',
  authorization_endpoint: 'https://issuer.example.com/auth',
  token_endpoint: 'https://issuer.example.com/token',
  logout_endpoint: 'https://issuer.example.com/logout',
  redirect_uri: 'http://localhost:5173/auth/callback',
  allowed_roles: ['admin'],
};

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('useAuthStore', () => {
  it('初始状态未登录', () => {
    const store = useAuthStore();
    expect(store.isLoggedIn).toBe(false);
    expect(store.isSuperAdmin).toBe(false);
    expect(store.loaded).toBe(false);
  });

  it('fetchUser 成功后进入登录态', async () => {
    mockLoadCurrentUser.mockResolvedValue(sampleUser);
    const store = useAuthStore();

    await store.fetchUser();

    expect(store.user).toEqual(sampleUser);
    expect(store.loaded).toBe(true);
    expect(store.isLoggedIn).toBe(true);
    expect(store.isSuperAdmin).toBe(true);
  });

  it('fetchUser 失败时视为未登录且标记已加载', async () => {
    mockLoadCurrentUser.mockRejectedValue(new Error('unauthorized'));
    const store = useAuthStore();

    await store.fetchUser();

    expect(store.user).toBeNull();
    expect(store.loaded).toBe(true);
    expect(store.isLoggedIn).toBe(false);
  });

  it('loadConfig 有缓存时不重复请求', async () => {
    mockLoadAuthConfig.mockResolvedValue(logtoConfig);
    const store = useAuthStore();

    await store.loadConfig();
    await store.loadConfig();

    expect(mockLoadAuthConfig).toHaveBeenCalledTimes(1);
  });

  it('signInLocal 记录用户并进入登录态', async () => {
    mockLoginLocal.mockResolvedValue(sampleUser);
    const store = useAuthStore();

    await store.signInLocal('admin', 'password');

    expect(mockLoginLocal).toHaveBeenCalledWith('admin', 'password');
    expect(store.isLoggedIn).toBe(true);
  });

  it('logto 模式登出走 OIDC logout 并保留用户态', async () => {
    const store = useAuthStore();
    store.config = logtoConfig;
    store.user = sampleUser;

    await store.signOut();

    expect(mockLogout).toHaveBeenCalledWith(logtoConfig);
    expect(store.user).toEqual(sampleUser);
    expect(mockLogoutSession).not.toHaveBeenCalled();
  });

  it('local 模式登出清空会话与用户态', async () => {
    const store = useAuthStore();
    store.user = sampleUser;

    await store.signOut();

    expect(mockLogoutSession).toHaveBeenCalledTimes(1);
    expect(mockClearSession).toHaveBeenCalledTimes(1);
    expect(store.user).toBeNull();
    expect(store.isLoggedIn).toBe(false);
  });

  it('reset 清空全部状态', async () => {
    mockLoadCurrentUser.mockResolvedValue(sampleUser);
    const store = useAuthStore();
    await store.fetchUser();

    store.reset();

    expect(store.user).toBeNull();
    expect(store.config).toBeNull();
    expect(store.loaded).toBe(false);
  });
});
