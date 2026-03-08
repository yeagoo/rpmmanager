import { apiClient } from './client';

export interface GPGKey {
  id: number;
  name: string;
  fingerprint: string;
  key_id: string;
  uid_name: string;
  uid_email: string;
  algorithm: string;
  key_length: number;
  created_date: string | null;
  expires_date: string | null;
  has_private: boolean;
  public_key_armor: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface GenerateKeyRequest {
  name: string;
  email: string;
  algorithm: string;
  key_length: number;
  expire: string;
}

export const gpgKeysApi = {
  list: () => apiClient.get<GPGKey[]>('/gpg-keys'),
  get: (id: number) => apiClient.get<GPGKey>(`/gpg-keys/${id}`),
  importKey: (keyData: string) => apiClient.post<GPGKey>('/gpg-keys/import', { key_data: keyData }),
  generate: (req: GenerateKeyRequest) => apiClient.post<GPGKey>('/gpg-keys/generate', req),
  delete: (id: number) => apiClient.delete(`/gpg-keys/${id}`),
  export: async (id: number): Promise<string> => {
    const resp = await fetch(`/api/gpg-keys/${id}/export`, {
      headers: { Authorization: `Bearer ${localStorage.getItem('token') || ''}` },
    });
    if (!resp.ok) throw new Error(`Export failed: HTTP ${resp.status}`);
    return resp.text();
  },
  setDefault: (id: number) => apiClient.post(`/gpg-keys/${id}/default`),
};
