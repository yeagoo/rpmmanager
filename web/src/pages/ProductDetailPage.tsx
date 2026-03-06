import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { productsApi, type Product } from '@/api/products';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ArrowLeft } from 'lucide-react';
import { toast } from 'sonner';
import ProductForm from '@/components/products/ProductForm';
import RepoRPMTab from '@/components/products/RepoRPMTab';

export default function ProductDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isNew = id === 'new';
  const [activeTab, setActiveTab] = useState('settings');

  const { data: product, isLoading } = useQuery({
    queryKey: ['product', id],
    queryFn: () => productsApi.get(Number(id)),
    enabled: !isNew,
  });

  const createMutation = useMutation({
    mutationFn: (data: Partial<Product>) => productsApi.create(data),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      toast.success('Product created');
      navigate(`/products/${result.id}`);
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const updateMutation = useMutation({
    mutationFn: (data: Partial<Product>) => productsApi.update(Number(id), data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['product', id] });
      toast.success('Product updated');
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const handleSubmit = (data: Partial<Product>) => {
    if (isNew) {
      createMutation.mutate(data);
    } else {
      updateMutation.mutate(data);
    }
  };

  if (!isNew && isLoading) {
    return <p className="text-muted-foreground">Loading...</p>;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="sm" onClick={() => navigate('/products')}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <h1 className="text-3xl font-bold">
          {isNew ? 'New Product' : product?.display_name || 'Edit Product'}
        </h1>
      </div>

      {isNew ? (
        <ProductForm
          onSubmit={handleSubmit}
          loading={createMutation.isPending}
        />
      ) : (
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="settings">Product Settings</TabsTrigger>
            <TabsTrigger value="repo-rpm">Repo RPM</TabsTrigger>
          </TabsList>

          <TabsContent value="settings" className="mt-4">
            <ProductForm
              initialData={product}
              onSubmit={handleSubmit}
              loading={updateMutation.isPending}
            />
          </TabsContent>

          <TabsContent value="repo-rpm" className="mt-4">
            {product && <RepoRPMTab product={product} />}
          </TabsContent>
        </Tabs>
      )}
    </div>
  );
}
