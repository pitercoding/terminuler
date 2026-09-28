"use client";

import { useRef, useState } from "react";
import {
  AppointmentForm,
  type CustomerDetails,
} from "@/components/AppointmentForm";
import { AppointmentSuccess } from "@/components/AppointmentSuccess";
import { DateSelector } from "@/components/DateSelector";
import { ErrorMessage } from "@/components/ErrorMessage";
import { StepCard } from "@/components/StepCard";
import { TimeSlotGrid } from "@/components/TimeSlotGrid";
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

  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  // Availability is fetched on every date change, so responses can arrive
  // out of order. Only the response of the latest request is applied.
  const latestAvailabilityRequest = useRef(0);

  async function loadAvailability(forDate: string) {
    const requestId = ++latestAvailabilityRequest.current;

    setLoading(true);

    try {
      const data = await getAvailableSlots(forDate);

      if (requestId === latestAvailabilityRequest.current) {
        setAvailability(data);
      }
    } catch (err) {
      if (requestId === latestAvailabilityRequest.current) {
        setError(
          err instanceof Error ? err.message : "Failed to fetch availability.",
        );
      }
    } finally {
      if (requestId === latestAvailabilityRequest.current) {
        setLoading(false);
      }
    }
  }

  function handleDateChange(newDate: string) {
    setDate(newDate);
    setSelectedSlot(null);
    setAvailability(null);
    setError(null);

    if (!newDate) {
      // Discard any request still in flight for the previous date.
      latestAvailabilityRequest.current++;
      setLoading(false);
      return;
    }

    void loadAvailability(newDate);
  }

  function handleSlotSelect(slot: AvailableSlot) {
    setSelectedSlot(slot);
    setError(null);
  }

  function handleChangeTime() {
    setSelectedSlot(null);
    setError(null);
  }

  async function handleSubmit() {
    if (!selectedSlot) {
      setError("Please select an appointment time.");
      return;
    }

    if (!customer.name.trim()) {
      setError("Please enter your full name.");
      return;
    }

    if (!customer.phone.trim()) {
      setError("Please enter your phone number.");
      return;
    }

    if (!customer.email.trim()) {
      setError("Please enter your email address.");
      return;
    }

    setError(null);
    setLoading(true);

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
        setError(
          "This appointment slot is no longer available. Please select another time.",
        );

        try {
          const data = await getAvailableSlots(date);

          setAvailability(data);
          setSelectedSlot(null);
        } catch {
          setAvailability(null);
          setSelectedSlot(null);
        }

        return;
      }

      setError(
        err instanceof Error ? err.message : "Failed to create appointment.",
      );
    } finally {
      setLoading(false);
    }
  }

  if (createdAppointment) {
    return <AppointmentSuccess appointment={createdAppointment} />;
  }

  return (
    <main className="mx-auto w-full max-w-2xl px-4 py-12 sm:py-16">
      <header className="mb-10 text-center">
        <p className="text-sm font-semibold uppercase tracking-[0.2em] text-teal-700">
          Terminuler
        </p>
        <h1 className="mt-3 text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
          Book your appointment
        </h1>
        <p className="mt-3 text-slate-600">
          Choose a date and a convenient time for your visit.
        </p>
      </header>

      {error && <ErrorMessage message={error} />}

      <div className="space-y-6">
        <StepCard step={1} title="Choose a date">
          <DateSelector date={date} onDateChange={handleDateChange} />

          {loading && !availability && (
            <p className="mt-8 text-sm text-slate-500">
              Loading available times...
            </p>
          )}

          {availability && (
            <TimeSlotGrid
              slots={availability.available_slots}
              selectedSlot={selectedSlot}
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
              isSubmitting={loading}
              onCustomerChange={setCustomer}
              onChangeTime={handleChangeTime}
              onSubmit={handleSubmit}
            />
          </StepCard>
        )}
      </div>
    </main>
  );
}
