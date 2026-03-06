import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { monitorsApi, type Monitor } from '@/api/monitors';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { RefreshCw, Power, PowerOff, Zap } from 'lucide-react';
import { toast } from 'sonner';

import type { TFunction } from 'i18next';

function timeAgo(dateStr: string | null, t: TFunction, tc: TFunction): string {
  if (!dateStr) return tc('never');
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return t('time.justNow');
  if (mins < 60) return t('time.minsAgo', { count: mins });
  const hours = Math.floor(mins / 60);
  if (hours < 24) return t('time.hoursAgo', { count: hours });
  const days = Math.floor(hours / 24);
  return t('time.daysAgo', { count: days });
}

export default function MonitorsPage() {
  const { t } = useTranslation('monitors');
  const { t: tc } = useTranslation('common');
  const queryClient = useQueryClient();

  const { data: monitors, isLoading } = useQuery({
    queryKey: ['monitors'],
    queryFn: monitorsApi.list,
    refetchInterval: 30_000,
  });

  const toggleMutation = useMutation({
    mutationFn: ({ productId, enabled }: { productId: number; enabled: boolean }) =>
      monitorsApi.update(productId, { enabled }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['monitors'] });
      toast.success(t('toast.monitorUpdated'));
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const toggleAutoBuild = useMutation({
    mutationFn: ({ productId, auto_build }: { productId: number; auto_build: boolean }) =>
      monitorsApi.update(productId, { auto_build }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['monitors'] });
      toast.success(t('toast.autoBuildUpdated'));
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const updateInterval = useMutation({
    mutationFn: ({ productId, interval }: { productId: number; interval: string }) =>
      monitorsApi.update(productId, { check_interval: interval }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['monitors'] });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const [checkingProductId, setCheckingProductId] = useState<number | null>(null);

  const checkNowMutation = useMutation({
    mutationFn: (productId: number) => {
      setCheckingProductId(productId);
      return monitorsApi.checkNow(productId);
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['monitors'] });
      toast.success(t('toast.latestVersion', { version: data.version }));
      setCheckingProductId(null);
    },
    onError: (err: Error) => {
      toast.error(err.message);
      setCheckingProductId(null);
    },
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t('page.title')}</h1>
      </div>

      {isLoading ? (
        <p className="text-muted-foreground">{tc('loading')}</p>
      ) : !monitors?.length ? (
        <Card>
          <CardContent className="p-8 text-center">
            <p className="text-muted-foreground">
              {t('page.empty')}
            </p>
          </CardContent>
        </Card>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('table.product')}</TableHead>
              <TableHead>{t('table.source')}</TableHead>
              <TableHead>{t('table.latestVersion')}</TableHead>
              <TableHead>{t('table.lastChecked')}</TableHead>
              <TableHead>{t('table.interval')}</TableHead>
              <TableHead>{t('table.autoBuild')}</TableHead>
              <TableHead>{t('table.status')}</TableHead>
              <TableHead className="w-[120px]" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {monitors.map((m: Monitor) => (
              <TableRow key={m.id}>
                <TableCell className="font-medium">{m.product_display_name}</TableCell>
                <TableCell>
                  {m.source_type === 'github' ? (
                    <span className="text-sm">{m.source_github_owner}/{m.source_github_repo}</span>
                  ) : (
                    <span className="text-sm text-muted-foreground">{t('table.custom')}</span>
                  )}
                </TableCell>
                <TableCell>
                  {m.last_known_version ? (
                    <Badge variant="secondary">{m.last_known_version}</Badge>
                  ) : (
                    <span className="text-sm text-muted-foreground">-</span>
                  )}
                </TableCell>
                <TableCell>
                  <span className="text-sm text-muted-foreground">
                    {timeAgo(m.last_checked_at, t, tc)}
                  </span>
                  {m.last_error && (
                    <div className="text-xs text-destructive" title={m.last_error}>
                      {tc('error')}
                    </div>
                  )}
                </TableCell>
                <TableCell>
                  <Select
                    value={m.check_interval}
                    onValueChange={(v) => updateInterval.mutate({ productId: m.product_id, interval: v })}
                  >
                    <SelectTrigger className="h-8 w-24">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="30m">{t('intervals.30min')}</SelectItem>
                      <SelectItem value="1h">{t('intervals.1h')}</SelectItem>
                      <SelectItem value="6h">{t('intervals.6h')}</SelectItem>
                      <SelectItem value="12h">{t('intervals.12h')}</SelectItem>
                      <SelectItem value="24h">{t('intervals.24h')}</SelectItem>
                    </SelectContent>
                  </Select>
                </TableCell>
                <TableCell>
                  <Button
                    variant={m.auto_build ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => toggleAutoBuild.mutate({ productId: m.product_id, auto_build: !m.auto_build })}
                  >
                    <Zap className="mr-1 h-3 w-3" />
                    {m.auto_build ? t('table.on') : t('table.off')}
                  </Button>
                </TableCell>
                <TableCell>
                  <Button
                    variant={m.enabled ? 'default' : 'secondary'}
                    size="sm"
                    onClick={() => toggleMutation.mutate({ productId: m.product_id, enabled: !m.enabled })}
                  >
                    {m.enabled ? (
                      <><Power className="mr-1 h-3 w-3" /> {tc('active')}</>
                    ) : (
                      <><PowerOff className="mr-1 h-3 w-3" /> {tc('disabled')}</>
                    )}
                  </Button>
                </TableCell>
                <TableCell>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => checkNowMutation.mutate(m.product_id)}
                    disabled={checkingProductId === m.product_id}
                  >
                    <RefreshCw className={`mr-1 h-3 w-3 ${checkingProductId === m.product_id ? 'animate-spin' : ''}`} />
                    {t('table.check')}
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
