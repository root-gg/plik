import type { ClientBinary, ServerConfig, UploadDescription, UserInfo } from '../types';

const apiBase = window.location.origin + window.location.pathname.replace(/\/$/, '');

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    headers: {
      'Content-Type': 'application/json',
      ...init.headers,
    },
    ...init,
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

export interface UploadOptions {
  oneShot?: boolean;
  removable?: boolean;
  stream?: boolean;
  password?: string;
  extend_ttl?: boolean;
  comments?: string;
  ttl?: number;
}

export interface TokenRecord {
  token: string;
  comment?: string;
}

export interface PaginatedResult<T> {
  results: T[];
  after?: string;
}

export const api = {
  base: apiBase,
  async getConfig(): Promise<ServerConfig> {
    return request<ServerConfig>(`${apiBase}/config`);
  },
  async getUser(): Promise<UserInfo> {
    return request<UserInfo>(`${apiBase}/me`);
  },
  async login(provider: 'local' | 'google' | 'ovh', login?: string, password?: string): Promise<unknown> {
    if (provider === 'local') {
      return request(`${apiBase}/auth/local/login`, {
        method: 'POST',
        body: JSON.stringify({ login, password }),
      });
    }
    return request(`${apiBase}/auth/${provider}/login`);
  },
  async logout(): Promise<void> {
    await request(`${apiBase}/auth/logout`);
  },
  async getVersion(): Promise<{ clients: ClientBinary[] }> {
    return request(`${apiBase}/version`);
  },
  async getUserTokens(limit = 50, cursor?: string): Promise<PaginatedResult<TokenRecord>> {
    const params = new URLSearchParams({ limit: limit.toString() });
    if (cursor) params.set('after', cursor);
    return request(`${apiBase}/me/token?${params.toString()}`);
  },
  async getUserUploads(limit = 50, cursor?: string): Promise<PaginatedResult<UploadDescription>> {
    const params = new URLSearchParams({ limit: limit.toString() });
    if (cursor) params.set('after', cursor);
    return request(`${apiBase}/me/uploads?${params.toString()}`);
  },
  async createToken(comment?: string): Promise<void> {
    await request(`${apiBase}/me/token`, { method: 'POST', body: JSON.stringify({ comment }) });
  },
  async revokeToken(token: string): Promise<void> {
    await request(`${apiBase}/me/token/${token}`, { method: 'DELETE' });
  },
  async deleteUploads(token?: string): Promise<void> {
    const params = token ? `?token=${encodeURIComponent(token)}` : '';
    await request(`${apiBase}/me/uploads${params}`, { method: 'DELETE' });
  },
  async createUpload(payload: UploadOptions): Promise<UploadDescription> {
    return request<UploadDescription>(`${apiBase}/upload`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },
  async uploadFile(upload: UploadDescription, file: File): Promise<unknown> {
    const mode = upload.stream ? 'stream' : 'file';
    const url = `${apiBase}/${mode}/${upload.id}`;
    const form = new FormData();
    form.append('file', file, (file as any).fileName || file.name);
    const response = await fetch(url, { method: 'POST', body: form });
    if (!response.ok) {
      const message = await response.text();
      throw new Error(message || 'Upload failed');
    }
    return response.json();
  },
  async removeUpload(upload: UploadDescription): Promise<void> {
    await request(`${apiBase}/upload/${upload.id}`, { method: 'DELETE' });
  },
  async getServerStats(): Promise<unknown> {
    return request(`${apiBase}/stats`);
  },
};
