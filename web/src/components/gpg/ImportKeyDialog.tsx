import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { gpgKeysApi } from '@/api/gpgkeys';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { toast } from 'sonner';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export default function ImportKeyDialog({ open, onOpenChange }: Props) {
  const queryClient = useQueryClient();
  const [keyData, setKeyData] = useState('');
  const [dragOver, setDragOver] = useState(false);

  const importMutation = useMutation({
    mutationFn: (data: string) => gpgKeysApi.importKey(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gpg-keys'] });
      toast.success('Key imported successfully');
      setKeyData('');
      onOpenChange(false);
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const handleFileRead = (file: File) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      setKeyData(e.target?.result as string);
    };
    reader.readAsText(file);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files[0];
    if (file) handleFileRead(file);
  };

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) handleFileRead(file);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Import GPG Key</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div
            className={`rounded-md border-2 border-dashed p-6 text-center transition-colors ${
              dragOver ? 'border-primary bg-primary/5' : 'border-muted-foreground/25'
            }`}
            onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
          >
            <p className="text-sm text-muted-foreground">
              Drag & drop a .asc or .gpg file here, or{' '}
              <label className="cursor-pointer text-primary underline">
                browse
                <input
                  type="file"
                  className="hidden"
                  accept=".asc,.gpg,.pub,.key"
                  onChange={handleFileInput}
                />
              </label>
            </p>
          </div>

          <div className="space-y-2">
            <Label>Or paste the armored key below</Label>
            <Textarea
              placeholder="-----BEGIN PGP PUBLIC KEY BLOCK-----&#10;...&#10;-----END PGP PUBLIC KEY BLOCK-----"
              value={keyData}
              onChange={(e) => setKeyData(e.target.value)}
              rows={8}
              className="font-mono text-xs"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => { setKeyData(''); setDragOver(false); onOpenChange(false); }}>
            Cancel
          </Button>
          <Button
            onClick={() => importMutation.mutate(keyData)}
            disabled={!keyData.trim() || importMutation.isPending}
          >
            {importMutation.isPending ? 'Importing...' : 'Import'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
