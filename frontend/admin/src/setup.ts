import { getAccessToken, type AuthConfig } from './auth';

export interface SetupStatus {
  setup_required: boolean;
  reason: string;
  environment: string;
  database: {
    host: string;
    port: string;
    database: string;
    user: string;
    params: string;
    dsn_configured: boolean;
  };
  redis: {
    host: string;
    port: string;
    db: number;
    addr_configured: boolean;
  };
  auth: {
    mode: 'logto' | 'local';
    issuer: string;
    app_id: string;
    audience: string;
    jwks_url: string;
    scopes: string;
    admin_base_url: string;
    allowed_roles: string[];
    verified: boolean;
    owner_username: string;
    owner_email: string;
  };
}

export interface SetupPayload {
  database: {
    host: string;
    port: string;
    database: string;
    user: string;
    password: string;
    params: string;
    dsn: string;
  };
  redis: {
    host: string;
    port: string;
    addr: string;
    password: string;
    db: number;
  };
  auth: {
    mode: 'logto' | 'local';
    issuer: string;
    app_id: string;
    audience: string;
    jwks_url: string;
    scopes: string;
    admin_base_url: string;
    allowed_roles: string[];
    username?: string;
    email?: string;
    password?: string;
  };
  public_base_url?: string;
}

export interface LogtoCheckResult {
  issuer: string;
  authorization_endpoint: string;
  token_endpoint: string;
  jwks_uri: string;
  redirect_uri: string;
}

export async function loadSetupStatus(): Promise<SetupStatus> {
  return setupRequest<SetupStatus>('/api/setup/status');
}

export async function checkSetupLogto(payload: SetupPayload): Promise<LogtoCheckResult> {
  return setupRequest('/api/setup/auth/logto/check', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function claimSetupLogto(): Promise<{ verified: boolean; username: string; email: string }> {
  return setupRequest('/api/setup/auth/logto/claim', {
    method: 'POST',
    headers: { Authorization: `Bearer ${getAccessToken()}` },
  });
}

export async function configureLocalOwner(payload: SetupPayload): Promise<{ verified: boolean; username: string; email: string }> {
  return setupRequest('/api/setup/auth/local', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function testSetupDatabase(payload: SetupPayload): Promise<{ mysql: boolean; redis: boolean; schema_ready: boolean }> {
  return setupRequest('/api/setup/database/test', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function completeSetup(payload: SetupPayload): Promise<{ initialized: boolean }> {
  return setupRequest('/api/setup/complete', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function setupAuthConfig(payload: SetupPayload, result: LogtoCheckResult): AuthConfig {
  return {
    mode: 'logto',
    local_enabled: false,
    issuer: result.issuer,
    client_id: payload.auth.app_id,
    audience: payload.auth.audience,
    scopes: payload.auth.scopes,
    authorization_endpoint: result.authorization_endpoint,
    token_endpoint: result.token_endpoint,
    logout_endpoint: '',
    redirect_uri: result.redirect_uri,
    allowed_roles: payload.auth.allowed_roles,
  };
}

async function setupRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init.headers },
    credentials: 'include',
  });
  const body = await response.json();
  if (!response.ok || body.code !== 0) throw new Error(body.message || '初始化请求失败');
  return body.data as T;
}
