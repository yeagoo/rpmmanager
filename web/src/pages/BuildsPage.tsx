import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { buildsApi, type Build } from '@/api/builds';
import { productsApi, type Product } from '@/api/products';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import BuildStatusBadge from '@/components/builds/BuildStatusBadge';
import { Plus } from 'lucide-react';
import { toast } from 'sonner';

export default function BuildsPage() {
  const { t } = useTranslation('builds');
  const { t: tc } = useTranslation('common');
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [selectedProduct, setSelectedProduct] = useState('');
  const [version, setVersion] = useState('');

  const { data: builds, isLoading } = useQuery({
    queryKey: ['builds'],
    queryFn: () => buildsApi.list(undefined, 50),
    refetchInterval: 5000,
  });

  const { data: products } = useQuery({
    queryKey: ['products'],
    queryFn: productsApi.list,
  });

  const triggerMutation = useMutation({
    mutationFn: () => buildsApi.trigger(Number(selectedProduct), version),
    onSuccess: (build: Build) => {
      queryClient.invalidateQueries({ queryKey: ['builds'] });
      toast.success(t('page.buildTriggered'));
      setOpen(false);
      navigate(`/builds/${build.id}`);
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const formatDuration = (seconds: number) => {
    if (seconds === 0) return '-';
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return m > 0 ? `${m}m ${s}s` : `${s}s`;
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t('page.title')}</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              {t('page.newBuild')}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t('page.triggerBuild')}</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t('page.product')}</label>
                <Select value={selectedProduct} onValueChange={setSelectedProduct}>
                  <SelectTrigger>
                    <SelectValue placeholder={t('page.selectProduct')} />
                  </SelectTrigger>
                  <SelectContent>
                    {products?.map((p: Product) => (
                      <SelectItem key={p.id} value={String(p.id)}>
                        {p.display_name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t('page.version')}</label>
                <Input
                  value={version}
                  onChange={(e) => setVersion(e.target.value)}
                  placeholder="2.9.0"
                />
              </div>
              <Button
                className="w-full"
                onClick={() => triggerMutation.mutate()}
                disabled={!selectedProduct || !version || triggerMutation.isPending}
              >
                {triggerMutation.isPending ? t('page.triggering') : t('page.startBuild')}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {isLoading ? (
        <p className="text-muted-foreground">{tc('loading')}</p>
      ) : !builds?.length ? (
        <div className="rounded-md border border-dashed p-8 text-center">
          <p className="text-muted-foreground">{t('page.empty')}</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('table.id')}</TableHead>
              <TableHead>{t('table.product')}</TableHead>
              <TableHead>{t('table.version')}</TableHead>
              <TableHead>{t('table.status')}</TableHead>
              <TableHead>{t('table.trigger')}</TableHead>
              <TableHead>{t('table.rpms')}</TableHead>
              <TableHead>{t('table.duration')}</TableHead>
              <TableHead>{t('table.time')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {builds.map((b: Build) => (
              <TableRow
                key={b.id}
                className="cursor-pointer"
                onClick={() => navigate(`/builds/${b.id}`)}
              >
                <TableCell className="font-mono">#{b.id}</TableCell>
                <TableCell>{b.product_display_name || b.product_name}</TableCell>
                <TableCell className="font-mono">{b.version}</TableCell>
                <TableCell>
                  <BuildStatusBadge status={b.status} />
                </TableCell>
                <TableCell className="capitalize text-sm">{b.trigger_type}</TableCell>
                <TableCell>{b.rpm_count || '-'}</TableCell>
                <TableCell>{formatDuration(b.duration_seconds)}</TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {new Date(b.created_at).toLocaleString()}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
