import { PhoneOutgoing, Phone, Clock, BookOpen, Star, Heart, AlertOctagon } from "lucide-react";
import type { Patient } from "../api/contracts";

function initials(name: string) {
  return name.split(" ").map((w) => w[0]).join("").slice(0, 2).toUpperCase();
}

interface Props {
  patient: Patient | null;
  onScheduleCall: () => void;
}

export function Patients({ patient, onScheduleCall }: Props) {
  if (!patient) {
    return (
      <div className="flex h-full items-center justify-center text-gray-400 text-sm p-8">
        Loading patient...
      </div>
    );
  }

  const contextSections = [
    { label: "Routine Anchors", Icon: BookOpen, items: patient.routineAnchors },
    { label: "Favourite Topics", Icon: Star, items: patient.favoriteTopics },
    { label: "Calming Cues", Icon: Heart, items: patient.calmingCues },
    { label: "Topics to Avoid", Icon: AlertOctagon, items: patient.topicsToAvoid },
  ];

  return (
    <div className="p-8 max-w-2xl">
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">My Patient</h1>
          <p className="mt-0.5 text-sm text-gray-400">Profile and call context</p>
        </div>
        <button
          onClick={onScheduleCall}
          className="flex items-center gap-2 px-4 py-2 bg-gray-900 text-white text-sm font-medium rounded-lg hover:bg-gray-700 transition-colors"
        >
          <PhoneOutgoing size={15} strokeWidth={2.25} />
          Start Call
        </button>
      </div>

      {/* Patient card */}
      <div className="bg-white border border-gray-200 rounded-2xl p-6 mb-4">
        <div className="flex items-center gap-4 mb-5">
          <div className="w-14 h-14 rounded-full bg-gray-100 flex items-center justify-center flex-shrink-0">
            <span className="text-lg font-semibold text-gray-500">{initials(patient.displayName)}</span>
          </div>
          <div className="flex-1 min-w-0">
            <h2 className="text-xl font-semibold text-gray-900">{patient.displayName}</h2>
            {patient.preferredName && patient.preferredName !== patient.displayName && (
              <p className="text-sm text-gray-400">Goes by &ldquo;{patient.preferredName}&rdquo;</p>
            )}
          </div>
          <span
            className={`px-2.5 py-1 rounded-full text-xs font-medium ${
              patient.callingState === "active"
                ? "bg-green-50 text-green-700"
                : "bg-gray-100 text-gray-500"
            }`}
          >
            {patient.callingState === "active" ? "Active" : "Paused"}
          </span>
        </div>

        <div className="grid grid-cols-2 gap-3 text-sm">
          <div className="flex items-center gap-2 text-gray-600">
            <Phone size={14} className="text-gray-400 flex-shrink-0" strokeWidth={1.75} />
            {patient.phoneE164 ?? "No phone on file"}
          </div>
          <div className="flex items-center gap-2 text-gray-600">
            <Clock size={14} className="text-gray-400 flex-shrink-0" strokeWidth={1.75} />
            {patient.timezone}
          </div>
        </div>

        {patient.notes && (
          <p className="mt-4 text-sm text-gray-500 italic border-t border-gray-100 pt-4">
            &ldquo;{patient.notes}&rdquo;
          </p>
        )}

        {patient.callingState === "paused" && patient.pauseReason && (
          <div className="mt-4 bg-amber-50 border border-amber-100 rounded-lg px-3 py-2.5 text-sm text-amber-700 border-t border-gray-100 pt-4">
            Paused: {patient.pauseReason}
          </div>
        )}
      </div>

      {/* Call context */}
      <div className="grid grid-cols-2 gap-4">
        {contextSections.map(({ label, Icon, items }) => (
          <div key={label} className="bg-white border border-gray-200 rounded-xl p-4">
            <div className="flex items-center gap-1.5 mb-3">
              <Icon size={13} className="text-gray-400" strokeWidth={1.75} />
              <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider">{label}</h3>
            </div>
            {items.length > 0 ? (
              <div className="flex flex-wrap gap-1.5">
                {items.map((item) => (
                  <span
                    key={item}
                    className="text-xs bg-gray-50 border border-gray-100 rounded px-2 py-1 text-gray-600"
                  >
                    {item}
                  </span>
                ))}
              </div>
            ) : (
              <p className="text-xs text-gray-400 italic">None configured</p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
