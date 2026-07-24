export interface AuthConfig {
  auth_disabled: boolean;
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
  subject: string;
  email: string;
  username: string;
  role: string;
}

const accessTokenKey = 'gravitylink_access_token';
const idTokenKey = 'gravitylink_id_token';
const verifierKey = 'gravitylink_pkce_verifier';
const stateKey = 'gravitylink_oidc_state';

export function getAccessToken(): string {
  return localStorage.getItem(accessTokenKey) ?? '';
}

export function clearSession(): void {
  localStorage.removeItem(accessTokenKey);
  localStorage.removeItem(idTokenKey);
  sessionStorage.removeItem(verifierKey);
  sessionStorage.removeItem(stateKey);
}

export async function loadAuthConfig(): Promise<AuthConfig> {
  const response = await fetch('/api/v1/auth/config');
  const body = await response.json();
  if (!response.ok || body.code !== 0) {
    throw new Error(body.message || '加载登录配置失败');
  }
  return body.data as AuthConfig;
}

export function currentUser(config: AuthConfig): AuthUser | null {
  if (config.auth_disabled) {
    return { subject: 'dev', email: '', username: 'dev', role: 'admin' };
  }

  const token = getAccessToken();
  if (!token) {
    return null;
  }

  const payload = decodeJWT(token);
  if (!payload) {
    return null;
  }

  const roles = Array.isArray(payload.roles) ? payload.roles : [];
  const role = typeof payload.role === 'string' ? payload.role : roles.includes('admin') ? 'admin' : 'user';
  return {
    subject: String(payload.sub ?? ''),
    email: String(payload.email ?? ''),
    username: String(payload.username ?? payload.name ?? payload.email ?? payload.sub ?? 'user'),
    role,
  };
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
  if (config.audience) {
    params.set('resource', config.audience);
  }

  window.location.href = `${config.authorization_endpoint}?${params.toString()}`;
}

export async function handleCallback(config: AuthConfig): Promise<boolean> {
  const url = new URL(window.location.href);
  if (url.pathname !== '/auth/callback') {
    return false;
  }

  const code = url.searchParams.get('code');
  const state = url.searchParams.get('state');
  const expectedState = sessionStorage.getItem(stateKey);
  const verifier = sessionStorage.getItem(verifierKey);
  if (!code || !state || state !== expectedState || !verifier) {
    throw new Error('登录回调校验失败');
  }

  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    client_id: config.client_id,
    code,
    redirect_uri: config.redirect_uri,
    code_verifier: verifier,
  });
  if (config.audience) {
    body.set('resource', config.audience);
  }

  const response = await fetch(config.token_endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  });
  const tokenBody = await response.json();
  if (!response.ok || !tokenBody.access_token) {
    throw new Error(tokenBody.error_description || tokenBody.error || '登录换取 token 失败');
  }

  localStorage.setItem(accessTokenKey, tokenBody.access_token);
  if (tokenBody.id_token) {
    localStorage.setItem(idTokenKey, tokenBody.id_token);
  }
  sessionStorage.removeItem(verifierKey);
  sessionStorage.removeItem(stateKey);
  window.history.replaceState({}, '', '/');
  return true;
}

export function logout(config: AuthConfig): void {
  const idToken = localStorage.getItem(idTokenKey);
  clearSession();
  if (config.logout_endpoint && config.client_id) {
    const params = new URLSearchParams({ client_id: config.client_id, post_logout_redirect_uri: window.location.origin });
    if (idToken) {
      params.set('id_token_hint', idToken);
    }
    window.location.href = `${config.logout_endpoint}?${params.toString()}`;
  }
}

function decodeJWT(token: string): Record<string, unknown> | null {
  const [, payload] = token.split('.');
  if (!payload) {
    return null;
  }
  try {
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/');
    const decoded = atob(normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '='));
    return JSON.parse(decoded) as Record<string, unknown>;
  } catch {
    return null;
  }
}

function randomString(length: number): string {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return base64URL(bytes).slice(0, length);
}

async function codeChallenge(verifier: string): Promise<string> {
  const data = new TextEncoder().encode(verifier);
  const digest = await crypto.subtle.digest('SHA-256', data);
  return base64URL(new Uint8Array(digest));
}

function base64URL(bytes: Uint8Array): string {
  let binary = '';
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}
