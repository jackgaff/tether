import { ApiError } from "../api/client";
import { createPatient, updateConsent } from "../api/admin";
import type { Patient, PatientInput } from "../api/contracts";
import { PatientProfileForm } from "../components/patient/PatientProfileForm";

interface Props {
  caregiverId: string;
  caregiverBootstrapError?: string | null;
  canRetryCaregiverBootstrap?: boolean;
  onRetryCaregiverBootstrap?: () => void;
  onInvalidCaregiverReference?: () => void;
  onCreated: (patient: Patient) => void;
  onCancel: () => void;
}

export function CreatePatient({
  caregiverId,
  caregiverBootstrapError,
  canRetryCaregiverBootstrap = false,
  onRetryCaregiverBootstrap,
  onInvalidCaregiverReference,
  onCreated,
  onCancel
}: Props) {
  async function handleSubmit(input: PatientInput) {
    try {
      const patient = await createPatient(input);

      await updateConsent(patient.id, {
        outboundCallStatus: "granted",
        transcriptStorageStatus: "granted",
        notes: "Consent granted during patient setup."
      });

      onCreated(patient);
    } catch (err) {
      if (
        err instanceof ApiError &&
        err.code === "validation_error" &&
        err.message.includes("primaryCaregiverId must reference an existing caregiver.")
      ) {
        onInvalidCaregiverReference?.();
        throw new Error("The caregiver profile expired. Refreshing it now, then try saving again.");
      }
      throw err;
    }
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {!caregiverId && (
        <div className="mb-6 rounded-[28px] border border-amber-200 bg-amber-50/90 px-5 py-4 text-sm text-amber-900 shadow-[0_20px_40px_rgba(217,119,6,0.12)]">
          <p>
            {caregiverBootstrapError ??
              "Setting up the caregiver profile for this local demo. The patient form will work in a moment."}
          </p>
          {canRetryCaregiverBootstrap && onRetryCaregiverBootstrap && (
            <button
              type="button"
              onClick={onRetryCaregiverBootstrap}
              className="mt-3 text-sm font-semibold text-amber-950 underline underline-offset-4"
            >
              Retry caregiver setup
            </button>
          )}
        </div>
      )}

      <PatientProfileForm
        mode="create"
        caregiverId={caregiverId}
        title="Create a patient profile that feels ready for real care"
        subtitle="Add the basics, seed the memory profile, and upload a profile photo so the dashboard feels human from the first call onward."
        submitLabel="Create patient"
        onSubmit={handleSubmit}
        onCancel={onCancel}
      />
    </div>
  );
}
