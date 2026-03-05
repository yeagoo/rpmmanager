import { NavLink } from 'react-router-dom';
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
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/products', label: 'Products', icon: Package },
  { to: '/builds', label: 'Builds', icon: Hammer },
  { to: '/gpg-keys', label: 'GPG Keys', icon: KeyRound },
  { to: '/repos', label: 'Repositories', icon: FolderTree },
  { to: '/monitors', label: 'Monitors', icon: Radar },
  { to: '/settings', label: 'Settings', icon: Settings },
];

export default function AppSidebar() {
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
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
