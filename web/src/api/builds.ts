import { apiClient } from './client';

export interface Build {
  id: number;
  product_id: number;
  version: string;
  status: 'pending' | 'building' | 'signing' | 'publishing' | 'verifying' | 'success' | 'failed' | 'cancelled';
  current_stage: string;
  trigger_type: string;
  target_distros: string[];
  architectures: string[];
  rpm_count: number;
  symlink_count: number;
  error_message: string;
  log_file: string;
  started_at: string | null;
  finished_at: string | null;
  duration_seconds: number;
  created_at: string;
  product_name?: string;
  product_display_name?: string;
}

export const buildsApi = {
  list: (productId?: number, limit?: number) => {
    const params: Record<string, string> = {};
    if (productId) params.product_id = String(productId);
    if (limit) params.limit = String(limit);
    return apiClient.get<Build[]>('/builds', params);
  },
  get: (id: number) => apiClient.get<Build>(`/builds/${id}`),
  trigger: (productId: number, version: string) =>
    apiClient.post<Build>('/builds', { product_id: productId, version }),
  cancel: (id: number) => apiClient.post(`/builds/${id}/cancel`),
  getLog: async (id: number): Promise<string> => {
    const token = apiClient.getToken();
    const headers: Record<string, string> = {};
    if (token) headers['Authorization'] = `Bearer ${token}`;
    const res = await fetch(`/api/builds/${id}/log`, { headers });
    if (!res.ok) throw new Error('Failed to fetch log');
    return res.text();
  },
};
