import type { AvailableSlot } from "@/services/appointmentService";

interface TimeSlotGridProps {
  slots: AvailableSlot[];
  selectedSlot: AvailableSlot | null;
  onSelect: (slot: AvailableSlot) => void;
}

export function TimeSlotGrid({
  slots,
  selectedSlot,
  onSelect,
}: TimeSlotGridProps) {
  return (
    <section>
      <h2>Available times</h2>

      {slots.length === 0 ? (
        <p>No appointments available for this date.</p>
      ) : (
        <div>
          {slots.map((slot) => {
            const isSelected = selectedSlot?.start_time === slot.start_time;

            return (
              <button
                key={slot.start_time}
                type="button"
                onClick={() => onSelect(slot)}
                aria-pressed={isSelected}
              >
                {slot.start_time} - {slot.end_time}
              </button>
            );
          })}
        </div>
      )}
    </section>
  );
}
