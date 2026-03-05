import { apiClient } from './client';

export const settingsApi = {
  getAll: () => apiClient.get<Record<string, string>>('/settings'),
  update: (settings: Record<string, string>) =>
    apiClient.put('/settings', settings),
};
