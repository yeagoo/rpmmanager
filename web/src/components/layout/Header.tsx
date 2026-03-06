import { useTranslation } from 'react-i18next';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { LogOut } from 'lucide-react';
import LanguageSwitcher from '@/components/LanguageSwitcher';

export default function Header() {
  const { username, logout } = useAuth();
  const { t } = useTranslation('layout');

  return (
    <header className="flex h-14 items-center justify-end border-b px-6">
      <div className="flex items-center gap-4">
        <LanguageSwitcher />
        <span className="text-sm text-muted-foreground">{username}</span>
        <Button variant="ghost" size="sm" onClick={logout}>
          <LogOut className="mr-2 h-4 w-4" />
          {t('logout')}
        </Button>
      </div>
    </header>
  );
}
