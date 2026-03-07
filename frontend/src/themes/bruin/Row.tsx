import type { RowProps, WidgetContainerProps } from "../../types/template";

export function BruinRow({ children }: RowProps) {
  return (
    <div className="dac-row">
      {children}
    </div>
  );
}

export function BruinWidgetContainer({ col, children }: WidgetContainerProps) {
  return (
    <div className="dac-widget-col" style={{ "--dac-col": col } as React.CSSProperties}>
      {children}
    </div>
  );
}
