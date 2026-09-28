import type { AvailableSlot } from "@/services/appointmentService";

interface TimeSlotGridProps {
  slots: AvailableSlot[];
  selectedSlot: AvailableSlot | null;
  onSelect: (slot: AvailableSlot) => void;
}

const slotClassName =
  "rounded-xl border px-3 py-3 text-sm font-semibold tabular-nums transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-600";

const idleSlotClassName =
  "border-slate-200 bg-white text-slate-700 hover:border-teal-600 hover:bg-teal-50 hover:text-teal-800";

const selectedSlotClassName =
  "border-teal-700 bg-teal-700 text-white shadow-sm";

export function TimeSlotGrid({
  slots,
  selectedSlot,
  onSelect,
}: TimeSlotGridProps) {
  return (
    <div className="mt-8">
      <h3 className="text-sm font-medium text-slate-700">Available times</h3>

      {slots.length === 0 ? (
        <div className="mt-3 rounded-xl border border-dashed border-slate-300 px-4 py-6 text-center">
          <p className="font-medium text-slate-700">
            No appointments are available for this date.
          </p>
          <p className="mt-1 text-sm text-slate-500">
            Please choose another date.
          </p>
        </div>
      ) : (
        <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
          {slots.map((slot) => {
            const isSelected = selectedSlot?.start_time === slot.start_time;

            return (
              <button
                key={slot.start_time}
                type="button"
                onClick={() => onSelect(slot)}
                aria-pressed={isSelected}
                aria-label={`${slot.start_time} to ${slot.end_time}`}
                className={`${slotClassName} ${
                  isSelected ? selectedSlotClassName : idleSlotClassName
                }`}
              >
                {slot.start_time}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
