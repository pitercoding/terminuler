import type { AvailableSlot } from "@/services/appointmentService";

interface TimeSlotGridProps {
  slots: AvailableSlot[];
  selectedSlot: AvailableSlot | null;
  disabled: boolean;
  onSelect: (slot: AvailableSlot) => void;
}

const headingId = "available-times-heading";

const gridClassName = "mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4";

const slotClassName =
  "rounded-xl border px-3 py-3 text-sm font-semibold tabular-nums transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-600 disabled:cursor-not-allowed disabled:opacity-60";

const idleSlotClassName =
  "border-slate-200 bg-white text-slate-700 enabled:hover:border-teal-600 enabled:hover:bg-teal-50 enabled:hover:text-teal-800";

const selectedSlotClassName =
  "border-teal-700 bg-teal-700 text-white shadow-sm";

function Heading() {
  return (
    <h3 id={headingId} className="text-sm font-medium text-slate-700">
      Available times
    </h3>
  );
}

export function TimeSlotGrid({
  slots,
  selectedSlot,
  disabled,
  onSelect,
}: TimeSlotGridProps) {
  return (
    <div className="mt-8">
      <Heading />

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
        <div role="group" aria-labelledby={headingId} className={gridClassName}>
          {slots.map((slot) => {
            const isSelected = selectedSlot?.start_time === slot.start_time;

            return (
              <button
                key={slot.start_time}
                type="button"
                onClick={() => onSelect(slot)}
                disabled={disabled}
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

/** Placeholder shown while the available times are being loaded. */
export function TimeSlotGridSkeleton() {
  return (
    <div className="mt-8" role="status">
      <Heading />

      <span className="sr-only">Loading available times…</span>

      <div aria-hidden="true" className={gridClassName}>
        {Array.from({ length: 8 }, (_, index) => (
          <div
            key={index}
            className="h-11.5 animate-pulse rounded-xl bg-slate-100"
          />
        ))}
      </div>
    </div>
  );
}
