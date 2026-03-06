import { apiClient } from './client';

export interface Product {
  id: number;
  name: string;
  display_name: string;
  description: string;
  source_type: 'github' | 'url';
  source_github_owner: string;
  source_github_repo: string;
  source_url_template: string;
  nfpm_config: string;
  target_distros: string[];
  architectures: string[];
  product_lines?: string;
  maintainer: string;
  vendor: string;
  homepage: string;
  license: string;
  script_postinstall: string;
  script_preremove: string;
  systemd_service: string;
  default_config: string;
  default_config_path: string;
  extra_files: string;
  gpg_key_id: number | null;
  base_url: string;
  sm2_enabled: boolean;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  latest_version?: string;
  last_build_at?: string;
}

export interface DistroInfo {
  product_lines: { id: string; path: string; tag: string; compression: string }[];
  distro_groups: Record<string, { distro: string; version: string }[]>;
  all_distros: string[];
}

export interface RepoRPMResult {
  filename: string;
  file_path: string;
  size: number;
  download_url: string;
}

export const productsApi = {
  list: () => apiClient.get<Product[]>('/products'),
  get: (id: number) => apiClient.get<Product>(`/products/${id}`),
  create: (data: Partial<Product>) => apiClient.post<Product>('/products', data),
  update: (id: number, data: Partial<Product>) => apiClient.put<Product>(`/products/${id}`, data),
  delete: (id: number) => apiClient.delete(`/products/${id}`),
  duplicate: (id: number) => apiClient.post<Product>(`/products/${id}/duplicate`),
  getDistros: () => apiClient.get<DistroInfo>('/distros'),
  generateRepoRPM: (id: number, data?: { distros?: string[]; version?: string }) =>
    apiClient.post<RepoRPMResult>(`/products/${id}/repo-rpm`, data),
  getRepoRPM: (id: number) => apiClient.get<RepoRPMResult>(`/products/${id}/repo-rpm`),
};
