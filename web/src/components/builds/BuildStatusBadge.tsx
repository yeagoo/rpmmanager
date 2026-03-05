import { Badge } from '@/components/ui/badge';

const statusConfig: Record<string, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }> = {
  pending: { label: 'Pending', variant: 'secondary' },
  building: { label: 'Building', variant: 'default' },
  signing: { label: 'Signing', variant: 'default' },
  publishing: { label: 'Publishing', variant: 'default' },
  verifying: { label: 'Verifying', variant: 'default' },
  success: { label: 'Success', variant: 'outline' },
  failed: { label: 'Failed', variant: 'destructive' },
  cancelled: { label: 'Cancelled', variant: 'secondary' },
};

export default function BuildStatusBadge({ status }: { status: string }) {
  const cfg = statusConfig[status] || { label: status, variant: 'secondary' as const };
  const isRunning = ['building', 'signing', 'publishing', 'verifying'].includes(status);

  return (
    <Badge variant={cfg.variant} className={isRunning ? 'animate-pulse' : ''}>
      {cfg.label}
    </Badge>
  );
}
