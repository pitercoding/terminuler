import { useEffect, useRef } from "react";
import { formatDisplayDate } from "@/lib/date";
import type { Appointment } from "@/services/appointmentService";

interface AppointmentSuccessProps {
  appointment: Appointment;
  onBookAnother: () => void;
}

export function AppointmentSuccess({
  appointment,
  onBookAnother,
}: AppointmentSuccessProps) {
  const headingRef = useRef<HTMLHeadingElement>(null);

  // The form this screen replaces had focus, so focus moves to the heading
  // for screen readers to announce the confirmation.
  useEffect(() => {
    headingRef.current?.focus();
  }, []);

  return (
    <section className="rounded-2xl border border-slate-200 bg-white px-6 py-10 text-center shadow-sm sm:px-10">
      <div
        aria-hidden="true"
        className="mx-auto flex size-14 items-center justify-center rounded-full bg-teal-50"
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2.5}
          strokeLinecap="round"
          strokeLinejoin="round"
          className="size-7 text-teal-700"
        >
          <path d="m5 12.5 4.5 4.5L19 7.5" />
        </svg>
      </div>

      <h1
        ref={headingRef}
        tabIndex={-1}
        className="mt-5 text-2xl font-semibold tracking-tight text-slate-900 focus:outline-none"
      >
        Appointment confirmed!
      </h1>

      <p className="mt-2 text-slate-600">
        Your appointment has been successfully booked.
      </p>

      <div className="mt-8 rounded-xl bg-slate-50 px-4 py-5">
        <p className="font-semibold text-slate-900">
          {formatDisplayDate(appointment.appointment_date)}
        </p>
        <p className="mt-1 tabular-nums text-slate-700">
          {appointment.start_time} – {appointment.end_time}
        </p>
      </div>

      <p className="mt-8 text-sm text-slate-600">
        A confirmation email will be sent to
      </p>
      <p className="mt-1 break-all font-medium text-slate-900">
        {appointment.customer_email}
      </p>

      <button
        type="button"
        onClick={onBookAnother}
        className="mt-10 w-full rounded-lg border border-slate-300 bg-white px-4 py-3 font-semibold text-slate-800 shadow-sm transition-colors hover:bg-slate-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-600 sm:w-auto sm:px-6"
      >
        Book another appointment
      </button>
    </section>
  );
}
