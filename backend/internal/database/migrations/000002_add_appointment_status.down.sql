-- Before this migration a cancelled appointment was deleted, and keeping
-- cancelled rows would break the restored unique constraint whenever their
-- slot was booked again.
DELETE FROM appointments
WHERE status <> 'confirmed';

DROP INDEX IF EXISTS unique_confirmed_appointment_slot;

ALTER TABLE appointments
    ADD CONSTRAINT unique_appointment_slot
        UNIQUE (appointment_date, start_time);

ALTER TABLE appointments
    DROP CONSTRAINT IF EXISTS appointment_cancelled_at_matches_status,
    DROP CONSTRAINT IF EXISTS appointment_status_valid,
    DROP COLUMN IF EXISTS cancelled_at,
    DROP COLUMN IF EXISTS status;
