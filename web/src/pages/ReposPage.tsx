import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { reposApi, type RepoInfo, type RepoEntry } from '@/api/repos';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Folder, File, ChevronRight, RotateCcw, HardDrive } from 'lucide-react';
import { toast } from 'sonner';

function formatSize(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

function TreeNode({ entry, product, depth = 0 }: { entry: RepoEntry; product: string; depth?: number }) {
  const [expanded, setExpanded] = useState(depth < 1);

  return (
    <div>
      <div
        className={`flex items-center gap-2 py-1 px-2 hover:bg-muted/50 rounded text-sm cursor-pointer`}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        onClick={() => entry.is_dir && setExpanded(!expanded)}
      >
        {entry.is_dir ? (
          <>
            <ChevronRight className={`h-3 w-3 transition-transform ${expanded ? 'rotate-90' : ''}`} />
            <Folder className="h-4 w-4 text-blue-500" />
          </>
        ) : (
          <>
            <span className="w-3" />
            <File className="h-4 w-4 text-muted-foreground" />
          </>
        )}
        <span className="flex-1">{entry.name}</span>
        {!entry.is_dir && (
          <span className="text-xs text-muted-foreground">{formatSize(entry.size)}</span>
        )}
      </div>
      {entry.is_dir && expanded && entry.items?.map((child) => (
        <TreeNode key={child.path} entry={child} product={product} depth={depth + 1} />
      ))}
    </div>
  );
}

export default function ReposPage() {
  const queryClient = useQueryClient();
  const [selectedProduct, setSelectedProduct] = useState<string | null>(null);
  const [rollbackProduct, setRollbackProduct] = useState<string | null>(null);

  const { data: repos, isLoading } = useQuery({
    queryKey: ['repos'],
    queryFn: reposApi.list,
  });

  const { data: tree } = useQuery({
    queryKey: ['repo-tree', selectedProduct],
    queryFn: () => reposApi.getTree(selectedProduct!, undefined, 4),
    enabled: !!selectedProduct,
  });

  const { data: rollbacks } = useQuery({
    queryKey: ['rollbacks', rollbackProduct],
    queryFn: () => reposApi.listRollbacks(rollbackProduct!),
    enabled: !!rollbackProduct,
  });

  const rollbackMutation = useMutation({
    mutationFn: ({ product, snapshot }: { product: string; snapshot: string }) =>
      reposApi.rollback(product, snapshot),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repos'] });
      queryClient.invalidateQueries({ queryKey: ['repo-tree'] });
      toast.success('Rollback completed');
      setRollbackProduct(null);
    },
    onError: (err: Error) => toast.error(err.message),
  });

  return (
    <div className="space-y-4">
      <h1 className="text-3xl font-bold">Repositories</h1>

      {isLoading ? (
        <p className="text-muted-foreground">Loading...</p>
      ) : !repos?.length ? (
        <div className="rounded-md border border-dashed p-8 text-center">
          <p className="text-muted-foreground">No repositories found. Build a product to create repositories.</p>
        </div>
      ) : (
        <div className="grid gap-6 lg:grid-cols-3">
          {/* Repo list */}
          <div className="space-y-2">
            {repos.map((repo: RepoInfo) => (
              <Card
                key={repo.product}
                className={`cursor-pointer transition-colors ${
                  selectedProduct === repo.product ? 'border-primary' : ''
                }`}
                onClick={() => setSelectedProduct(repo.product)}
              >
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <HardDrive className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">{repo.product}</span>
                    </div>
                    {repo.has_repomd && <Badge variant="secondary">Active</Badge>}
                  </div>
                  <div className="mt-2 flex gap-3 text-xs text-muted-foreground">
                    <span>{repo.rpm_count} RPMs</span>
                    <span>{formatSize(repo.total_size)}</span>
                    <span>{repo.file_count} files</span>
                  </div>
                  <div className="mt-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        setRollbackProduct(repo.product);
                      }}
                    >
                      <RotateCcw className="mr-1 h-3 w-3" />
                      Rollback
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          {/* File tree */}
          <div className="lg:col-span-2">
            {selectedProduct ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{selectedProduct}/</CardTitle>
                </CardHeader>
                <CardContent>
                  {tree?.length ? (
                    <div className="max-h-[600px] overflow-auto rounded border p-2 font-mono text-sm">
                      {tree.map((entry) => (
                        <TreeNode key={entry.path} entry={entry} product={selectedProduct} />
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">Empty directory</p>
                  )}
                </CardContent>
              </Card>
            ) : (
              <div className="flex h-64 items-center justify-center rounded-md border border-dashed">
                <p className="text-muted-foreground">Select a repository to browse files</p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Rollback dialog */}
      <Dialog open={!!rollbackProduct} onOpenChange={(open) => !open && setRollbackProduct(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rollback: {rollbackProduct}</DialogTitle>
          </DialogHeader>
          {!rollbacks?.length ? (
            <p className="text-sm text-muted-foreground">No rollback snapshots available.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Snapshot</TableHead>
                  <TableHead className="w-[100px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {rollbacks.map((snapshot: string) => (
                  <TableRow key={snapshot}>
                    <TableCell className="font-mono text-sm">{snapshot}</TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          if (confirm(`Rollback ${rollbackProduct} to ${snapshot}?`)) {
                            rollbackMutation.mutate({ product: rollbackProduct!, snapshot });
                          }
                        }}
                      >
                        Restore
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setRollbackProduct(null)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
