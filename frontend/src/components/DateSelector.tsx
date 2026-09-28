import { useSyncExternalStore } from "react";
import { inputClassName, labelClassName } from "@/components/styles";
import { getTodayDate } from "@/lib/date";

interface DateSelectorProps {
  date: string;
  disabled: boolean;
  onDateChange: (date: string) => void;
}

function subscribe() {
  return () => {};
}

/**
 * Today's date for the date picker's min attribute. The page is prerendered
 * at build time, so the server snapshot is empty and the browser computes
 * the date itself; the backend still rejects past dates either way.
 */
function useTodayDate(): string {
  return useSyncExternalStore(subscribe, getTodayDate, () => "");
}

export function DateSelector({
  date,
  disabled,
  onDateChange,
}: DateSelectorProps) {
  const today = useTodayDate();

  return (
    <div>
      <label htmlFor="appointment-date" className={labelClassName}>
        Date
      </label>

      <input
        id="appointment-date"
        type="date"
        value={date}
        min={today || undefined}
        disabled={disabled}
        aria-describedby={date ? undefined : "appointment-date-hint"}
        onChange={(event) => onDateChange(event.target.value)}
        className={`${inputClassName} disabled:cursor-not-allowed disabled:bg-slate-100 sm:max-w-xs`}
      />

      {!date && (
        <p id="appointment-date-hint" className="mt-2 text-sm text-slate-500">
          Choose a date to see the available times.
        </p>
      )}
    </div>
  );
}
