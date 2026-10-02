"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import {
  AppointmentForm,
  type CustomerDetails,
} from "@/components/AppointmentForm";
import { AppointmentSuccess } from "@/components/AppointmentSuccess";
import { DateSelector } from "@/components/DateSelector";
import { ErrorMessage } from "@/components/ErrorMessage";
import { StepCard } from "@/components/StepCard";
import {
  TimeSlotGrid,
  TimeSlotGridSkeleton,
} from "@/components/TimeSlotGrid";
import { scrollIntoView } from "@/lib/scroll";
import {
  AppointmentApiError,
  createAppointment,
  getAvailableSlots,
  type Appointment,
  type AvailabilityResponse,
  type AvailableSlot,
} from "@/services/appointmentService";

const emptyCustomer: CustomerDetails = {
  name: "",
  phone: "",
  email: "",
};

const availabilityErrorMessage =
  "Unable to load available times. Please try again.";

const conflictMessage =
  "This time slot is no longer available. Please choose another time.";

const bookingErrorMessage =
  "We couldn't confirm your appointment. Please try again.";

/**
 * Returns the message shown when a booking fails. Validation errors (400)
 * explain what to fix, so the API message is shown; anything else gets a
 * generic message instead of technical details.
 */
function bookingFailureMessage(err: unknown): string {
  if (err instanceof AppointmentApiError && err.status === 400) {
    return `${err.message.charAt(0).toUpperCase()}${err.message.slice(1)}.`;
  }

  return bookingErrorMessage;
}

