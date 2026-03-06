import { useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { productsApi, type Product } from '@/api/products';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Plus, MoreHorizontal, Copy, Trash2, Download, Upload } from 'lucide-react';
import { toast } from 'sonner';

export default function ProductsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { t } = useTranslation('products');
  const { t: tc } = useTranslation('common');

  const { data: products, isLoading } = useQuery({
    queryKey: ['products'],
    queryFn: productsApi.list,
  });

  const deleteMutation = useMutation({
    mutationFn: productsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      toast.success(t('page.productDeleted'));
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const duplicateMutation = useMutation({
    mutationFn: productsApi.duplicate,
    onSuccess: (product: Product) => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      toast.success(t('page.duplicatedAs', { name: product.name }));
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const importMutation = useMutation({
    mutationFn: productsApi.importProducts,
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      toast.success(t('page.importedCount', { count: result.count }));
      if (result.errors?.length) {
        result.errors.forEach((e) => toast.error(e));
      }
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const handleImport = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (ev) => {
      try {
        const data = JSON.parse(ev.target?.result as string);
        const products = Array.isArray(data) ? data : [data];
        importMutation.mutate(products);
      } catch {
        toast.error(t('page.invalidJson'));
      }
    };
    reader.readAsText(file);
    e.target.value = '';
  };

  const handleExportAll = async () => {
    try {
      await productsApi.exportAll();
      toast.success(t('page.productsExported'));
    } catch {
      toast.error(t('page.exportFailed'));
    }
  };

  const handleExportOne = async (id: number, e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await productsApi.exportProduct(id);
    } catch {
      toast.error(t('page.exportFailed'));
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t('page.title')}</h1>
        <div className="flex items-center gap-2">
          <input
            ref={fileInputRef}
            type="file"
            accept=".json"
            className="hidden"
            onChange={handleImport}
          />
          <Button variant="outline" onClick={() => fileInputRef.current?.click()} disabled={importMutation.isPending}>
            <Upload className="mr-2 h-4 w-4" />
            {t('page.import')}
          </Button>
          {products && products.length > 0 && (
            <Button variant="outline" onClick={handleExportAll}>
              <Download className="mr-2 h-4 w-4" />
              {t('page.exportAll')}
            </Button>
          )}
          <Button onClick={() => navigate('/products/new')}>
            <Plus className="mr-2 h-4 w-4" />
            {t('page.newProduct')}
          </Button>
        </div>
      </div>

      {isLoading ? (
        <p className="text-muted-foreground">{tc('loading')}</p>
      ) : !products?.length ? (
        <div className="rounded-md border border-dashed p-8 text-center">
          <p className="text-muted-foreground">{t('page.empty')}</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('table.name')}</TableHead>
              <TableHead>{t('table.source')}</TableHead>
              <TableHead>{t('table.distros')}</TableHead>
              <TableHead>{t('table.arch')}</TableHead>
              <TableHead>{t('table.latestVersion')}</TableHead>
              <TableHead>{t('table.status')}</TableHead>
              <TableHead className="w-[50px]" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {products.map((p) => (
              <TableRow
                key={p.id}
                className="cursor-pointer"
                onClick={() => navigate(`/products/${p.id}`)}
              >
                <TableCell>
                  <div>
                    <div className="font-medium">{p.display_name}</div>
                    <div className="text-xs text-muted-foreground">{p.name}</div>
                  </div>
                </TableCell>
                <TableCell>
                  {p.source_type === 'github' ? (
                    <span className="text-sm">{p.source_github_owner}/{p.source_github_repo}</span>
                  ) : (
                    <span className="text-sm text-muted-foreground">{t('table.customUrl')}</span>
                  )}
                </TableCell>
                <TableCell>
                  <Badge variant="secondary">{t('table.distroCount', { count: p.target_distros?.length || 0 })}</Badge>
                </TableCell>
                <TableCell>
                  <span className="text-sm">{(p.architectures || []).join(', ')}</span>
                </TableCell>
                <TableCell>
                  {p.latest_version ? (
                    <Badge>{p.latest_version}</Badge>
                  ) : (
                    <span className="text-sm text-muted-foreground">-</span>
                  )}
                </TableCell>
                <TableCell>
                  <Badge variant={p.enabled ? 'default' : 'secondary'}>
                    {p.enabled ? tc('active') : tc('disabled')}
                  </Badge>
                </TableCell>
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
                      <Button variant="ghost" size="sm">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={(e) => { e.stopPropagation(); duplicateMutation.mutate(p.id); }}>
                        <Copy className="mr-2 h-4 w-4" />
                        {t('table.duplicate')}
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={(e) => handleExportOne(p.id, e)}>
                        <Download className="mr-2 h-4 w-4" />
                        {t('table.export')}
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        className="text-destructive"
                        onClick={(e) => {
                          e.stopPropagation();
                          if (confirm(t('page.confirmDelete', { name: p.display_name }))) {
                            deleteMutation.mutate(p.id);
                          }
                        }}
                      >
                        <Trash2 className="mr-2 h-4 w-4" />
                        {tc('delete')}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
