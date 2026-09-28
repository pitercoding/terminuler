interface DateSelectorProps {
  date: string;
  isSearching: boolean;
  onDateChange: (date: string) => void;
  onSearch: () => void;
}

export function DateSelector({
  date,
  isSearching,
  onDateChange,
  onSearch,
}: DateSelectorProps) {
  return (
    <div>
      <label htmlFor="appointment-date">Select a date</label>

      <input
        id="appointment-date"
        type="date"
        value={date}
        min={new Date().toISOString().split("T")[0]}
        onChange={(event) => onDateChange(event.target.value)}
      />

      <button type="button" onClick={onSearch} disabled={isSearching}>
        {isSearching ? "Searching..." : "Search availability"}
      </button>
    </div>
  );
}
