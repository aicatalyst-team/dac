import { useCallback, useRef } from "react";

interface ResizeHandleProps {
  side: "left" | "right";
  onResize: (delta: number) => void;
}

export function ResizeHandle({ side, onResize }: ResizeHandleProps) {
  const startXRef = useRef(0);

  const onPointerDown = useCallback(
    (e: React.PointerEvent) => {
      e.preventDefault();
      startXRef.current = e.clientX;
      const el = e.currentTarget as HTMLElement;
      el.setPointerCapture(e.pointerId);

      const onMove = (ev: PointerEvent) => {
        const dx = ev.clientX - startXRef.current;
        startXRef.current = ev.clientX;
        // For a left-edge handle, dragging right shrinks; for right-edge, dragging right grows.
        onResize(side === "left" ? -dx : dx);
      };

      const onUp = () => {
        el.removeEventListener("pointermove", onMove);
        el.removeEventListener("pointerup", onUp);
      };

      el.addEventListener("pointermove", onMove);
      el.addEventListener("pointerup", onUp);
    },
    [side, onResize],
  );

  return (
    <div
      onPointerDown={onPointerDown}
      className={`resize-handle ${side === "left" ? "resize-handle-left" : "resize-handle-right"}`}
    />
  );
}
