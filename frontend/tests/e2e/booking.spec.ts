import { expect, test } from "@playwright/test";
import {
  bookableDate,
  chooseDate,
  confirmButton,
  displayDate,
  fillCustomer,
  firstAvailableSlot,
  slotButton,
} from "./helpers";

const customer = {
  name: "Jane Doe",
  email: "jane@example.com",
  phone: "+49 151 23456789",
};

test("books an appointment from date selection to confirmation", async ({
  page,
}) => {
  const date = bookableDate(7);

  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "Book your appointment" }),
  ).toBeVisible();

  await chooseDate(page, date);

  const slot = await firstAvailableSlot(page);

  await slotButton(page, slot).click();

  await expect(slotButton(page, slot)).toHaveAttribute("aria-pressed", "true");

  await fillCustomer(page, customer);

  const createResponse = page.waitForResponse(
    (response) =>
      response.url().endsWith("/api/appointments") &&
      response.request().method() === "POST",
  );

  await confirmButton(page).click();

  expect((await createResponse).status()).toBe(201);

  await expect(
    page.getByRole("heading", { name: "Appointment confirmed!" }),
  ).toBeVisible();

  await expect(page.getByText(displayDate(date))).toBeVisible();
  await expect(page.getByText(`${slot.start} – ${slot.end}`)).toBeVisible();
  await expect(page.getByText(customer.email)).toBeVisible();

  // The booking was stored: choosing the same date again no longer offers
  // the booked time.
  await page.getByRole("button", { name: "Book another appointment" }).click();

  await chooseDate(page, date);

  await expect(slotButton(page, slot)).toHaveCount(0);
});
