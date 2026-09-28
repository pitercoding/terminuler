import { inputClassName, labelClassName } from "@/components/styles";

interface DateSelectorProps {
  date: string;
  onDateChange: (date: string) => void;
}

export function DateSelector({ date, onDateChange }: DateSelectorProps) {
  return (
    <div>
      <label htmlFor="appointment-date" className={labelClassName}>
        Date
      </label>

      <input
        id="appointment-date"
        type="date"
        value={date}
        min={new Date().toISOString().split("T")[0]}
        onChange={(event) => onDateChange(event.target.value)}
        className={`${inputClassName} sm:max-w-xs`}
      />

      {!date && (
        <p className="mt-2 text-sm text-slate-500">
          Choose a date to see the available times.
        </p>
      )}
    </div>
  );
}
