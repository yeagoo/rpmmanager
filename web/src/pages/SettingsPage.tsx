import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
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
  label: string;
  description: string;
  type?: 'text' | 'url' | 'number' | 'password';
  section: 'general' | 'notification';
}

const settingFields: SettingField[] = [
  { key: 'site_name', label: 'Site Name', description: 'Display name for this RPM Manager instance', section: 'general' },
  { key: 'base_url', label: 'Base URL', description: 'Public URL for repository access', type: 'url', section: 'general' },
  { key: 'repo_base_url', label: 'Repository Base URL', description: 'Base URL used in .repo file generation', type: 'url', section: 'general' },
  { key: 'github_token', label: 'GitHub Token', description: 'Token for GitHub API (version monitoring, release downloads)', type: 'password', section: 'general' },
  { key: 'max_concurrent_builds', label: 'Max Concurrent Builds', description: 'Maximum number of builds running at the same time', type: 'number', section: 'general' },
  { key: 'build_timeout', label: 'Build Timeout (seconds)', description: 'Maximum time allowed for a single build', type: 'number', section: 'general' },
  { key: 'rollback_keep_count', label: 'Rollback Keep Count', description: 'Number of rollback snapshots to keep (default: 3)', type: 'number', section: 'general' },
  { key: 'notification_url', label: 'Webhook URL', description: 'URL for build notifications (supports Telegram Bot, WeChat Work, DingTalk, or generic webhook)', type: 'url', section: 'notification' },
  { key: 'notification_events', label: 'Notification Events', description: 'Comma-separated events to notify on (default: build.success,build.failed)', section: 'notification' },
];

export default function SettingsPage() {
  const { data: settings, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: settingsApi.getAll,
  });

  if (isLoading) {
    return <p className="text-muted-foreground">Loading...</p>;
  }

  return <SettingsForm initialValues={settings || {}} />;
}

function SettingsForm({ initialValues }: { initialValues: Record<string, string> }) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<Record<string, string>>(initialValues);

  const saveMutation = useMutation({
    mutationFn: settingsApi.update,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      toast.success('Settings saved');
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
      toast.info('No changes to save');
      return;
    }
    saveMutation.mutate(changes);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Settings</h1>
        <Button onClick={handleSave} disabled={saveMutation.isPending}>
          <Save className="mr-2 h-4 w-4" />
          {saveMutation.isPending ? 'Saving...' : 'Save'}
        </Button>
      </div>

      {(['general', 'notification'] as const).map((section) => {
        const fields = settingFields.filter((f) => f.section === section);
        const title = section === 'general' ? 'General' : 'Build Notifications';
        const desc = section === 'notification'
          ? 'Configure webhook notifications for build events. Supports Telegram Bot, WeChat Work, DingTalk, and generic webhooks.'
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
                    <Label htmlFor={field.key}>{field.label}</Label>
                    <Input
                      id={field.key}
                      type={field.type === 'password' ? 'password' : (field.type || 'text')}
                      value={form[field.key] || ''}
                      onChange={(e) => setForm((prev) => ({ ...prev, [field.key]: e.target.value }))}
                      placeholder={field.description}
                      autoComplete={field.type === 'password' ? 'off' : undefined}
                    />
                    <p className="text-xs text-muted-foreground">{field.description}</p>
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
