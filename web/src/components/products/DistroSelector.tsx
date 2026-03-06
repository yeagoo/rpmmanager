import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { productsApi, type DistroInfo } from '@/api/products';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';

interface DistroSelectorProps {
  value: string[];
  onChange: (distros: string[]) => void;
}

const PRODUCT_LINE_LABELS: Record<string, string> = {
  el8: 'EL 8',
  el9: 'EL 9',
  el10: 'EL 10',
  al2023: 'Amazon Linux 2023',
  fedora: 'Fedora',
  oe22: 'openEuler 22',
  oe24: 'openEuler 24',
};

export default function DistroSelector({ value, onChange }: DistroSelectorProps) {
  const { t } = useTranslation('products');
  const { data } = useQuery<DistroInfo>({
    queryKey: ['distros'],
    queryFn: productsApi.getDistros,
  });

  if (!data) return <div className="text-muted-foreground text-sm">{t('distroSelector.loadingDistros')}</div>;

  const toggleDistro = (dv: string) => {
    if (value.includes(dv)) {
      onChange(value.filter((d) => d !== dv));
    } else {
      onChange([...value, dv]);
    }
  };

  const toggleProductLine = (plId: string) => {
    const distros = (data.distro_groups[plId] || []).map((d) => `${d.distro}:${d.version}`);
    const allSelected = distros.every((d) => value.includes(d));
    if (allSelected) {
      onChange(value.filter((d) => !distros.includes(d)));
    } else {
      const newValue = [...value];
      for (const d of distros) {
        if (!newValue.includes(d)) newValue.push(d);
      }
      onChange(newValue);
    }
  };

  const selectAll = () => {
    onChange(data.all_distros);
  };

  const selectNone = () => {
    onChange([]);
  };

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <button type="button" onClick={selectAll} className="text-xs text-primary hover:underline">
          {t('distroSelector.selectAll')}
        </button>
        <span className="text-xs text-muted-foreground">|</span>
        <button type="button" onClick={selectNone} className="text-xs text-primary hover:underline">
          {t('distroSelector.clearAll')}
        </button>
        <span className="ml-auto text-xs text-muted-foreground">
          {t('distroSelector.selectedCount', { selected: value.length, total: data.all_distros.length })}
        </span>
      </div>

      {data.product_lines.map((pl) => {
        const distros = data.distro_groups[pl.id] || [];
        const distroStrings = distros.map((d) => `${d.distro}:${d.version}`);
        const selectedCount = distroStrings.filter((d) => value.includes(d)).length;
        const allSelected = distros.length > 0 && selectedCount === distros.length;

        return (
          <div key={pl.id} className="rounded-md border p-3">
            <div className="mb-2 flex items-center gap-2">
              <Checkbox
                checked={allSelected}
                onCheckedChange={() => toggleProductLine(pl.id)}
              />
              <span className="text-sm font-medium">
                {PRODUCT_LINE_LABELS[pl.id] || pl.id}
              </span>
              <Badge variant="outline" className="text-xs">
                {pl.compression}
              </Badge>
              <span className="text-xs text-muted-foreground">
                ({selectedCount}/{distros.length})
              </span>
            </div>
            <div className="ml-6 flex flex-wrap gap-x-4 gap-y-1">
              {distros
                .sort((a, b) => `${a.distro}:${a.version}`.localeCompare(`${b.distro}:${b.version}`))
                .map((d) => {
                  const dv = `${d.distro}:${d.version}`;
                  return (
                    <label key={dv} className="flex items-center gap-1.5 text-sm">
                      <Checkbox
                        checked={value.includes(dv)}
                        onCheckedChange={() => toggleDistro(dv)}
                      />
                      {d.distro}:{d.version}
                    </label>
                  );
                })}
            </div>
          </div>
        );
      })}
    </div>
  );
}
