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
    disabled: boolean;
    issuer: string;
    app_id: string;
    audience: string;
    jwks_url: string;
    scopes: string;
    admin_base_url: string;
    allowed_roles: string[];
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
    disabled: boolean;
    issuer: string;
    app_id: string;
    audience: string;
    jwks_url: string;
    scopes: string;
    admin_base_url: string;
    allowed_roles: string[];
  };
}

export async function loadSetupStatus(): Promise<SetupStatus> {
  return setupRequest<SetupStatus>('/api/setup/status');
}

export async function testSetupDatabase(payload: SetupPayload): Promise<{ mysql: boolean; redis: boolean; schema_ready: boolean }> {
  return setupRequest('/api/setup/test-database', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function completeSetup(payload: SetupPayload): Promise<{ initialized: boolean; auth_disabled: boolean }> {
  return setupRequest('/api/setup/complete', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

async function setupRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init.headers },
  });
  const body = await response.json();
  if (!response.ok || body.code !== 0) {
    throw new Error(body.message || '初始化请求失败');
  }
  return body.data as T;
}
