import { apiClient } from './client';

export interface DashboardData {
  product_count: number;
  build_count: number;
  gpg_key_count: number;
  active_builds: number;
  recent_builds: DashboardBuild[];
  product_summary: ProductSummary[];
}

export interface DashboardBuild {
  id: number;
  product_display_name: string;
  version: string;
  status: string;
  created_at: string;
}

export interface ProductSummary {
  id: number;
  display_name: string;
  latest_version: string;
  last_build_status: string;
  enabled: boolean;
}

export const dashboardApi = {
  get: () => apiClient.get<DashboardData>('/dashboard'),
};
