import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { LogOut } from 'lucide-react';

export default function Header() {
  const { username, logout } = useAuth();

  return (
    <header className="flex h-14 items-center justify-end border-b px-6">
      <div className="flex items-center gap-4">
        <span className="text-sm text-muted-foreground">{username}</span>
        <Button variant="ghost" size="sm" onClick={logout}>
          <LogOut className="mr-2 h-4 w-4" />
          Logout
        </Button>
      </div>
    </header>
  );
}
