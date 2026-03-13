import type { PropsWithChildren, ReactNode } from "react";

interface SectionProps extends PropsWithChildren {
  title: string;
  actions?: ReactNode;
}

export function Section({ title, actions, children }: SectionProps) {
  return (
    <section className="section-block">
      <header className="section-header">
        <h2>{title}</h2>
        {actions ? <div>{actions}</div> : null}
      </header>
      {children}
    </section>
  );
}
