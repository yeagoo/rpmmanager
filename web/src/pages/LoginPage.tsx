import { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { toast } from 'sonner';
import PowCaptcha from '@/components/pow/PowCaptcha';

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [altchaPayload, setAltchaPayload] = useState<string | null>(null);
  const [captchaKey, setCaptchaKey] = useState(0);

  const handleSolved = useCallback((payload: string) => {
    setAltchaPayload(payload);
  }, []);

  const handleCaptchaError = useCallback(() => {
    // Auto-retry after 2 seconds on challenge fetch failure
    setTimeout(() => setCaptchaKey(k => k + 1), 2000);
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!altchaPayload) return;

    setLoading(true);
    try {
      await login(username, password, altchaPayload);
      navigate('/');
    } catch {
      toast.error('Login failed. Check your credentials.');
      // Reset captcha for a fresh challenge
      setAltchaPayload(null);
      setCaptchaKey(k => k + 1);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">RPM Manager</CardTitle>
          <CardDescription>Sign in to manage your RPM repositories</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                required
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <PowCaptcha key={captchaKey} onSolved={handleSolved} onError={handleCaptchaError} />
            <Button type="submit" className="w-full" disabled={loading || !altchaPayload}>
              {loading ? 'Signing in...' : 'Sign In'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
