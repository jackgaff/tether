interface AvatarProps {
  name: string;
  imageUrl?: string;
  size?: "sm" | "md" | "lg" | "xl";
  accent?: "sage" | "sky" | "amber";
  className?: string;
}

const sizeClasses = {
  sm: "h-10 w-10 text-sm",
  md: "h-14 w-14 text-base",
  lg: "h-16 w-16 text-lg",
  xl: "h-24 w-24 text-2xl"
};

const accentClasses = {
  sage: "from-emerald-100 via-white to-teal-100 ring-emerald-200/70",
  sky: "from-sky-100 via-white to-indigo-100 ring-sky-200/70",
  amber: "from-amber-100 via-white to-rose-100 ring-amber-200/70"
};

function initials(name: string) {
  return name
    .split(/\s+/)
    .map((part) => part[0] ?? "")
    .join("")
    .slice(0, 2)
    .toUpperCase();
}

export function Avatar({
  name,
  imageUrl,
  size = "md",
  accent = "sage",
  className = ""
}: AvatarProps) {
  const label = initials(name || "Patient");

  if (imageUrl) {
    return (
      <div
        className={[
          "overflow-hidden rounded-[28px] ring-4 shadow-[0_14px_32px_rgba(15,23,42,0.12)]",
          sizeClasses[size],
          className
        ].join(" ")}
      >
        <img
          src={imageUrl}
          alt={name}
          className="h-full w-full object-cover"
        />
      </div>
    );
  }

  return (
    <div
      className={[
        "flex items-center justify-center rounded-[28px] bg-gradient-to-br font-semibold text-slate-700 ring-4 shadow-[0_14px_32px_rgba(15,23,42,0.08)]",
        sizeClasses[size],
        accentClasses[accent],
        className
      ].join(" ")}
      aria-hidden="true"
    >
      {label}
    </div>
  );
}
