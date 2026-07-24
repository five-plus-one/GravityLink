import { getAccessToken } from './auth';

export interface LinkItem {
  ID: number;
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
    }>;
  };
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
  ID: number;
  Template: 'liveqr' | 'redirect_notice' | 'custom';
  Title: string;
  DomainID: number;
}

export interface LandingPageListData {
  items: LandingPageItem[];
  total: number;
}

export interface CreateLandingPagePayload {
  template: 'liveqr' | 'redirect_notice' | 'custom';
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

export interface AuthConfigStatus {
  auth_disabled: boolean;
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

export async function listLinks(): Promise<LinkListData> {
  return request<LinkListData>('/api/v1/links');
}

export async function createLink(payload: CreateLinkPayload): Promise<LinkItem> {
  return request<LinkItem>('/api/v1/links', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
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

export async function getSummaryStats(linkId: number): Promise<SummaryStats> {
  return request<SummaryStats>(`/api/v1/stats/${linkId}/summary`);
}

export async function getDailyStats(linkId: number): Promise<DailyPoint[]> {
  return request<DailyPoint[]>(`/api/v1/stats/${linkId}/daily`);
}

export async function getHourlyStats(linkId: number): Promise<HourlyPoint[]> {
  return request<HourlyPoint[]>(`/api/v1/stats/${linkId}/hourly`);
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

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Content-Type', 'application/json');

  const token = getAccessToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const response = await fetch(path, {
    ...init,
    headers,
  });
  const body = (await response.json()) as ApiResponse<T>;
  if (!response.ok || body.code !== 0) {
    throw new Error(body.message || 'request failed');
  }
  return body.data;
}
