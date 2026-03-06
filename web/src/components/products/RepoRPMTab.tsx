import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { productsApi, type Product, type RepoRPMResult } from '@/api/products';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { toast } from 'sonner';
import DistroSelector from './DistroSelector';
import { Download, Package, RefreshCw, Copy, Check } from 'lucide-react';

interface RepoRPMTabProps {
  product: Product;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function RepoRPMTab({ product }: RepoRPMTabProps) {
  const { t } = useTranslation('products');
  const [distros, setDistros] = useState<string[]>(product.target_distros || []);
  const [version, setVersion] = useState('1.0');
  const [copiedCmd, setCopiedCmd] = useState(false);

  const baseURL = product.base_url || window.location.origin;

  const { data: existing, refetch } = useQuery({
    queryKey: ['repo-rpm', product.id],
    queryFn: () => productsApi.getRepoRPM(product.id).catch(() => null),
  });

  const generateMutation = useMutation({
    mutationFn: () => productsApi.generateRepoRPM(product.id, { distros, version }),
    onSuccess: (result: RepoRPMResult) => {
      toast.success(t('repoRpm.generated', { filename: result.filename }));
      refetch();
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const { data: distroInfo } = useQuery({
    queryKey: ['distros'],
    queryFn: productsApi.getDistros,
  });

  const getProductLinesForDistros = () => {
    if (!distroInfo) return [];
    const plIds = new Set<string>();
    for (const dv of distros) {
      for (const pl of distroInfo.product_lines) {
        const group = distroInfo.distro_groups[pl.id] || [];
        if (group.some((d) => `${d.distro}:${d.version}` === dv)) {
          plIds.add(pl.id);
        }
      }
    }
    return distroInfo.product_lines.filter((pl) => plIds.has(pl.id));
  };

  const selectedPLs = getProductLinesForDistros();

  const handleDownload = () => {
    window.open(`/api/products/${product.id}/repo-rpm/download`, '_blank');
  };

  const installCmd = existing
    ? `dnf install ${baseURL}/${product.name}/repo-rpm/${existing.filename}`
    : `dnf install ${baseURL}/${product.name}/repo-rpm/${product.name}-repo-${version}-1.noarch.rpm`;

  const copyInstallCmd = () => {
    navigator.clipboard.writeText(installCmd);
    setCopiedCmd(true);
    setTimeout(() => setCopiedCmd(false), 2000);
  };

  const hasGPGKey = product.gpg_key_id !== null && product.gpg_key_id !== undefined;

  return (
    <div className="space-y-6">
      {existing && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-lg">
              <Package className="h-5 w-5" />
              {t('repoRpm.currentRepoRpm')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <p className="font-mono text-sm">{existing.filename}</p>
                <p className="text-sm text-muted-foreground">{formatSize(existing.size)}</p>
              </div>
              <Button onClick={handleDownload} variant="outline" size="sm">
                <Download className="mr-2 h-4 w-4" />
                {t('repoRpm.download')}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-lg">{t('repoRpm.installCommand')}</CardTitle>
          <CardDescription>
            {t('repoRpm.installCommandDesc')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-md bg-muted p-3 font-mono text-sm break-all">
              {installCmd}
            </code>
            <Button variant="ghost" size="icon" onClick={copyInstallCmd}>
              {copiedCmd ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">{t('repoRpm.generateTitle')}</CardTitle>
          <CardDescription>
            {t('repoRpm.generateDesc')}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {!hasGPGKey && (
            <div className="rounded-md border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-800 dark:border-yellow-900 dark:bg-yellow-950 dark:text-yellow-200">
              {t('repoRpm.noGpgWarning')}
            </div>
          )}

          <div className="max-w-xs space-y-2">
            <Label htmlFor="repo-rpm-version">{t('repoRpm.version')}</Label>
            <Input
              id="repo-rpm-version"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder="1.0"
            />
            <p className="text-xs text-muted-foreground">
              {t('repoRpm.versionHint')}
            </p>
          </div>

          <div className="space-y-2">
            <Label>{t('repoRpm.targetDistros')}</Label>
            <p className="text-xs text-muted-foreground">
              {t('repoRpm.targetDistrosHint')}
            </p>
            <DistroSelector value={distros} onChange={setDistros} />
          </div>

          {selectedPLs.length > 0 && (
            <div className="space-y-2">
              <Label>{t('repoRpm.rpmContentsPreview')}</Label>
              <div className="rounded-md border bg-muted/50 p-4 font-mono text-sm">
                <p className="mb-2 text-muted-foreground">
                  # {product.name}-repo-{version}-1.noarch.rpm
                </p>
                <p>/etc/pki/rpm-gpg/RPM-GPG-KEY-{product.name}</p>
                {selectedPLs.map((pl) => (
                  <p key={pl.id}>/etc/yum.repos.d/{product.name}-{pl.id}.repo</p>
                ))}
              </div>
            </div>
          )}

          {selectedPLs.length > 0 && (
            <div className="space-y-2">
              <Label>{t('repoRpm.repoFilePreview')}</Label>
              <div className="max-h-64 overflow-auto rounded-md border bg-muted/50 p-4 font-mono text-xs">
                {selectedPLs.map((pl, i) => (
                  <div key={pl.id}>
                    {i > 0 && <div className="my-2 border-t border-dashed" />}
                    <p className="text-muted-foreground"># /etc/yum.repos.d/{product.name}-{pl.id}.repo</p>
                    <p>[{product.name}-{pl.id}]</p>
                    <p>name={product.display_name || product.name} for {pl.id}</p>
                    <p>baseurl={baseURL}/{product.name}/{pl.path}/$basearch/</p>
                    <p>enabled=1</p>
                    <p>gpgcheck=1</p>
                    <p>repo_gpgcheck=1</p>
                    <p>gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-{product.name}</p>
                  </div>
                ))}
              </div>
            </div>
          )}

          {selectedPLs.length > 0 && (
            <div className="flex flex-wrap gap-2">
              <span className="text-sm text-muted-foreground">{t('repoRpm.productLines')}</span>
              {selectedPLs.map((pl) => (
                <Badge key={pl.id} variant="secondary">{pl.id}</Badge>
              ))}
            </div>
          )}

          <div className="flex items-center gap-4">
            <Button
              onClick={() => generateMutation.mutate()}
              disabled={generateMutation.isPending || distros.length === 0 || !hasGPGKey}
            >
              {generateMutation.isPending ? (
                <>
                  <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                  {t('repoRpm.generating')}
                </>
              ) : (
                <>
                  <Package className="mr-2 h-4 w-4" />
                  {t('repoRpm.generate')}
                </>
              )}
            </Button>
            {distros.length === 0 && (
              <span className="text-sm text-muted-foreground">{t('repoRpm.selectAtLeastOne')}</span>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
