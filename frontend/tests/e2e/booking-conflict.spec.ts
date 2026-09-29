import { expect, test } from "@playwright/test";
import {
  bookableDate,
  chooseDate,
  confirmButton,
  fillCustomer,
  firstAvailableSlot,
  slotButton,
  timeSlots,
} from "./helpers";

const conflictMessage =
  "This time slot is no longer available. Please choose another time.";

test("shows a conflict when the slot is taken by someone else", async ({
  browser,
}) => {
  const date = bookableDate(14);

  // Two separate browser contexts behave like two customers on different
  // devices: both see the same time as available before either books it.
  const aliceContext = await browser.newContext();
  const bobContext = await browser.newContext();

  const alice = await aliceContext.newPage();
  const bob = await bobContext.newPage();

  await alice.goto("/");
  await bob.goto("/");

  await chooseDate(alice, date);
  await chooseDate(bob, date);

  const slot = await firstAvailableSlot(alice);

  await slotButton(alice, slot).click();
  await slotButton(bob, slot).click();

  await fillCustomer(alice, {
    name: "Alice Example",
    email: "alice@example.com",
    phone: "+49 151 11111111",
  });

  await fillCustomer(bob, {
    name: "Bob Example",
    email: "bob@example.com",
    phone: "+49 151 22222222",
  });

  // Alice books first.
  await confirmButton(alice).click();

  await expect(
    alice.getByRole("heading", { name: "Appointment confirmed!" }),
  ).toBeVisible();

  // Bob books the same slot: PostgreSQL rejects the duplicate, the API
  // answers 409 and the Next.js proxy passes the status through.
  const conflictResponse = bob.waitForResponse(
    (response) =>
      response.url().endsWith("/api/appointments") &&
      response.request().method() === "POST",
  );

  await confirmButton(bob).click();

  expect((await conflictResponse).status()).toBe(409);

  // The form closes, the conflict is explained and the availability is
  // reloaded without the taken slot.
  await expect(bob.getByText(conflictMessage)).toBeVisible();
  await expect(bob.getByLabel("Full name")).toBeHidden();
  await expect(timeSlots(bob)).toBeVisible();
  await expect(slotButton(bob, slot)).toHaveCount(0);

  // Bob can recover by choosing another time; his details were kept.
  const otherSlot = await firstAvailableSlot(bob);

  await slotButton(bob, otherSlot).click();

  await expect(bob.getByLabel("Full name")).toHaveValue("Bob Example");

  await confirmButton(bob).click();

  await expect(
    bob.getByRole("heading", { name: "Appointment confirmed!" }),
  ).toBeVisible();
  await expect(
    bob.getByText(`${otherSlot.start} – ${otherSlot.end}`),
  ).toBeVisible();

  await aliceContext.close();
  await bobContext.close();
});
