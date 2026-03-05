import { useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { buildsApi } from '@/api/builds';
import { useBuildLogWebSocket } from '@/hooks/useWebSocket';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ArrowLeft, StopCircle } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';
import BuildStatusBadge from '@/components/builds/BuildStatusBadge';
import PipelineProgress from '@/components/builds/PipelineProgress';
import BuildLogViewer from '@/components/builds/BuildLogViewer';

export default function BuildDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const buildId = Number(id);
  const isValidId = !isNaN(buildId) && buildId > 0;

  const { data: build } = useQuery({
    queryKey: ['build', id],
    queryFn: () => buildsApi.get(buildId),
    enabled: isValidId,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status && ['success', 'failed', 'cancelled'].includes(status)) return false;
      return 2000;
    },
  });

  const isRunning = build && ['pending', 'building', 'signing', 'publishing', 'verifying'].includes(build.status);
  const isFinished = build && ['success', 'failed', 'cancelled'].includes(build.status);

  const { lines, connected } = useBuildLogWebSocket(isRunning ? buildId : null);

  // Fetch log content for finished builds
  const { data: logContent } = useQuery({
    queryKey: ['build-log', buildId],
    queryFn: () => buildsApi.getLog(buildId),
    enabled: isValidId && !!isFinished,
  });

  const cancelMutation = useMutation({
    mutationFn: () => buildsApi.cancel(buildId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['build', id] });
      toast.success('Build cancelled');
    },
    onError: (err: Error) => toast.error(err.message),
  });

  // Combine log sources: websocket lines for running, fetched log for finished
  const displayLines = isRunning ? lines : (logContent ? logContent.split('\n') : lines);

  if (!isValidId) return <p className="text-destructive">Invalid build ID</p>;
  if (!build) return <p className="text-muted-foreground">Loading...</p>;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="sm" onClick={() => navigate('/builds')}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <h1 className="text-3xl font-bold">Build #{build.id}</h1>
        <BuildStatusBadge status={build.status} />
        {isRunning && (
          <Button
            variant="destructive"
            size="sm"
            onClick={() => cancelMutation.mutate()}
            disabled={cancelMutation.isPending}
          >
            <StopCircle className="mr-2 h-4 w-4" />
            Cancel
          </Button>
        )}
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Product</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-lg font-bold">{build.product_display_name}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Version</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-lg font-bold font-mono">{build.version}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">RPMs Built</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-lg font-bold">{build.rpm_count}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Duration</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-lg font-bold">
              {build.duration_seconds > 0 ? `${build.duration_seconds}s` : '-'}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Pipeline Progress</CardTitle>
        </CardHeader>
        <CardContent>
          <PipelineProgress currentStage={build.current_stage} status={build.status} />
        </CardContent>
      </Card>

      {build.error_message && (
        <Card className="border-destructive">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-destructive">Error</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="text-sm text-destructive whitespace-pre-wrap">{build.error_message}</pre>
          </CardContent>
        </Card>
      )}

      <div>
        <h2 className="mb-2 text-lg font-semibold">Build Log</h2>
        <BuildLogViewer lines={displayLines} connected={connected} />
      </div>
    </div>
  );
}
