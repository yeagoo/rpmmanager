import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { settingsApi } from '@/api/settings';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { toast } from 'sonner';
import { Save } from 'lucide-react';

interface SettingField {
  key: string;
  labelKey: string;
  descKey: string;
  type?: 'text' | 'url' | 'number' | 'password';
  section: 'general' | 'notification';
}

const settingFields: SettingField[] = [
  { key: 'site_name', labelKey: 'fields.siteName.label', descKey: 'fields.siteName.desc', section: 'general' },
  { key: 'base_url', labelKey: 'fields.baseUrl.label', descKey: 'fields.baseUrl.desc', type: 'url', section: 'general' },
  { key: 'repo_base_url', labelKey: 'fields.repoBaseUrl.label', descKey: 'fields.repoBaseUrl.desc', type: 'url', section: 'general' },
  { key: 'github_token', labelKey: 'fields.githubToken.label', descKey: 'fields.githubToken.desc', type: 'password', section: 'general' },
  { key: 'max_concurrent_builds', labelKey: 'fields.maxConcurrentBuilds.label', descKey: 'fields.maxConcurrentBuilds.desc', type: 'number', section: 'general' },
  { key: 'build_timeout', labelKey: 'fields.buildTimeout.label', descKey: 'fields.buildTimeout.desc', type: 'number', section: 'general' },
  { key: 'rollback_keep_count', labelKey: 'fields.rollbackKeepCount.label', descKey: 'fields.rollbackKeepCount.desc', type: 'number', section: 'general' },
  { key: 'notification_url', labelKey: 'fields.webhookUrl.label', descKey: 'fields.webhookUrl.desc', type: 'url', section: 'notification' },
  { key: 'notification_events', labelKey: 'fields.notificationEvents.label', descKey: 'fields.notificationEvents.desc', section: 'notification' },
];

export default function SettingsPage() {
  const { t: tc } = useTranslation('common');
  const { data: settings, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: settingsApi.getAll,
  });

  if (isLoading) {
    return <p className="text-muted-foreground">{tc('loading')}</p>;
  }

  return <SettingsForm initialValues={settings || {}} />;
}

function SettingsForm({ initialValues }: { initialValues: Record<string, string> }) {
  const { t } = useTranslation('settings');
  const { t: tc } = useTranslation('common');
  const queryClient = useQueryClient();
  const [form, setForm] = useState<Record<string, string>>(initialValues);

  const saveMutation = useMutation({
    mutationFn: settingsApi.update,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      toast.success(t('page.saved'));
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const handleSave = () => {
    // Only send changed values
    const changes: Record<string, string> = {};
    for (const field of settingFields) {
      const newVal = form[field.key] || '';
      const oldVal = initialValues[field.key] || '';
      if (newVal !== oldVal) {
        changes[field.key] = newVal;
      }
    }
    if (Object.keys(changes).length === 0) {
      toast.info(tc('noChanges'));
      return;
    }
    saveMutation.mutate(changes);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t('page.title')}</h1>
        <Button onClick={handleSave} disabled={saveMutation.isPending}>
          <Save className="mr-2 h-4 w-4" />
          {saveMutation.isPending ? tc('saving') : tc('save')}
        </Button>
      </div>

      {(['general', 'notification'] as const).map((section) => {
        const fields = settingFields.filter((f) => f.section === section);
        const title = section === 'general' ? t('sections.general') : t('sections.notifications');
        const desc = section === 'notification'
          ? t('sections.notificationsDesc')
          : undefined;
        return (
          <Card key={section}>
            <CardHeader>
              <CardTitle>{title}</CardTitle>
              {desc && <p className="text-sm text-muted-foreground">{desc}</p>}
            </CardHeader>
            <CardContent className="space-y-6">
              {fields.map((field, i) => (
                <div key={field.key}>
                  {i > 0 && <Separator className="mb-6" />}
                  <div className="space-y-2">
                    <Label htmlFor={field.key}>{t(field.labelKey)}</Label>
                    <Input
                      id={field.key}
                      type={field.type === 'password' ? 'password' : (field.type || 'text')}
                      value={form[field.key] || ''}
                      onChange={(e) => setForm((prev) => ({ ...prev, [field.key]: e.target.value }))}
                      placeholder={t(field.descKey)}
                      autoComplete={field.type === 'password' ? 'off' : undefined}
                    />
                    <p className="text-xs text-muted-foreground">{t(field.descKey)}</p>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
