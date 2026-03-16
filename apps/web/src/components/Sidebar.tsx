import type { ReactNode } from "react";
import {
  LayoutGrid,
  PhoneOutgoing,
  Clock,
  ChevronLeft,
  ChevronRight,
  Radio,
  Settings2
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { Page } from "../App";

interface SidebarProps {
  currentPage: Page;
  onNavigate: (page: Page) => void;
  collapsed: boolean;
  onToggleCollapse: () => void;
  patientSwitcher?: ReactNode;
}

const sections: { label: string; items: { id: Page; label: string; Icon: LucideIcon }[] }[] = [
  {
    label: "Overview",
    items: [
      { id: "dashboard", label: "Dashboard", Icon: LayoutGrid },
      { id: "settings", label: "Settings", Icon: Settings2 }
    ],
  },
  {
    label: "Calls",
    items: [
      { id: "schedule-call", label: "Schedule Call", Icon: PhoneOutgoing },
      { id: "recent-calls", label: "Recent Calls", Icon: Clock },
    ],
  },
];

export function Sidebar({ currentPage, onNavigate, collapsed, onToggleCollapse, patientSwitcher }: SidebarProps) {
  return (
    <aside
      style={{ width: collapsed ? 58 : 216, transition: "width 0.18s ease", flexShrink: 0 }}
      className="flex h-full flex-col overflow-hidden border-r border-slate-200/70 bg-white/68 backdrop-blur-xl"
    >
      {/* Logo */}
      <div
        className="flex items-center gap-2.5 border-b border-slate-200/70 px-4"
        style={{ height: 60 }}
      >
        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-xl bg-slate-950/92">
          <Radio size={14} className="text-white" />
        </div>
        {!collapsed && (
          <div className="min-w-0">
            <span className="block truncate text-sm font-semibold text-slate-950">Tether</span>
            <span className="block truncate text-[11px] text-slate-400">Care workspace</span>
          </div>
        )}
      </div>

      {/* Patient switcher */}
      {!collapsed && patientSwitcher && (
        <div className="border-b border-slate-200/70 px-3 py-3">{patientSwitcher}</div>
      )}

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        {sections.map((section) => (
          <div key={section.label} className="mb-5">
            {!collapsed && (
              <p className="mb-2 px-2 text-[11px] font-medium uppercase tracking-[0.14em] text-slate-400">
                {section.label}
              </p>
            )}
            {section.items.map(({ id, label, Icon }) => {
              const active = currentPage === id;
              return (
                <button
                  key={id}
                  onClick={() => onNavigate(id)}
                  title={collapsed ? label : undefined}
                  className={[
                    "mb-1 flex w-full items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm transition-colors",
                    active
                      ? "border border-slate-200 bg-slate-100 text-slate-950 font-medium"
                      : "border border-transparent text-slate-500 hover:bg-white/80 hover:text-slate-900",
                    collapsed ? "justify-center" : "",
                  ].join(" ")}
                >
                  <Icon
                    size={16}
                    className="flex-shrink-0"
                    strokeWidth={active ? 2.25 : 1.75}
                  />
                  {!collapsed && <span className="truncate">{label}</span>}
                </button>
              );
            })}
          </div>
        ))}
      </nav>

      {/* Bottom */}
      <div className="border-t border-slate-200/70 px-3 py-3">
        <button
          onClick={onToggleCollapse}
          title={collapsed ? "Expand" : "Collapse"}
          className="flex w-full items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm text-slate-400 transition-colors hover:bg-white/80 hover:text-slate-900"
        >
          {collapsed ? (
            <ChevronRight size={16} className="flex-shrink-0" strokeWidth={1.75} />
          ) : (
            <ChevronLeft size={16} className="flex-shrink-0" strokeWidth={1.75} />
          )}
          {!collapsed && <span>Collapse sidebar</span>}
        </button>
      </div>
    </aside>
  );
}
