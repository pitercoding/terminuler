-- Existing appointments were all active, so the default marks them confirmed.
ALTER TABLE appointments
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'confirmed',
    ADD COLUMN cancelled_at TIMESTAMPTZ;

ALTER TABLE appointments
    ADD CONSTRAINT appointment_status_valid
        CHECK (status IN ('confirmed', 'cancelled')),
    ADD CONSTRAINT appointment_cancelled_at_matches_status
        CHECK ((status = 'cancelled') = (cancelled_at IS NOT NULL));

-- Only confirmed appointments hold their slot, so a cancelled appointment
-- stays stored without blocking a new booking at the same date and time.
ALTER TABLE appointments
    DROP CONSTRAINT unique_appointment_slot;

CREATE UNIQUE INDEX unique_confirmed_appointment_slot
    ON appointments (appointment_date, start_time)
    WHERE status = 'confirmed';
