import type { PropsWithChildren, ReactNode } from "react";

interface SectionCardProps extends PropsWithChildren {
  eyebrow: string;
  title: string;
  action?: ReactNode;
}

export function SectionCard({
  eyebrow,
  title,
  action,
  children
}: SectionCardProps) {
  return (
    <section className="rounded-3xl border border-white/10 bg-white/70 p-6 shadow-[0_18px_80px_rgba(15,23,42,0.12)] backdrop-blur md:p-7">
      <header className="flex flex-col gap-4 border-b border-slate-200/70 pb-5 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-500">
            {eyebrow}
          </p>
          <h2 className="mt-2 text-xl font-semibold tracking-tight text-slate-950">
            {title}
          </h2>
        </div>
        {action ? <div className="shrink-0">{action}</div> : null}
      </header>
      <div className="pt-5">{children}</div>
    </section>
  );
}

