import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { dashboardApi, type DashboardBuild } from '@/api/dashboard';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Package, Hammer, KeyRound, Activity } from 'lucide-react';
import BuildStatusBadge from '@/components/builds/BuildStatusBadge';

export default function DashboardPage() {
  const navigate = useNavigate();
  const { t } = useTranslation('dashboard');
  const { t: tc } = useTranslation('common');
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard'],
    queryFn: dashboardApi.get,
    refetchInterval: 10_000,
  });

  if (isLoading || !data) {
    return <p className="text-muted-foreground">{tc('loading')}</p>;
  }

  const stats = [
    { label: t('stats.products'), value: data.product_count, icon: Package, color: 'text-blue-500' },
    { label: t('stats.totalBuilds'), value: data.build_count, icon: Hammer, color: 'text-green-500' },
    { label: t('stats.gpgKeys'), value: data.gpg_key_count, icon: KeyRound, color: 'text-purple-500' },
    { label: t('stats.activeBuilds'), value: data.active_builds, icon: Activity, color: 'text-orange-500' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">{t('title')}</h1>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((s) => (
          <Card key={s.label}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{s.label}</CardTitle>
              <s.icon className={`h-4 w-4 ${s.color}`} />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{s.value}</div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t('recentBuilds')}</CardTitle>
          </CardHeader>
          <CardContent>
            {data.recent_builds.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('noBuilds')}</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('table.product')}</TableHead>
                    <TableHead>{t('table.version')}</TableHead>
                    <TableHead>{t('table.status')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.recent_builds.map((b: DashboardBuild) => (
                    <TableRow
                      key={b.id}
                      className="cursor-pointer"
                      onClick={() => navigate(`/builds/${b.id}`)}
                    >
                      <TableCell className="text-sm">{b.product_display_name}</TableCell>
                      <TableCell>
                        <Badge variant="secondary">{b.version}</Badge>
                      </TableCell>
                      <TableCell>
                        <BuildStatusBadge status={b.status} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t('products')}</CardTitle>
          </CardHeader>
          <CardContent>
            {data.product_summary.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('noProducts')}</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('table.name')}</TableHead>
                    <TableHead>{t('table.latest')}</TableHead>
                    <TableHead>{t('table.status')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.product_summary.map((p) => (
                    <TableRow
                      key={p.id}
                      className="cursor-pointer"
                      onClick={() => navigate(`/products/${p.id}`)}
                    >
                      <TableCell className="text-sm">{p.display_name}</TableCell>
                      <TableCell>
                        {p.latest_version ? (
                          <Badge variant="secondary">{p.latest_version}</Badge>
                        ) : (
                          <span className="text-sm text-muted-foreground">-</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <Badge variant={p.enabled ? 'default' : 'secondary'}>
                          {p.enabled ? tc('active') : tc('disabled')}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
