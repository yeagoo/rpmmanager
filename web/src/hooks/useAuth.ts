import { createContext, useContext, useState, useCallback, useEffect } from 'react';
import { apiClient } from '@/api/client';

interface AuthContextType {
  username: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
}

export const AuthContext = createContext<AuthContextType>({
  username: null,
  isAuthenticated: false,
  isLoading: true,
  login: async () => {},
  logout: () => {},
});

export function useAuth() {
  return useContext(AuthContext);
}

export function useAuthProvider(): AuthContextType {
  const [username, setUsername] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const token = apiClient.getToken();
    if (token) {
      apiClient.get<{ username: string }>('/auth/me')
        .then((data) => setUsername(data.username))
        .catch(() => {
          apiClient.setToken(null);
          setUsername(null);
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }

    // Listen for 401 events from apiClient
    const handleUnauthorized = () => {
      setUsername(null);
    };
    window.addEventListener('auth:unauthorized', handleUnauthorized);
    return () => window.removeEventListener('auth:unauthorized', handleUnauthorized);
  }, []);

  const login = useCallback(async (user: string, password: string) => {
    const data = await apiClient.post<{ token: string; username: string }>('/auth/login', {
      username: user,
      password,
    });
    apiClient.setToken(data.token);
    setUsername(data.username);
  }, []);

  const logout = useCallback(() => {
    apiClient.setToken(null);
    setUsername(null);
  }, []);

  return {
    username,
    isAuthenticated: !!username,
    isLoading,
    login,
    logout,
  };
}
