import { useEffect, useState } from 'react';
import { sha256 } from '@/lib/pow/sha256';
import { Loader2, CheckCircle2, AlertCircle } from 'lucide-react';

interface Challenge {
  algorithm: string;
  challenge: string;
  maxNumber: number;
  salt: string;
  signature: string;
}

type Status = 'loading' | 'solving' | 'solved' | 'error';

interface PowCaptchaProps {
  onSolved: (payload: string) => void;
  onError?: (error: string) => void;
}

const BATCH_SIZE = 10000;
const API_BASE = import.meta.env.VITE_API_BASE || '';

async function fetchChallenge(): Promise<Challenge> {
  const resp = await fetch(`${API_BASE}/api/auth/challenge`);
  if (!resp.ok) {
    const data = await resp.json().catch(() => ({}));
    throw new Error(data.error || `HTTP ${resp.status}`);
  }
  return resp.json();
}

async function solveChallenge(
  challenge: Challenge,
  onProgress: (pct: number) => void,
  cancelled: () => boolean,
): Promise<string | null> {
  const { salt, challenge: target, maxNumber, algorithm, signature } = challenge;

  for (let start = 0; start <= maxNumber; start += BATCH_SIZE) {
    if (cancelled()) return null;

    const end = Math.min(start + BATCH_SIZE, maxNumber + 1);
    for (let n = start; n < end; n++) {
      const hash = sha256(salt + n.toString());
      if (hash === target) {
        return btoa(JSON.stringify({
          algorithm,
          challenge: target,
          number: n,
          salt,
          signature,
        }));
      }
    }

    onProgress(Math.min(99, Math.floor((end / maxNumber) * 100)));
    // Yield to UI thread
    await new Promise(resolve => setTimeout(resolve, 0));
  }

  return null;
}

export default function PowCaptcha({ onSolved, onError }: PowCaptchaProps) {
  const [status, setStatus] = useState<Status>('loading');
  const [progress, setProgress] = useState(0);

  // Component is remounted via key change on retry, so callbacks are stable at mount.
  useEffect(() => {
    let active = true;

    (async () => {
      try {
        const challenge = await fetchChallenge();
        if (!active) return;

        setStatus('solving');

        const payload = await solveChallenge(
          challenge,
          (pct) => { if (active) setProgress(pct); },
          () => !active,
        );

        if (!active) return;

        if (payload) {
          setStatus('solved');
          setProgress(100);
          onSolved(payload);
        } else {
          setStatus('error');
          onError?.('Failed to solve challenge');
        }
      } catch (err) {
        if (!active) return;
        setStatus('error');
        onError?.(err instanceof Error ? err.message : 'Failed to load challenge');
      }
    })();

    return () => {
      active = false;
    };
  }, []);

  return (
    <div className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
      {status === 'loading' && (
        <>
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          <span className="text-muted-foreground">Loading security challenge...</span>
        </>
      )}
      {status === 'solving' && (
        <>
          <Loader2 className="h-4 w-4 animate-spin text-blue-500" />
          <div className="flex flex-1 items-center gap-2">
            <span className="text-muted-foreground">Verifying...</span>
            <div className="h-1.5 flex-1 rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-blue-500 transition-all duration-200"
                style={{ width: `${progress}%` }}
              />
            </div>
            <span className="text-xs text-muted-foreground">{progress}%</span>
          </div>
        </>
      )}
      {status === 'solved' && (
        <>
          <CheckCircle2 className="h-4 w-4 text-green-500" />
          <span className="text-green-600">Verified</span>
        </>
      )}
      {status === 'error' && (
        <>
          <AlertCircle className="h-4 w-4 text-destructive" />
          <span className="text-destructive">Verification failed</span>
        </>
      )}
    </div>
  );
}
