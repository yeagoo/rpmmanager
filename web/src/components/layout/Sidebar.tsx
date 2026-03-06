import { NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  LayoutDashboard,
  Package,
  Hammer,
  KeyRound,
  FolderTree,
  Radar,
  Settings,
} from 'lucide-react';

const navItems = [
  { to: '/', labelKey: 'nav.dashboard', icon: LayoutDashboard },
  { to: '/products', labelKey: 'nav.products', icon: Package },
  { to: '/builds', labelKey: 'nav.builds', icon: Hammer },
  { to: '/gpg-keys', labelKey: 'nav.gpgKeys', icon: KeyRound },
  { to: '/repos', labelKey: 'nav.repos', icon: FolderTree },
  { to: '/monitors', labelKey: 'nav.monitors', icon: Radar },
  { to: '/settings', labelKey: 'nav.settings', icon: Settings },
];

export default function AppSidebar() {
  const { t } = useTranslation('layout');

  return (
    <aside className="flex h-screen w-60 flex-col border-r bg-sidebar">
      <div className="flex h-14 items-center border-b px-4">
        <span className="text-lg font-bold">RPM Manager</span>
      </div>
      <nav className="flex-1 space-y-1 p-2">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors ${
                isActive
                  ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
                  : 'text-sidebar-foreground hover:bg-sidebar-accent/50'
              }`
            }
          >
            <item.icon className="h-4 w-4" />
            {t(item.labelKey)}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
