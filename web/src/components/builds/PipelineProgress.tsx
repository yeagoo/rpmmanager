import { useTranslation } from 'react-i18next';
import { Check, Loader2, X, Circle } from 'lucide-react';

const STAGES = ['build', 'sign', 'publish', 'verify'];

interface PipelineProgressProps {
  currentStage: string;
  status: string;
}

export default function PipelineProgress({ currentStage, status }: PipelineProgressProps) {
  const { t } = useTranslation('builds');
  const getStageState = (stage: string) => {
    if (status === 'success') return 'completed';
    if (status === 'failed') {
      const idx = STAGES.indexOf(currentStage);
      const stageIdx = STAGES.indexOf(stage);
      if (stageIdx < idx) return 'completed';
      if (stageIdx === idx) return 'failed';
      return 'pending';
    }
    if (status === 'cancelled' || status === 'pending') return 'pending';

    const currentIdx = STAGES.indexOf(currentStage);
    const stageIdx = STAGES.indexOf(stage);
    if (stageIdx < currentIdx) return 'completed';
    if (stageIdx === currentIdx) return 'running';
    return 'pending';
  };

  return (
    <div className="flex items-center gap-2">
      {STAGES.map((stage, i) => {
        const state = getStageState(stage);
        return (
          <div key={stage} className="flex items-center gap-2">
            {i > 0 && (
              <div className={`h-px w-8 ${state === 'pending' ? 'bg-muted' : 'bg-primary'}`} />
            )}
            <div className="flex items-center gap-1.5">
              {state === 'completed' && <Check className="h-4 w-4 text-green-500" />}
              {state === 'running' && <Loader2 className="h-4 w-4 animate-spin text-blue-500" />}
              {state === 'failed' && <X className="h-4 w-4 text-red-500" />}
              {state === 'pending' && <Circle className="h-4 w-4 text-muted-foreground" />}
              <span className={`text-xs ${
                state === 'running' ? 'font-medium text-blue-500' :
                state === 'completed' ? 'text-green-600' :
                state === 'failed' ? 'text-red-500' :
                'text-muted-foreground'
              }`}>
                {t(`pipeline.${stage}`)}
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
}
