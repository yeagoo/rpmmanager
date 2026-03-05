import { useState, useEffect } from 'react';
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
  type?: 'text' | 'url' | 'number';
}

const settingFields: SettingField[] = [
  { key: 'site_name', label: 'Site Name', description: 'Display name for this RPM Manager instance' },
  { key: 'base_url', label: 'Base URL', description: 'Public URL for repository access', type: 'url' },
  { key: 'repo_base_url', label: 'Repository Base URL', description: 'Base URL used in .repo file generation', type: 'url' },
  { key: 'github_token', label: 'GitHub Token', description: 'Token for GitHub API (version monitoring, release downloads)' },
  { key: 'max_concurrent_builds', label: 'Max Concurrent Builds', description: 'Maximum number of builds running at the same time', type: 'number' },
  { key: 'build_timeout', label: 'Build Timeout (seconds)', description: 'Maximum time allowed for a single build', type: 'number' },
];

export default function SettingsPage() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<Record<string, string>>({});

  const { data: settings, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: settingsApi.getAll,
  });

  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    // Only set form from server data on first load, not on refetch
    if (settings && !initialized) {
      setForm({ ...settings });
      setInitialized(true);
    }
  }, [settings, initialized]);

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
      const oldVal = settings?.[field.key] || '';
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

  if (isLoading) {
    return <p className="text-muted-foreground">Loading...</p>;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Settings</h1>
        <Button onClick={handleSave} disabled={saveMutation.isPending}>
          <Save className="mr-2 h-4 w-4" />
          {saveMutation.isPending ? 'Saving...' : 'Save'}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {settingFields.map((field, i) => (
            <div key={field.key}>
              {i > 0 && <Separator className="mb-6" />}
              <div className="space-y-2">
                <Label htmlFor={field.key}>{field.label}</Label>
                <Input
                  id={field.key}
                  type={field.key === 'github_token' ? 'password' : (field.type || 'text')}
                  value={form[field.key] || ''}
                  onChange={(e) => setForm((prev) => ({ ...prev, [field.key]: e.target.value }))}
                  placeholder={field.description}
                  autoComplete={field.key === 'github_token' ? 'off' : undefined}
                />
                <p className="text-xs text-muted-foreground">{field.description}</p>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
