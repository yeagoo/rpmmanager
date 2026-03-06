import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { gpgKeysApi, type GenerateKeyRequest } from '@/api/gpgkeys';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { toast } from 'sonner';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const defaultForm: GenerateKeyRequest = {
  name: '',
  email: '',
  algorithm: 'RSA',
  key_length: 4096,
  expire: '0',
};

export default function GenerateKeyDialog({ open, onOpenChange }: Props) {
  const queryClient = useQueryClient();
  const { t } = useTranslation('gpg');
  const { t: tc } = useTranslation('common');
  const [form, setForm] = useState<GenerateKeyRequest>({ ...defaultForm });

  const generateMutation = useMutation({
    mutationFn: gpgKeysApi.generate,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gpg-keys'] });
      toast.success(t('generate.keyGenerated'));
      setForm({ ...defaultForm });
      onOpenChange(false);
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const update = (field: keyof GenerateKeyRequest, value: string | number) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  const isEdDSA = form.algorithm === 'EdDSA';

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('generate.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="gen-name">{t('generate.name')}</Label>
            <Input
              id="gen-name"
              placeholder="Your Name"
              value={form.name}
              onChange={(e) => update('name', e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="gen-email">{t('generate.email')}</Label>
            <Input
              id="gen-email"
              type="email"
              placeholder="you@example.com"
              value={form.email}
              onChange={(e) => update('email', e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>{t('generate.algorithm')}</Label>
              <Select
                value={form.algorithm}
                onValueChange={(v) => update('algorithm', v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="RSA">{t('generate.rsa')}</SelectItem>
                  <SelectItem value="EdDSA">{t('generate.eddsa')}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>{t('generate.keyLength')}</Label>
              {isEdDSA ? (
                <Input value={t('generate.ed25519Length')} disabled />
              ) : (
                <Select
                  value={String(form.key_length)}
                  onValueChange={(v) => update('key_length', parseInt(v))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="2048">2048</SelectItem>
                    <SelectItem value="3072">3072</SelectItem>
                    <SelectItem value="4096">4096</SelectItem>
                  </SelectContent>
                </Select>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label>{t('generate.expiration')}</Label>
            <Select
              value={form.expire}
              onValueChange={(v) => update('expire', v)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">{t('generate.never')}</SelectItem>
                <SelectItem value="1y">{t('generate.oneYear')}</SelectItem>
                <SelectItem value="2y">{t('generate.twoYears')}</SelectItem>
                <SelectItem value="3y">{t('generate.threeYears')}</SelectItem>
                <SelectItem value="5y">{t('generate.fiveYears')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => { setForm({ ...defaultForm }); onOpenChange(false); }}>
            {tc('cancel')}
          </Button>
          <Button
            onClick={() => generateMutation.mutate(form)}
            disabled={!form.name || !form.email || generateMutation.isPending}
          >
            {generateMutation.isPending ? t('generate.generating') : t('generate.generateButton')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
