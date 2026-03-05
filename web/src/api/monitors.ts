import { apiClient } from './client';

export interface Monitor {
  id: number;
  product_id: number;
  enabled: boolean;
  check_interval: string;
  auto_build: boolean;
  last_checked_at: string | null;
  last_known_version: string;
  last_error: string;
  created_at: string;
  updated_at: string;
  product_name: string;
  product_display_name: string;
  source_type: string;
  source_github_owner: string;
  source_github_repo: string;
}

export interface UpdateMonitorRequest {
  enabled?: boolean;
  check_interval?: string;
  auto_build?: boolean;
}

export const monitorsApi = {
  list: () => apiClient.get<Monitor[]>('/monitors'),
  get: (productId: number) => apiClient.get<Monitor>(`/monitors/${productId}`),
  update: (productId: number, req: UpdateMonitorRequest) =>
    apiClient.put(`/monitors/${productId}`, req),
  checkNow: (productId: number) =>
    apiClient.post<{ version: string }>(`/monitors/${productId}/check`),
};
