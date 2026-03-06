import { useTranslation } from 'react-i18next';
import { Badge } from '@/components/ui/badge';

const statusVariant: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  pending: 'secondary',
  building: 'default',
  signing: 'default',
  publishing: 'default',
  verifying: 'default',
  success: 'outline',
  failed: 'destructive',
  cancelled: 'secondary',
};

export default function BuildStatusBadge({ status }: { status: string }) {
  const { t } = useTranslation('builds');
  const variant = statusVariant[status] || 'secondary';
  const isRunning = ['building', 'signing', 'publishing', 'verifying'].includes(status);

  return (
    <Badge variant={variant} className={isRunning ? 'animate-pulse' : ''}>
      {t(`status.${status}`, status)}
    </Badge>
  );
}
