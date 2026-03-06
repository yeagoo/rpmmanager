import { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { toast } from 'sonner';
import PowCaptcha from '@/components/pow/PowCaptcha';
import LanguageSwitcher from '@/components/LanguageSwitcher';

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const { t } = useTranslation('login');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [altchaPayload, setAltchaPayload] = useState<string | null>(null);
  const [captchaKey, setCaptchaKey] = useState(0);

  const handleSolved = useCallback((payload: string) => {
    setAltchaPayload(payload);
  }, []);

  const handleCaptchaError = useCallback(() => {
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
      toast.error(t('loginFailed'));
      setAltchaPayload(null);
      setCaptchaKey(k => k + 1);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="relative flex min-h-screen items-center justify-center bg-background">
      <div className="absolute top-4 right-4">
        <LanguageSwitcher />
      </div>
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">{t('title')}</CardTitle>
          <CardDescription>{t('description')}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">{t('username')}</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder={t('usernamePlaceholder')}
                required
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">{t('password')}</Label>
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
              {loading ? t('signingIn') : t('signIn')}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
