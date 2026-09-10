import { clearSession, getAccessToken } from './auth';

export interface LinkItem {
  ID: number;
  EntryDomainID: number;
  PublicURL?: string;
  Code: string;
  Type: string;
  TargetURL: string | null;
  Title: string | null;
  Status: string;
  CreatedAt: string;
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface LinkListData {
  items: LinkItem[];
  total: number;
}

export interface CreateLinkPayload {
  type: 'short' | 'channel' | 'liveqr';
  code?: string;
  entry_domain_id: number;
  target_url: string;
  title?: string;
  access_rule?: 'none' | 'wechat' | 'ios' | 'android' | 'mobile' | 'pc';
  online_schedule?: string;
  expire_at?: string | null;
  channel?: {
    utm_source?: string;
    utm_medium?: string;
    utm_campaign?: string;
    utm_term?: string;
    utm_content?: string;
  };
  landing_domain_id?: number;
  landing_page_id?: number;
  strategy?: {
    mode: 'round_robin' | 'weighted';
    targets: Array<{
      label?: string;
      target_url: string;
      weight?: number;
      scan_limit?: number;
      wx_remark?: string;
    }>;
  };
}

export interface UpdateLinkPayload {
  target_url?: string;
  title?: string;
  expire_at?: string | null;
  status?: 'active' | 'disabled';
  online_schedule?: string;
  code?: string;
  entry_domain_id?: number;
}

export interface DomainItem {
  ID: number;
  Host: string;
  Type: 'entry' | 'transit' | 'landing';
  Scheme: 'http' | 'https';
  Remark: string | null;
  Status: string;
}

export interface DomainListData {
  items: DomainItem[];
  total: number;
}

export interface CreateDomainPayload {
  host: string;
  type: 'entry' | 'transit' | 'landing';
  scheme: 'http' | 'https';
  remark?: string;
}

export interface LandingPageItem {
  Content?: Record<string,unknown>;
  ID: number;
  Template: 'liveqr' | 'redirect_notice' | 'custom' | 'kf' | 'kami';
  Title: string;
  DomainID: number;
}

export interface LandingPageListData {
  items: LandingPageItem[];
  total: number;
}

export interface CreateLandingPagePayload {
  template: 'liveqr' | 'redirect_notice' | 'custom' | 'kf' | 'kami';
  title: string;
  domain_id: number;
  content: Record<string, unknown>;
}

export interface SummaryStats {
  total_pv: number;
  total_uv: number;
  today_pv: number;
  today_uv: number;
  yesterday_pv: number;
}

export interface DailyPoint {
  date: string;
  pv: number;
  uv: number;
}

export interface HourlyPoint {
  hour: number;
  pv: number;
}

export interface LabelValue {
  label: string;
  value: number;
}

export interface DeviceStats {
  device: LabelValue[];
  os: LabelValue[];
  browser: LabelValue[];
}

export interface AuthConfigStatus {
  issuer: string;
  client_id: string;
  audience: string;
  redirect_uri: string;
  allowed_roles: string[];
}

export interface ConfigListData {
  configs: Record<string, string>;
  auth: AuthConfigStatus;
}

export interface UserItem {
  id: number;
  auth_source: 'logto' | 'local';
  username: string;
  email: string | null;
  role: 'super_admin' | 'admin' | 'user';
  status: 'pending' | 'active' | 'disabled';
  last_login_at: string | null;
  created_at: string;
}

export async function listLinks(): Promise<LinkListData> {
  return request<LinkListData>('/api/v1/links');
}

export async function getLink(id: number): Promise<LinkItem> {
  return request<LinkItem>(`/api/v1/links/${id}`);
}

export async function createLink(payload: CreateLinkPayload): Promise<LinkItem> {
  return request<LinkItem>('/api/v1/links', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function updateLink(id: number, payload: UpdateLinkPayload): Promise<LinkItem> {
  return request<LinkItem>(`/api/v1/links/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export async function deleteLink(id: number): Promise<void> {
  await request<{ deleted: boolean }>(`/api/v1/links/${id}`, { method: 'DELETE' });
}

export async function listDomains(): Promise<DomainListData> {
  return request<DomainListData>('/api/admin/domains');
}

export async function createDomain(payload: CreateDomainPayload): Promise<DomainItem> {
  return request<DomainItem>('/api/admin/domains', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function deleteDomain(id: number): Promise<void> {
  await request<{ deleted: boolean }>(`/api/admin/domains/${id}`, {
    method: 'DELETE',
  });
}

export async function listLandingPages(): Promise<LandingPageListData> {
  return request<LandingPageListData>('/api/v1/landing-pages');
}

export async function createLandingPage(payload: CreateLandingPagePayload): Promise<LandingPageItem> {
  return request<LandingPageItem>('/api/v1/landing-pages', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function deleteLandingPage(id: number): Promise<void> {
  await request<{ deleted: boolean }>(`/api/v1/landing-pages/${id}`, { method: 'DELETE' });
}

export async function previewLandingDraft(payload: CreateLandingPagePayload): Promise<string> {
  const result = await request<{ html: string }>('/api/v1/landing-pages/preview-draft', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return result.html;
}

export async function getSummaryStats(linkId: number): Promise<SummaryStats> {
  return request<SummaryStats>(`/api/v1/stats/${linkId}/summary`);
}

export async function getOverviewSummary(): Promise<SummaryStats> {
  return request<SummaryStats>('/api/v1/stats/overview/summary');
}

export async function getOverviewDaily(query = ''): Promise<DailyPoint[]> {
  return request<DailyPoint[]>(`/api/v1/stats/overview/daily${query}`);
}

export async function getOverviewHourly(): Promise<HourlyPoint[]> {
  return request<HourlyPoint[]>('/api/v1/stats/overview/hourly');
}

export async function getDailyStats(linkId: number, query = ''): Promise<DailyPoint[]> {
  return request<DailyPoint[]>(`/api/v1/stats/${linkId}/daily${query}`);
}

export async function getHourlyStats(linkId: number): Promise<HourlyPoint[]> {
  return request<HourlyPoint[]>(`/api/v1/stats/${linkId}/hourly`);
}

export async function getGeoStats(linkId: number, query = ''): Promise<LabelValue[]> {
  return request<LabelValue[]>(`/api/v1/stats/${linkId}/geo${query}`);
}

export async function getDeviceStats(linkId: number, query = ''): Promise<DeviceStats> {
  return request<DeviceStats>(`/api/v1/stats/${linkId}/device${query}`);
}

export interface VisitorLogItem {
  id: number;
  link_id: number;
  link_code: string;
  link_title: string;
  visited_at: string;
  ip: string;
  country: string;
  province: string;
  city: string;
  device: string;
  os: string;
  browser: string;
  referer: string;
  source_app: string;
}

export interface VisitorLogResult {
  items: VisitorLogItem[];
  total: number;
}

export async function listVisitors(params: {
  link_id?: number;
  start?: string;
  end?: string;
  keyword?: string;
  limit?: number;
  offset?: number;
} = {}): Promise<VisitorLogResult> {
  const qs = new URLSearchParams();
  if (params.link_id) qs.set('link_id', String(params.link_id));
  if (params.start) qs.set('start', params.start);
  if (params.end) qs.set('end', params.end);
  if (params.keyword) qs.set('keyword', params.keyword);
  if (params.limit) qs.set('limit', String(params.limit));
  if (params.offset) qs.set('offset', String(params.offset));
  const query = qs.toString();
  return request<VisitorLogResult>(`/api/v1/stats/visitors${query ? '?' + query : ''}`);
}

export async function resetLinkStats(linkId: number): Promise<void> {
  await request<{ reset: boolean }>(`/api/v1/stats/${linkId}/reset`, { method: 'POST' });
}

export async function getSystemConfigs(): Promise<ConfigListData> {
  return request<ConfigListData>('/api/admin/configs');
}

export async function updateSystemConfigs(configs: Record<string, string>): Promise<ConfigListData> {
  return request<ConfigListData>('/api/admin/configs', {
    method: 'PUT',
    body: JSON.stringify({ configs }),
  });
}

export async function listUsers(): Promise<{ items: UserItem[]; total: number }> {
  return request('/api/admin/users');
}

export async function updateUserRole(id: number, role: 'admin' | 'user'): Promise<{ updated: boolean }> {
  return request(`/api/admin/users/${id}/role`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
}

export async function updateUserStatus(id: number, status: 'active' | 'disabled'): Promise<{ updated: boolean }> {
  return request(`/api/admin/users/${id}/status`, {
    method: 'PUT',
    body: JSON.stringify({ status }),
  });
}

export async function resetSystem(confirmation: string, password = ''): Promise<{ setup_required: boolean }> {
  return request('/api/admin/system/reset', {
    method: 'POST',
    body: JSON.stringify({ confirmation, password }),
  });
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (!(init.body instanceof FormData)) headers.set('Content-Type', 'application/json');

  const token = getAccessToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const response = await fetch(path, {
    ...init,
    headers,
    credentials: 'include',
  });
  const body = (await response.json()) as ApiResponse<T>;
  if (!response.ok || body.code !== 0) {
    // 401：token 失效，清除会话并跳登录
    if (response.status === 401 && !path.includes('/auth/')) {
      clearSession();
      if (!window.location.pathname.startsWith('/login') && !window.location.pathname.startsWith('/setup')) {
        window.location.href = '/login';
      }
    }
    throw new Error(body.message || 'request failed');
  }
  return body.data;
}
