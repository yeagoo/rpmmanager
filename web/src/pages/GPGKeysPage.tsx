import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { gpgKeysApi, type GPGKey } from '@/api/gpgkeys';
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
import { Plus, Upload, MoreHorizontal, Trash2, Download, Star } from 'lucide-react';
import { toast } from 'sonner';
import ImportKeyDialog from '@/components/gpg/ImportKeyDialog';
import GenerateKeyDialog from '@/components/gpg/GenerateKeyDialog';

export default function GPGKeysPage() {
  const { t } = useTranslation('gpg');
  const { t: tc } = useTranslation('common');
  const queryClient = useQueryClient();
  const [importOpen, setImportOpen] = useState(false);
  const [generateOpen, setGenerateOpen] = useState(false);

  const { data: keys, isLoading } = useQuery({
    queryKey: ['gpg-keys'],
    queryFn: gpgKeysApi.list,
  });

  const deleteMutation = useMutation({
    mutationFn: gpgKeysApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gpg-keys'] });
      toast.success(t('page.keyDeleted'));
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const setDefaultMutation = useMutation({
    mutationFn: gpgKeysApi.setDefault,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gpg-keys'] });
      toast.success(t('page.defaultKeyUpdated'));
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const handleExport = async (key: GPGKey) => {
    try {
      const armor = await gpgKeysApi.export(key.id);
      const blob = new Blob([armor], { type: 'application/pgp-keys' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${key.key_id || key.fingerprint}.pub.asc`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Export failed');
    }
  };

  const formatFingerprint = (fp: string) => {
    return fp.replace(/(.{4})/g, '$1 ').trim();
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t('page.title')}</h1>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setImportOpen(true)}>
            <Upload className="mr-2 h-4 w-4" />
            {t('page.importKey')}
          </Button>
          <Button onClick={() => setGenerateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t('page.generateKey')}
          </Button>
        </div>
      </div>

      {isLoading ? (
        <p className="text-muted-foreground">{tc('loading')}</p>
      ) : !keys?.length ? (
        <div className="rounded-md border border-dashed p-8 text-center">
          <p className="text-muted-foreground">
            {t('page.empty')}
          </p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('table.name')}</TableHead>
              <TableHead>{t('table.keyId')}</TableHead>
              <TableHead>{t('table.algorithm')}</TableHead>
              <TableHead>{t('table.uid')}</TableHead>
              <TableHead>{t('table.expires')}</TableHead>
              <TableHead>{t('table.status')}</TableHead>
              <TableHead className="w-[50px]" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {keys.map((key) => (
              <TableRow key={key.id}>
                <TableCell>
                  <div>
                    <div className="font-medium">{key.name}</div>
                    <div className="font-mono text-xs text-muted-foreground">
                      {formatFingerprint(key.fingerprint)}
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <code className="text-sm">{key.key_id}</code>
                </TableCell>
                <TableCell>
                  <span className="text-sm">
                    {key.algorithm}
                    {key.key_length > 0 && ` ${key.key_length}`}
                  </span>
                </TableCell>
                <TableCell>
                  <div className="text-sm">
                    <div>{key.uid_name}</div>
                    <div className="text-muted-foreground">{key.uid_email}</div>
                  </div>
                </TableCell>
                <TableCell>
                  {key.expires_date ? (
                    <span className="text-sm">
                      {new Date(key.expires_date).toLocaleDateString()}
                    </span>
                  ) : (
                    <span className="text-sm text-muted-foreground">{tc('never')}</span>
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex gap-1">
                    {key.is_default && <Badge>{t('table.default')}</Badge>}
                    {key.has_private && (
                      <Badge variant="secondary">{t('table.private')}</Badge>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="sm">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => handleExport(key)}>
                        <Download className="mr-2 h-4 w-4" />
                        {t('table.exportPublicKey')}
                      </DropdownMenuItem>
                      {!key.is_default && (
                        <DropdownMenuItem onClick={() => setDefaultMutation.mutate(key.id)}>
                          <Star className="mr-2 h-4 w-4" />
                          {t('table.setAsDefault')}
                        </DropdownMenuItem>
                      )}
                      <DropdownMenuItem
                        className="text-destructive"
                        onClick={() => {
                          if (confirm(t('page.confirmDelete', { name: key.name }))) {
                            deleteMutation.mutate(key.id);
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

      <ImportKeyDialog
        open={importOpen}
        onOpenChange={setImportOpen}
      />
      <GenerateKeyDialog
        open={generateOpen}
        onOpenChange={setGenerateOpen}
      />
    </div>
  );
}
