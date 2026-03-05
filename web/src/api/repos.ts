import { apiClient } from './client';

export interface RepoInfo {
  product: string;
  path: string;
  total_size: number;
  file_count: number;
  dir_count: number;
  rpm_count: number;
  has_repomd: boolean;
}

export interface RepoEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  mod_time: string;
  items?: RepoEntry[];
}

export const reposApi = {
  list: () => apiClient.get<RepoInfo[]>('/repos'),
  getTree: (product: string, path?: string, depth?: number) => {
    const params = new URLSearchParams();
    if (path) params.set('path', path);
    if (depth) params.set('depth', String(depth));
    const qs = params.toString();
    return apiClient.get<RepoEntry[]>(`/repos/${product}/tree${qs ? '?' + qs : ''}`);
  },
  getFileContent: (product: string, path: string) =>
    apiClient.get<string>(`/repos/${product}/file?path=${encodeURIComponent(path)}`),
  listRollbacks: (product: string) =>
    apiClient.get<string[]>(`/repos/${product}/rollbacks`),
  rollback: (product: string, snapshot: string) =>
    apiClient.post(`/repos/${product}/rollback`, { snapshot }),
};
