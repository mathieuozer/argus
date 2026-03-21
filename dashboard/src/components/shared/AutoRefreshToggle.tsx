import { useState, useEffect, useRef } from 'react';

interface AutoRefreshToggleProps {
  onRefresh: () => void;
  intervalMs?: number;
}

function AutoRefreshToggle({ onRefresh, intervalMs = 10000 }: AutoRefreshToggleProps) {
  const [enabled, setEnabled] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (enabled) {
      timerRef.current = setInterval(onRefresh, intervalMs);
    }
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [enabled, onRefresh, intervalMs]);

  return (
    <button
      className={`btn btn-sm ${enabled ? 'btn-primary' : ''}`}
      onClick={() => setEnabled(!enabled)}
      title={enabled ? 'Disable auto-refresh' : 'Enable auto-refresh (10s)'}
    >
      <svg
        width="12"
        height="12"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className={enabled ? 'animate-spin-slow' : ''}
      >
        <polyline points="23 4 23 10 17 10" />
        <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
      </svg>
      {enabled ? 'Live' : 'Auto'}
    </button>
  );
}

export default AutoRefreshToggle;