export default function Home() {
  const [date, setDate] = useState("");
  const [availability, setAvailability] =
    useState<AvailabilityResponse | null>(null);
  const [selectedSlot, setSelectedSlot] =
    useState<AvailableSlot | null>(null);

  // Kept here rather than in the form so the details survive when the form
  // is hidden, for example after a 409 conflict clears the selected slot.
  const [customer, setCustomer] = useState<CustomerDetails>(emptyCustomer);
  const [createdAppointment, setCreatedAppointment] =
    useState<Appointment | null>(null);

  const [isLoadingAvailability, setIsLoadingAvailability] = useState(false);
  const [isBooking, setIsBooking] = useState(false);

  // Each error is shown next to the step it belongs to.
  const [availabilityError, setAvailabilityError] = useState<string | null>(
    null,
  );
  const [conflictError, setConflictError] = useState<string | null>(null);
  const [bookingError, setBookingError] = useState<string | null>(null);

  // Availability is fetched on every date change, so responses can arrive
  // out of order. Only the response of the latest request is applied.
  const latestAvailabilityRequest = useRef(0);

  // After a conflict the form disappears, so the message above the time
  // slots is brought into view; on small screens it would be off-screen.
  const conflictRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (conflictError) {
      scrollIntoView(conflictRef.current, "center");
    }
  }, [conflictError]);

  async function loadAvailability(forDate: string) {
    const requestId = ++latestAvailabilityRequest.current;

    setAvailability(null);
    setAvailabilityError(null);
    setIsLoadingAvailability(true);

    try {
      const data = await getAvailableSlots(forDate);

      if (requestId === latestAvailabilityRequest.current) {
        setAvailability(data);
      }
    } catch {
      if (requestId === latestAvailabilityRequest.current) {
        setAvailabilityError(availabilityErrorMessage);
      }
    } finally {
      if (requestId === latestAvailabilityRequest.current) {
        setIsLoadingAvailability(false);
      }
    }
  }

  function handleDateChange(newDate: string) {
    setDate(newDate);
    setSelectedSlot(null);
    setConflictError(null);
    setBookingError(null);

    if (!newDate) {
      // Discard any request still in flight for the previous date.
      latestAvailabilityRequest.current++;
      setAvailability(null);
      setAvailabilityError(null);
      setIsLoadingAvailability(false);
      return;
    }

    void loadAvailability(newDate);
  }

  function handleSlotSelect(slot: AvailableSlot) {
    setSelectedSlot(slot);
    setConflictError(null);
    setBookingError(null);
  }

  function handleChangeTime() {
    setSelectedSlot(null);
    setBookingError(null);
  }

  async function handleSubmit() {
    if (!selectedSlot) {
      return;
    }

    // The inputs are required, but a value made only of spaces passes the
    // browser's validation.
    if (
      !customer.name.trim() ||
      !customer.phone.trim() ||
      !customer.email.trim()
    ) {
      setBookingError("Please fill in your name, email and phone.");
      return;
    }

    setBookingError(null);
    setIsBooking(true);

    try {
      const appointment = await createAppointment({
        appointment_date: date,
        start_time: selectedSlot.start_time,
        end_time: selectedSlot.end_time,
        customer_name: customer.name.trim(),
        customer_phone: customer.phone.trim(),
        customer_email: customer.email.trim(),
      });

      setCreatedAppointment(appointment);
    } catch (err) {
      if (err instanceof AppointmentApiError && err.status === 409) {
        // The slot was taken by someone else: go back to the time slots,
        // which are reloaded so the taken slot disappears.
        setSelectedSlot(null);
        setConflictError(conflictMessage);
        void loadAvailability(date);
        return;
      }

      setBookingError(bookingFailureMessage(err));
    } finally {
      setIsBooking(false);
    }
  }

  function handleBookAnother() {
    // Discard any availability request still in flight.
    latestAvailabilityRequest.current++;

    setDate("");
    setAvailability(null);
    setSelectedSlot(null);
    setCustomer(emptyCustomer);
    setCreatedAppointment(null);
    setIsLoadingAvailability(false);
    setAvailabilityError(null);
    setConflictError(null);
    setBookingError(null);

    window.scrollTo({ top: 0 });
  }

  return (
    <main className="mx-auto w-full max-w-2xl px-4 py-12 sm:py-16">
      <nav className="mb-10 flex items-center justify-between">
        <p className="text-sm font-semibold uppercase tracking-[0.2em] text-teal-700">
          Terminuler
        </p>

        {/* Only navigates: the proxy shows the Clerk sign-in when needed. */}
        <Link
          href="/admin"
          className="rounded-lg px-3 py-1.5 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-600"
        >
          Admin
        </Link>
      </nav>

      {!createdAppointment && (
        <header className="mb-10 text-center">
          <h1 className="text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
            Book your appointment
          </h1>
          <p className="mt-3 text-slate-600">
            Choose a date and a convenient time for your visit.
          </p>
        </header>
      )}

      {createdAppointment ? (
        <AppointmentSuccess
          appointment={createdAppointment}
          onBookAnother={handleBookAnother}
        />
      ) : (
        <div className="space-y-6">
          <StepCard step={1} title="Choose a date">
            <DateSelector
              date={date}
              disabled={isBooking}
              onDateChange={handleDateChange}
            />

            {conflictError && (
              <div ref={conflictRef} className="mt-6">
                <ErrorMessage message={conflictError} />
              </div>
            )}

            {availabilityError && (
              <div className="mt-6">
                <ErrorMessage
                  message={availabilityError}
                  onRetry={() => void loadAvailability(date)}
                />
              </div>
            )}

            {isLoadingAvailability && <TimeSlotGridSkeleton />}

            {availability && (
              <TimeSlotGrid
                slots={availability.available_slots}
                selectedSlot={selectedSlot}
                disabled={isBooking}
                onSelect={handleSlotSelect}
              />
            )}
          </StepCard>

          {selectedSlot && (
            <StepCard step={2} title="Your information">
              <AppointmentForm
                date={date}
                slot={selectedSlot}
                customer={customer}
                error={bookingError}
                isSubmitting={isBooking}
                onCustomerChange={setCustomer}
                onChangeTime={handleChangeTime}
                onSubmit={handleSubmit}
              />
            </StepCard>
          )}
        </div>
      )}
    </main>
  );
}
