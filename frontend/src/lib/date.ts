/**
 * Formats a calendar date (YYYY-MM-DD) for display, e.g.
 * "Monday, September 28, 2026". The date is built from its parts instead of
 * parsed as a string, because new Date("2026-09-28") is UTC midnight and
 * shows the previous day in timezones behind UTC.
 */
export function formatDisplayDate(date: string): string {
  const [year, month, day] = date.split("-").map(Number);

  return new Date(year, month - 1, day).toLocaleDateString("en-US", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

/**
 * Returns today's date in the browser's timezone as YYYY-MM-DD.
 * toISOString() is not used because it returns the date in UTC.
 */
export function getTodayDate(): string {
  const today = new Date();

  const year = today.getFullYear();
  const month = String(today.getMonth() + 1).padStart(2, "0");
  const day = String(today.getDate()).padStart(2, "0");

  return `${year}-${month}-${day}`;
}
