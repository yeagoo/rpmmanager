import { useEffect, useRef, useState } from 'react';
import { apiClient } from '@/api/client';

const MAX_LOG_LINES = 10_000;

export function useBuildLogWebSocket(buildId: number | null) {
  const [lines, setLines] = useState<string[]>([]);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  // Reset lines when buildId changes
  useEffect(() => {
    setLines([]);
  }, [buildId]);

  useEffect(() => {
    if (!buildId) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const token = apiClient.getToken();
    const url = `${protocol}//${host}/api/builds/${buildId}/ws${token ? `?token=${encodeURIComponent(token)}` : ''}`;

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);

    ws.onmessage = (event) => {
      const text = typeof event.data === 'string' ? event.data : '';
      if (!text) return;
      const newLines = text.split('\n');
      setLines((prev) => {
        const updated = [...prev];
        if (updated.length > 0 && !prev[prev.length - 1].endsWith('\n') && newLines[0]) {
          updated[updated.length - 1] += newLines[0];
          const result = [...updated, ...newLines.slice(1)];
          return result.length > MAX_LOG_LINES ? result.slice(-MAX_LOG_LINES) : result;
        }
        const result = [...updated, ...newLines];
        return result.length > MAX_LOG_LINES ? result.slice(-MAX_LOG_LINES) : result;
      });
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [buildId]);

  return { lines, connected };
}
