import { useState, useRef, useCallback, useLayoutEffect, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

interface TooltipProps {
  children: ReactNode;
  content: string;
  side?: 'top' | 'bottom';
  followCursor?: boolean;
}

const OFFSET = 12;
const MARGIN = 8;

export default function Tooltip({ children, content, side = 'bottom', followCursor }: TooltipProps) {
  const [pos, setPos] = useState<{ x: number; y: number } | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const visibleRef = useRef(false);
  const tooltipRef = useRef<HTMLDivElement>(null);

  // Render at (0,0) hidden → measure natural size → clamp → show
  useLayoutEffect(() => {
    const el = tooltipRef.current;
    if (!el || !pos) return;

    const w = el.offsetWidth;
    const h = el.offsetHeight;
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    let x = pos.x;
    let y = pos.y;
    if (x + w > vw - MARGIN) x = pos.x - w - OFFSET * 2;
    if (x < MARGIN) x = MARGIN;
    if (y + h > vh - MARGIN) y = pos.y - h - OFFSET * 2;
    if (y < MARGIN) y = MARGIN;

    el.style.left = `${x}px`;
    el.style.top = `${y}px`;
    el.style.visibility = 'visible';
  }, [pos]);

  const show = useCallback((e: React.MouseEvent) => {
    if (followCursor) {
      const x = e.clientX + OFFSET;
      const y = e.clientY + OFFSET;
      timerRef.current = setTimeout(() => {
        visibleRef.current = true;
        setPos({ x, y });
      }, 200);
    } else {
      const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
      timerRef.current = setTimeout(() => {
        setPos({
          x: rect.left,
          y: side === 'top' ? rect.top - 4 : rect.bottom + 4,
        });
      }, 200);
    }
  }, [side, followCursor]);

  const move = useCallback((e: React.MouseEvent) => {
    if (visibleRef.current) {
      setPos({ x: e.clientX + OFFSET, y: e.clientY + OFFSET });
    }
  }, []);

  const hide = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    visibleRef.current = false;
    setPos(null);
  }, []);

  return (
    <>
      <span onMouseEnter={show} onMouseLeave={hide} onMouseMove={followCursor ? move : undefined}>
        {children}
      </span>
      {pos && createPortal(
        <div
          ref={tooltipRef}
          className="ss-tooltip fixed z-[9999] max-w-sm whitespace-pre-line bg-pencil text-paper text-xs px-2.5 py-1.5 shadow-lg pointer-events-none animate-fade-in rounded-[var(--radius-sm)]"
          style={{ left: 0, top: 0, visibility: 'hidden' }}
        >
          {content}
        </div>,
        document.body,
      )}
    </>
  );
}
