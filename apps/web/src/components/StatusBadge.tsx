const toneClasses: Record<string, string> = {
  ok: "border-emerald-200 bg-emerald-50 text-emerald-700",
  completed: "border-emerald-200 bg-emerald-50 text-emerald-700",
  scheduled: "border-sky-200 bg-sky-50 text-sky-700",
  needs_follow_up: "border-amber-200 bg-amber-50 text-amber-700"
};

interface StatusBadgeProps {
  value: string;
}

export function StatusBadge({ value }: StatusBadgeProps) {
  const classes =
    toneClasses[value] ?? "border-slate-200 bg-slate-100 text-slate-700";

  return (
    <span
      className={`inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium ${classes}`}
    >
      {value.replace(/_/g, " ")}
    </span>
  );
}
