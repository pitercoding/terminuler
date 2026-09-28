"use client";

import { useState } from "react";
import {
  AppointmentForm,
  type CustomerDetails,
} from "@/components/AppointmentForm";
import { AppointmentSuccess } from "@/components/AppointmentSuccess";
import { DateSelector } from "@/components/DateSelector";
import { ErrorMessage } from "@/components/ErrorMessage";
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

  // Kept here rather than in the form so the details survive when the form is hidden, for example after a 409 conflict clears the selected slot.
  const [customer, setCustomer] = useState<CustomerDetails>(emptyCustomer);
  const [createdAppointment, setCreatedAppointment] =
    useState<Appointment | null>(null);

  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function handleDateChange(newDate: string) {
    setDate(newDate);
    setSelectedSlot(null);
    setAvailability(null);
    setError(null);
  }

  async function handleSearch() {
    if (!date) {
      setError("Please select a date.");
      setAvailability(null);
      setSelectedSlot(null);
      return;
    }

    setLoading(true);
    setError(null);
    setAvailability(null);
    setSelectedSlot(null);

    try {
      const data = await getAvailableSlots(date);
      setAvailability(data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch availability.",
      );
    } finally {
      setLoading(false);
    }
  }

  function handleSlotSelect(slot: AvailableSlot) {
    setSelectedSlot(slot);
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
    <main>
      <h1>Book an appointment</h1>

      <DateSelector
        date={date}
        isSearching={loading}
        onDateChange={handleDateChange}
        onSearch={handleSearch}
      />

      {error && <ErrorMessage message={error} />}

      {availability && (
        <TimeSlotGrid
          slots={availability.available_slots}
          selectedSlot={selectedSlot}
          onSelect={handleSlotSelect}
        />
      )}

      {selectedSlot && (
        <AppointmentForm
          slot={selectedSlot}
          customer={customer}
          isSubmitting={loading}
          onCustomerChange={setCustomer}
          onSubmit={handleSubmit}
        />
      )}
    </main>
  );
}
