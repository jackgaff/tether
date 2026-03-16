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
      style={{ width: collapsed ? 56 : 210, transition: "width 0.2s ease", flexShrink: 0 }}
      className="flex h-full flex-col overflow-hidden border-r border-white/70 bg-white/75 backdrop-blur-xl"
    >
      {/* Logo */}
      <div
        className="flex items-center gap-2.5 border-b border-slate-100 px-3.5"
        style={{ height: 56 }}
      >
        <div className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-lg bg-slate-950 shadow-[0_12px_24px_rgba(15,23,42,0.16)]">
          <Radio size={14} className="text-white" />
        </div>
        {!collapsed && (
          <span className="truncate text-sm font-semibold text-slate-950">Tether</span>
        )}
      </div>

      {/* Patient switcher */}
      {!collapsed && patientSwitcher && (
        <div className="border-b border-slate-100 px-2.5 py-2">{patientSwitcher}</div>
      )}

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-3 px-2.5">
        {sections.map((section) => (
          <div key={section.label} className="mb-4">
            {!collapsed && (
              <p className="mb-1 px-2 text-[11px] font-medium uppercase tracking-[0.16em] text-slate-400">
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
                    "mb-1 flex w-full items-center gap-2.5 rounded-2xl px-3 py-[10px] text-sm transition-all",
                    active
                      ? "bg-slate-950 text-white font-medium shadow-[0_18px_34px_rgba(15,23,42,0.18)]"
                      : "text-slate-500 hover:bg-white hover:text-slate-900 hover:shadow-[0_12px_24px_rgba(15,23,42,0.06)]",
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
      <div className="border-t border-slate-100 px-2.5 py-3">
        <button
          onClick={onToggleCollapse}
          title={collapsed ? "Expand" : "Collapse"}
          className="flex w-full items-center gap-2.5 rounded-2xl px-3 py-[10px] text-sm text-slate-400 transition-colors hover:bg-white hover:text-slate-900"
        >
          {collapsed ? (
            <ChevronRight size={16} className="flex-shrink-0" strokeWidth={1.75} />
          ) : (
            <ChevronLeft size={16} className="flex-shrink-0" strokeWidth={1.75} />
          )}
          {!collapsed && <span>Collapse</span>}
        </button>
      </div>
    </aside>
  );
}
