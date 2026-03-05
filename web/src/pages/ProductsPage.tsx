import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
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
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Plus, MoreHorizontal, Copy, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

export default function ProductsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: products, isLoading } = useQuery({
    queryKey: ['products'],
    queryFn: productsApi.list,
  });

  const deleteMutation = useMutation({
    mutationFn: productsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      toast.success('Product deleted');
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const duplicateMutation = useMutation({
    mutationFn: productsApi.duplicate,
    onSuccess: (product: Product) => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      toast.success(`Duplicated as "${product.name}"`);
    },
    onError: (err: Error) => toast.error(err.message),
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Products</h1>
        <Button onClick={() => navigate('/products/new')}>
          <Plus className="mr-2 h-4 w-4" />
          New Product
        </Button>
      </div>

      {isLoading ? (
        <p className="text-muted-foreground">Loading...</p>
      ) : !products?.length ? (
        <div className="rounded-md border border-dashed p-8 text-center">
          <p className="text-muted-foreground">No products yet. Create your first product to get started.</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Source</TableHead>
              <TableHead>Distros</TableHead>
              <TableHead>Arch</TableHead>
              <TableHead>Latest Version</TableHead>
              <TableHead>Status</TableHead>
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
                    <span className="text-sm text-muted-foreground">Custom URL</span>
                  )}
                </TableCell>
                <TableCell>
                  <Badge variant="secondary">{p.target_distros?.length || 0} distros</Badge>
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
                    {p.enabled ? 'Active' : 'Disabled'}
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
                        Duplicate
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        className="text-destructive"
                        onClick={(e) => {
                          e.stopPropagation();
                          if (confirm(`Delete "${p.display_name}"?`)) {
                            deleteMutation.mutate(p.id);
                          }
                        }}
                      >
                        <Trash2 className="mr-2 h-4 w-4" />
                        Delete
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
