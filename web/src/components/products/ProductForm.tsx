import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { Product } from '@/api/products';
import { gpgKeysApi, type GPGKey } from '@/api/gpgkeys';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import DistroSelector from './DistroSelector';

interface ProductFormProps {
  initialData?: Product;
  onSubmit: (data: Partial<Product>) => void;
  loading?: boolean;
}

const EMPTY_PRODUCT: Partial<Product> = {
  name: '',
  display_name: '',
  description: '',
  source_type: 'github',
  source_github_owner: '',
  source_github_repo: '',
  source_url_template: '',
  nfpm_config: '{}',
  target_distros: [],
  architectures: ['x86_64', 'aarch64'],
  maintainer: '',
  vendor: '',
  homepage: '',
  license: 'Apache-2.0',
  script_postinstall: '',
  script_preremove: '',
  systemd_service: '',
  default_config: '',
  default_config_path: '',
  extra_files: '[]',
  base_url: '',
  sm2_enabled: false,
  enabled: true,
};

export default function ProductForm({ initialData, onSubmit, loading }: ProductFormProps) {
  const [form, setForm] = useState<Partial<Product>>(initialData || EMPTY_PRODUCT);
  const { t } = useTranslation('products');
  const { t: tc } = useTranslation('common');

  const { data: gpgKeys } = useQuery<GPGKey[]>({
    queryKey: ['gpg-keys'],
    queryFn: gpgKeysApi.list,
  });

  const update = <K extends keyof Product>(key: K, value: Product[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(form);
  };

  return (
    <form onSubmit={handleSubmit}>
      <Tabs defaultValue="basic" className="w-full">
        <TabsList className="mb-4">
          <TabsTrigger value="basic">{t('form.basic')}</TabsTrigger>
          <TabsTrigger value="packaging">{t('form.packaging')}</TabsTrigger>
          <TabsTrigger value="distros">{t('form.distros')}</TabsTrigger>
          <TabsTrigger value="scripts">{t('form.scripts')}</TabsTrigger>
          <TabsTrigger value="systemd">{t('form.systemd')}</TabsTrigger>
          <TabsTrigger value="config">{t('form.config')}</TabsTrigger>
        </TabsList>

        {/* Tab 1: Basic */}
        <TabsContent value="basic" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="name">{t('form.nameSlug')}</Label>
              <Input
                id="name"
                value={form.name || ''}
                onChange={(e) => update('name', e.target.value)}
                placeholder="caddy"
                pattern="[a-z0-9][a-z0-9-]*[a-z0-9]"
                required
              />
              <p className="text-xs text-muted-foreground">{t('form.nameHint')}</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="display_name">{t('form.displayName')}</Label>
              <Input
                id="display_name"
                value={form.display_name || ''}
                onChange={(e) => update('display_name', e.target.value)}
                placeholder="Caddy Server"
                required
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">{t('form.description')}</Label>
            <Textarea
              id="description"
              value={form.description || ''}
              onChange={(e) => update('description', e.target.value)}
              placeholder="A powerful, enterprise-ready web server..."
              rows={2}
            />
          </div>

          <div className="space-y-2">
            <Label>{t('form.sourceType')}</Label>
            <Select
              value={form.source_type || 'github'}
              onValueChange={(v) => update('source_type', v as 'github' | 'url')}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="github">{t('form.githubRelease')}</SelectItem>
                <SelectItem value="url">{t('form.customUrl')}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {form.source_type === 'github' ? (
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>{t('form.githubOwner')}</Label>
                <Input
                  value={form.source_github_owner || ''}
                  onChange={(e) => update('source_github_owner', e.target.value)}
                  placeholder="caddyserver"
                />
              </div>
              <div className="space-y-2">
                <Label>{t('form.githubRepo')}</Label>
                <Input
                  value={form.source_github_repo || ''}
                  onChange={(e) => update('source_github_repo', e.target.value)}
                  placeholder="caddy"
                />
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              <Label>{t('form.urlTemplate')}</Label>
              <Input
                value={form.source_url_template || ''}
                onChange={(e) => update('source_url_template', e.target.value)}
                placeholder="https://example.com/releases/{version}/binary-{arch}"
              />
              <p className="text-xs text-muted-foreground">
                {t('form.urlTemplateHint')}
              </p>
            </div>
          )}

          <div className="space-y-2">
            <Label>{t('form.homepage')}</Label>
            <Input
              value={form.homepage || ''}
              onChange={(e) => update('homepage', e.target.value)}
              placeholder="https://caddyserver.com"
            />
          </div>
        </TabsContent>

        {/* Tab 2: Packaging */}
        <TabsContent value="packaging" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label>{t('form.maintainer')}</Label>
              <Input
                value={form.maintainer || ''}
                onChange={(e) => update('maintainer', e.target.value)}
                placeholder="Your Name <email@example.com>"
              />
            </div>
            <div className="space-y-2">
              <Label>{t('form.vendor')}</Label>
              <Input
                value={form.vendor || ''}
                onChange={(e) => update('vendor', e.target.value)}
                placeholder="Your Organization"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label>{t('form.license')}</Label>
            <Input
              value={form.license || ''}
              onChange={(e) => update('license', e.target.value)}
              placeholder="Apache-2.0"
            />
          </div>

          <div className="space-y-2">
            <Label>{t('form.nfpmConfig')}</Label>
            <Textarea
              value={form.nfpm_config || '{}'}
              onChange={(e) => update('nfpm_config', e.target.value)}
              className="font-mono text-sm"
              rows={12}
              placeholder={`{
  "description": "...",
  "contents": [
    {"src": "{{binary}}", "dst": "/usr/bin/caddy", "mode": "0755"}
  ],
  "depends": ["shadow-utils"]
}`}
            />
          </div>

          <div className="space-y-2">
            <Label>{t('form.architectures')}</Label>
            <div className="flex gap-4">
              {['x86_64', 'aarch64'].map((arch) => (
                <label key={arch} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={(form.architectures || []).includes(arch)}
                    onCheckedChange={(checked) => {
                      const current = form.architectures || [];
                      if (checked) {
                        update('architectures', [...current, arch]);
                      } else {
                        update('architectures', current.filter((a) => a !== arch));
                      }
                    }}
                  />
                  {arch}
                </label>
              ))}
            </div>
          </div>
        </TabsContent>

        {/* Tab 3: Distros */}
        <TabsContent value="distros">
          <DistroSelector
            value={form.target_distros || []}
            onChange={(distros) => update('target_distros', distros)}
          />
        </TabsContent>

        {/* Tab 4: Scripts */}
        <TabsContent value="scripts" className="space-y-4">
          <div className="space-y-2">
            <Label>{t('form.postInstallScript')}</Label>
            <Textarea
              value={form.script_postinstall || ''}
              onChange={(e) => update('script_postinstall', e.target.value)}
              className="font-mono text-sm"
              rows={10}
              placeholder="#!/bin/bash&#10;# Create user, enable service, etc."
            />
          </div>
          <div className="space-y-2">
            <Label>{t('form.preRemoveScript')}</Label>
            <Textarea
              value={form.script_preremove || ''}
              onChange={(e) => update('script_preremove', e.target.value)}
              className="font-mono text-sm"
              rows={10}
              placeholder="#!/bin/bash&#10;# Stop service, cleanup, etc."
            />
          </div>
        </TabsContent>

        {/* Tab 5: Systemd */}
        <TabsContent value="systemd" className="space-y-4">
          <div className="space-y-2">
            <Label>{t('form.systemdServiceFile')}</Label>
            <Textarea
              value={form.systemd_service || ''}
              onChange={(e) => update('systemd_service', e.target.value)}
              className="font-mono text-sm"
              rows={20}
              placeholder={`[Unit]
Description=...
After=network.target

[Service]
Type=notify
User=caddy
ExecStart=/usr/bin/caddy run

[Install]
WantedBy=multi-user.target`}
            />
          </div>
        </TabsContent>

        {/* Tab 6: Config */}
        <TabsContent value="config" className="space-y-4">
          <div className="space-y-2">
            <Label>{t('form.gpgSigningKey')}</Label>
            <Select
              value={form.gpg_key_id != null ? String(form.gpg_key_id) : 'none'}
              onValueChange={(v) => update('gpg_key_id', v === 'none' ? null : Number(v))}
            >
              <SelectTrigger>
                <SelectValue placeholder={t('form.selectGpgKey')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">{t('form.noGpgKey')}</SelectItem>
                {(gpgKeys || []).map((key) => (
                  <SelectItem key={key.id} value={String(key.id)}>
                    {key.uid_name} ({key.key_id}) — {key.algorithm}
                    {key.is_default ? ` ${t('form.gpgDefault')}` : ''}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              {t('form.gpgKeyHint')}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t('form.defaultConfigPath')}</Label>
            <Input
              value={form.default_config_path || ''}
              onChange={(e) => update('default_config_path', e.target.value)}
              placeholder="/etc/caddy/Caddyfile"
            />
          </div>
          <div className="space-y-2">
            <Label>{t('form.defaultConfigContent')}</Label>
            <Textarea
              value={form.default_config || ''}
              onChange={(e) => update('default_config', e.target.value)}
              className="font-mono text-sm"
              rows={12}
              placeholder="# Default configuration file content..."
            />
          </div>
          <div className="space-y-2">
            <Label>{t('form.baseUrlOverride')}</Label>
            <Input
              value={form.base_url || ''}
              onChange={(e) => update('base_url', e.target.value)}
              placeholder={t('form.baseUrlOverrideHint')}
            />
          </div>
        </TabsContent>
      </Tabs>

      <div className="mt-6 flex justify-end gap-2">
        <Button type="submit" disabled={loading}>
          {loading ? tc('saving') : initialData ? t('form.updateProduct') : t('form.createProduct')}
        </Button>
      </div>
    </form>
  );
}
