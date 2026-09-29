import { expect, type Locator, type Page } from "@playwright/test";

export interface Customer {
  name: string;
  email: string;
  phone: string;
}

export interface Slot {
  start: string;
  end: string;
}

/**
 * Returns a bookable date (YYYY-MM-DD): the first weekday at least
 * daysAhead days from today. Each test uses its own date so tests running
 * in parallel never compete for the same slots, and dates a few days ahead
 * are in the future whatever the timezones of the browser and the API.
 */
export function bookableDate(daysAhead: number): string {
  const date = new Date();
  date.setDate(date.getDate() + daysAhead);

  while (date.getDay() === 0 || date.getDay() === 6) {
    date.setDate(date.getDate() + 1);
  }

  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");

  return `${year}-${month}-${day}`;
}

/** Formats a date the way the app displays it, e.g. "Monday, October 5, 2026". */
export function displayDate(date: string): string {
  const [year, month, day] = date.split("-").map(Number);

  return new Date(year, month - 1, day).toLocaleDateString("en-US", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

export function timeSlots(page: Page): Locator {
  return page.getByRole("group", { name: "Available times" });
}

export function slotButton(page: Page, slot: Slot): Locator {
  return timeSlots(page).getByRole("button", {
    name: `${slot.start} to ${slot.end}`,
    exact: true,
  });
}

/** Chooses a date and waits for its available times to load. */
export async function chooseDate(page: Page, date: string): Promise<void> {
  await page.getByLabel("Date").fill(date);

  await expect(timeSlots(page)).toBeVisible();
}

/**
 * Returns the slot of the first available time button. Tests pick the first
 * free slot instead of a fixed time so they can run again against the same
 * database, for example from `playwright test --ui`.
 */
export async function firstAvailableSlot(page: Page): Promise<Slot> {
  const label = await timeSlots(page)
    .getByRole("button")
    .first()
    .getAttribute("aria-label");

  const match = label?.match(/^(\d{2}:\d{2}) to (\d{2}:\d{2})$/);

  if (!match) {
    throw new Error(`Unexpected time slot label: ${label}`);
  }

  return { start: match[1], end: match[2] };
}

export async function fillCustomer(
  page: Page,
  customer: Customer,
): Promise<void> {
  await page.getByLabel("Full name").fill(customer.name);
  await page.getByLabel("Email").fill(customer.email);
  await page.getByLabel("Phone").fill(customer.phone);
}

export function confirmButton(page: Page): Locator {
  return page.getByRole("button", { name: "Confirm appointment" });
}
