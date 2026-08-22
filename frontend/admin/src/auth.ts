export interface AuthConfig {
  mode: 'logto' | 'local';
  local_enabled: boolean;
  issuer: string;
  client_id: string;
  audience: string;
  scopes: string;
  authorization_endpoint: string;
  token_endpoint: string;
  logout_endpoint: string;
  redirect_uri: string;
  allowed_roles: string[];
}

export interface AuthUser {
  id: number;
  subject: string;
  email: string;
  username: string;
  role: 'super_admin' | 'admin' | 'user';
  status: 'pending' | 'active' | 'disabled';
  auth_source: 'logto' | 'local' | 'development';
}

const accessTokenKey = 'gravitylink_access_token';
const idTokenKey = 'gravitylink_id_token';
const verifierKey = 'gravitylink_pkce_verifier';
const stateKey = 'gravitylink_oidc_state';
const oidcConfigKey = 'gravitylink_setup_oidc';

export function getAccessToken(): string {
  return localStorage.getItem(accessTokenKey) ?? '';
}

export function clearSession(): void {
  localStorage.removeItem(accessTokenKey);
  localStorage.removeItem(idTokenKey);
  sessionStorage.removeItem(verifierKey);
  sessionStorage.removeItem(stateKey);
  sessionStorage.removeItem(oidcConfigKey);
}

export async function loadAuthConfig(): Promise<AuthConfig> {
  return authRequest<AuthConfig>('/api/v1/auth/config');
}

export async function loadCurrentUser(): Promise<AuthUser> {
  return authRequest<AuthUser>('/api/v1/auth/me', {}, true);
}

export async function loginLocal(username: string, password: string): Promise<AuthUser> {
  return authRequest<AuthUser>('/api/v1/auth/local/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  });
}

export async function logoutSession(): Promise<void> {
  try {
    await authRequest('/api/v1/auth/logout', { method: 'POST' }, true);
  } finally {
    clearSession();
  }
}

export function rememberSetupOIDC(config: AuthConfig): void {
  sessionStorage.setItem(oidcConfigKey, JSON.stringify(config));
}

export function recallSetupOIDC(): AuthConfig | null {
  const value = sessionStorage.getItem(oidcConfigKey);
  if (!value) return null;
  try {
    return JSON.parse(value) as AuthConfig;
  } catch {
    return null;
  }
}

export async function login(config: AuthConfig): Promise<void> {
  if (!config.authorization_endpoint || !config.client_id || !config.redirect_uri) {
    throw new Error('Logto 登录配置不完整');
  }

  const verifier = randomString(64);
  const state = randomString(32);
  const challenge = await codeChallenge(verifier);
  sessionStorage.setItem(verifierKey, verifier);
  sessionStorage.setItem(stateKey, state);

  const params = new URLSearchParams({
    client_id: config.client_id,
    redirect_uri: config.redirect_uri,
    response_type: 'code',
    scope: config.scopes || 'openid profile email',
    code_challenge: challenge,
    code_challenge_method: 'S256',
    state,
  });
  if (config.audience) params.set('resource', config.audience);
  window.location.assign(`${config.authorization_endpoint}?${params.toString()}`);
}

export async function handleCallback(
  config: AuthConfig,
  callbackPath = '/auth/callback',
  returnPath = '/',
): Promise<boolean> {
  const url = new URL(window.location.href);
  if (url.pathname !== callbackPath) return false;

  const code = url.searchParams.get('code');
  const state = url.searchParams.get('state');
  const expectedState = sessionStorage.getItem(stateKey);
  const verifier = sessionStorage.getItem(verifierKey);
  if (!code || !state || state !== expectedState || !verifier) {
    throw new Error('登录回调校验失败，请重新发起登录');
  }

  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    client_id: config.client_id,
    code,
    redirect_uri: config.redirect_uri,
    code_verifier: verifier,
  });
  if (config.audience) body.set('resource', config.audience);

  const response = await fetch(config.token_endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  });
  const tokenBody = await response.json();
  if (!response.ok || !tokenBody.access_token) {
    throw new Error(tokenBody.error_description || tokenBody.error || '无法从 Logto 获取访问令牌');
  }

  localStorage.setItem(accessTokenKey, tokenBody.access_token);
  if (tokenBody.id_token) localStorage.setItem(idTokenKey, tokenBody.id_token);
  sessionStorage.removeItem(verifierKey);
  sessionStorage.removeItem(stateKey);
  window.history.replaceState({}, '', returnPath);
  return true;
}

export function logout(config: AuthConfig): void {
  const idToken = localStorage.getItem(idTokenKey);
  clearSession();
  if (config.logout_endpoint && config.client_id) {
    const params = new URLSearchParams({
      client_id: config.client_id,
      post_logout_redirect_uri: window.location.origin,
    });
    if (idToken) params.set('id_token_hint', idToken);
    window.location.assign(`${config.logout_endpoint}?${params.toString()}`);
    return;
  }
  window.location.replace('/');
}

async function authRequest<T>(path: string, init: RequestInit = {}, authenticated = false): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Content-Type', 'application/json');
  if (authenticated && getAccessToken()) headers.set('Authorization', `Bearer ${getAccessToken()}`);
  const response = await fetch(path, { ...init, headers, credentials: 'include' });
  const body = await response.json();
  if (!response.ok || body.code !== 0) throw new Error(body.message || '认证请求失败');
  return body.data as T;
}

function randomString(length: number): string {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return base64URL(bytes).slice(0, length);
}

async function codeChallenge(verifier: string): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier));
  return base64URL(new Uint8Array(digest));
}

function base64URL(bytes: Uint8Array): string {
  let binary = '';
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}
