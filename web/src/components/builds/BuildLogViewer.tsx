import { useEffect, useRef } from 'react';

interface BuildLogViewerProps {
  lines: string[];
  connected: boolean;
}

export default function BuildLogViewer({ lines, connected }: BuildLogViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const autoScrollRef = useRef(true);

  useEffect(() => {
    if (autoScrollRef.current && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [lines]);

  const handleScroll = () => {
    if (!containerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    autoScrollRef.current = scrollHeight - scrollTop - clientHeight < 50;
  };

  return (
    <div className="relative">
      <div className="absolute right-2 top-2 z-10">
        {connected && (
          <span className="flex items-center gap-1 text-xs text-green-500">
            <span className="h-2 w-2 rounded-full bg-green-500 animate-pulse" />
            Live
          </span>
        )}
      </div>
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="h-[500px] overflow-auto rounded-md border bg-zinc-950 p-4 font-mono text-sm text-zinc-100"
      >
        {lines.length === 0 ? (
          <span className="text-zinc-500">Waiting for output...</span>
        ) : (
          lines.map((line, i) => (
            <div key={i} className={
              line.startsWith('===') ? 'text-cyan-400 font-bold mt-2' :
              line.includes('FAILED') || line.includes('Error') ? 'text-red-400' :
              line.includes('Warning') ? 'text-yellow-400' :
              ''
            }>
              {line || '\u00A0'}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
