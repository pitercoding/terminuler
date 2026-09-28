import { useEffect, useRef } from "react";
import { ErrorMessage } from "@/components/ErrorMessage";
import { inputClassName, labelClassName } from "@/components/styles";
import { formatDisplayDate } from "@/lib/date";
import { scrollIntoView } from "@/lib/scroll";
import type { AvailableSlot } from "@/services/appointmentService";

export interface CustomerDetails {
  name: string;
  phone: string;
  email: string;
}

interface AppointmentFormProps {
  date: string;
  slot: AvailableSlot;
  customer: CustomerDetails;
  error: string | null;
  isSubmitting: boolean;
  onCustomerChange: (customer: CustomerDetails) => void;
  onChangeTime: () => void;
  onSubmit: () => void;
}

export function AppointmentForm({
  date,
  slot,
  customer,
  error,
  isSubmitting,
  onCustomerChange,
  onChangeTime,
  onSubmit,
}: AppointmentFormProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  // The form appears below the time slots, which on small screens is out of
  // view, so it is brought into view when it first shows up.
  useEffect(() => {
    scrollIntoView(containerRef.current, "start");
  }, []);

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSubmit();
  }

  return (
    // scroll-mt leaves room for the card title above the form.
    <div ref={containerRef} className="scroll-mt-28">
      <div className="flex flex-col gap-3 rounded-xl bg-teal-50 px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-teal-700">
            Selected appointment
          </p>
          <p className="mt-1 font-semibold text-slate-900">
            {formatDisplayDate(date)}
          </p>
          <p className="text-sm tabular-nums text-slate-700">
            {slot.start_time} – {slot.end_time}
          </p>
        </div>

        <button
          type="button"
          onClick={onChangeTime}
          disabled={isSubmitting}
          className="self-start rounded-lg px-3 py-2 text-sm font-semibold text-teal-700 hover:bg-teal-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-600 disabled:cursor-not-allowed disabled:opacity-60 sm:self-center"
        >
          Change time
        </button>
      </div>

      {/* Lengths mirror the limits enforced by the API. */}
      <form onSubmit={handleSubmit} className="mt-6">
        <fieldset disabled={isSubmitting} className="space-y-5">
          <legend className="sr-only">Your contact details</legend>

          <div>
            <label htmlFor="customer-name" className={labelClassName}>
              Full name
            </label>
            <input
              id="customer-name"
              name="name"
              type="text"
              autoComplete="name"
              required
              maxLength={255}
              value={customer.name}
              onChange={(event) =>
                onCustomerChange({ ...customer, name: event.target.value })
              }
              placeholder="Jane Doe"
              className={inputClassName}
            />
          </div>

          <div>
            <label htmlFor="customer-email" className={labelClassName}>
              Email
            </label>
            <input
              id="customer-email"
              name="email"
              type="email"
              autoComplete="email"
              required
              maxLength={255}
              value={customer.email}
              onChange={(event) =>
                onCustomerChange({ ...customer, email: event.target.value })
              }
              placeholder="jane@example.com"
              className={inputClassName}
            />
          </div>

          <div>
            <label htmlFor="customer-phone" className={labelClassName}>
              Phone
            </label>
            <input
              id="customer-phone"
              name="phone"
              type="tel"
              autoComplete="tel"
              required
              maxLength={50}
              value={customer.phone}
              onChange={(event) =>
                onCustomerChange({ ...customer, phone: event.target.value })
              }
              placeholder="+49 151 23456789"
              className={inputClassName}
            />
          </div>

          {error && <ErrorMessage message={error} />}

          <button
            type="submit"
            className="w-full rounded-lg bg-teal-700 px-4 py-3 font-semibold text-white shadow-sm transition-colors hover:bg-teal-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-600 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {isSubmitting ? "Confirming appointment..." : "Confirm appointment"}
          </button>
        </fieldset>
      </form>
    </div>
  );
}
