-- Drop the MVP one-patient-per-caregiver unique constraint so caregivers can manage multiple patients.
alter table patients
	drop constraint if exists patients_primary_caregiver_id_key;
