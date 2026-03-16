import type { ReactNode } from "react";
import {
  LayoutGrid,
  PhoneOutgoing,
  Clock,
  ChevronLeft,
  ChevronRight,
  Radio,
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
    items: [{ id: "dashboard", label: "Dashboard", Icon: LayoutGrid }],
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
      className="flex flex-col h-full bg-white border-r border-gray-200 overflow-hidden"
    >
      {/* Logo */}
      <div
        className="flex items-center gap-2.5 px-3.5 border-b border-gray-100"
        style={{ height: 56 }}
      >
        <div className="flex-shrink-0 w-7 h-7 rounded-lg bg-gray-900 flex items-center justify-center">
          <Radio size={14} className="text-white" />
        </div>
        {!collapsed && (
          <span className="text-sm font-semibold text-gray-900 truncate">Nova Echoes</span>
        )}
      </div>

      {/* Patient switcher */}
      {!collapsed && patientSwitcher && (
        <div className="px-2.5 py-2 border-b border-gray-100">{patientSwitcher}</div>
      )}

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-3 px-2.5">
        {sections.map((section) => (
          <div key={section.label} className="mb-4">
            {!collapsed && (
              <p className="px-2 mb-1 text-[11px] text-gray-400 font-medium">
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
                    "w-full flex items-center gap-2.5 px-2.5 py-[7px] rounded-md text-sm transition-colors mb-0.5",
                    active
                      ? "bg-gray-100 text-gray-900 font-medium"
                      : "text-gray-500 hover:bg-gray-50 hover:text-gray-800",
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
      <div className="border-t border-gray-100 px-2.5 py-3">
        <button
          onClick={onToggleCollapse}
          title={collapsed ? "Expand" : "Collapse"}
          className="w-full flex items-center gap-2.5 px-2.5 py-[7px] rounded-md text-sm text-gray-400 hover:bg-gray-50 hover:text-gray-700 transition-colors"
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
