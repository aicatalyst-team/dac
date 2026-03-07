import type { ReactNode } from "react";

export function Row({ children }: { children: ReactNode }) {
  return (
    <div className="dac-row">
      {children}
    </div>
  );
}

export function WidgetContainer({
  col,
  children,
}: {
  col: number;
  children: ReactNode;
}) {
  return (
    <div className="dac-widget-col" style={{ "--dac-col": col } as React.CSSProperties}>
      {children}
    </div>
  );
}
