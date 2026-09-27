"use client";

import { useState } from "react";
import {
  AppointmentApiError,
  createAppointment,
  getAvailableSlots,
  type AvailabilityResponse,
  type AvailableSlot,
} from "@/services/appointmentService";

export default function Home() {
  const [date, setDate] = useState("");
  const [availability, setAvailability] =
    useState<AvailabilityResponse | null>(null);
  const [selectedSlot, setSelectedSlot] =
    useState<AvailableSlot | null>(null);

  const [customerName, setCustomerName] = useState("");
  const [customerPhone, setCustomerPhone] = useState("");
  const [customerEmail, setCustomerEmail] = useState("");

  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

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

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!selectedSlot) {
      setError("Please select an appointment time.");
      return;
    }

    if (!customerName.trim()) {
      setError("Please enter your full name.");
      return;
    }

    if (!customerPhone.trim()) {
      setError("Please enter your phone number.");
      return;
    }

    if (!customerEmail.trim()) {
      setError("Please enter your email address.");
      return;
    }

    setError(null);
    setLoading(true);

    try {
      await createAppointment({
        appointment_date: date,
        start_time: selectedSlot.start_time,
        end_time: selectedSlot.end_time,
        customer_name: customerName.trim(),
        customer_phone: customerPhone.trim(),
        customer_email: customerEmail.trim(),
      });

      console.log("Appointment created successfully");
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
        err instanceof Error
          ? err.message
          : "Failed to create appointment.",
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <main>
      <h1>Book an appointment</h1>

      <div>
        <label htmlFor="appointment-date">Select a date</label>

        <input
          id="appointment-date"
          type="date"
          value={date}
          min={new Date().toISOString().split("T")[0]}
          onChange={(event) => {
            setDate(event.target.value);
            setSelectedSlot(null);
            setAvailability(null);
            setError(null);
          }}
        />

        <button type="button" onClick={handleSearch} disabled={loading}>
          {loading ? "Searching..." : "Search availability"}
        </button>
      </div>

      {error && <p>{error}</p>}

      {availability && (
        <section>
          <h2>Available times</h2>

          {availability.available_slots.length === 0 ? (
            <p>No appointments available for this date.</p>
          ) : (
            <div>
              {availability.available_slots.map((slot) => {
                const isSelected =
                  selectedSlot?.start_time === slot.start_time &&
                  selectedSlot?.end_time === slot.end_time;

                return (
                  <button
                    key={slot.start_time}
                    type="button"
                    onClick={() => handleSlotSelect(slot)}
                    aria-pressed={isSelected}
                  >
                    {slot.start_time} - {slot.end_time}
                  </button>
                );
              })}
            </div>
          )}
        </section>
      )}

      {selectedSlot && (
        <section>
          <h2>Appointment details</h2>

          <p>
            Selected time: {selectedSlot.start_time} -{" "}
            {selectedSlot.end_time}
          </p>

          <form onSubmit={handleSubmit}>
            <div>
              <label htmlFor="customer-name">Full name</label>
              <input
                id="customer-name"
                type="text"
                value={customerName}
                onChange={(event) => setCustomerName(event.target.value)}
                placeholder="Your full name"
              />
            </div>

            <div>
              <label htmlFor="customer-phone">Phone</label>
              <input
                id="customer-phone"
                type="tel"
                value={customerPhone}
                onChange={(event) => setCustomerPhone(event.target.value)}
                placeholder="Your phone number"
              />
            </div>

            <div>
              <label htmlFor="customer-email">Email</label>
              <input
                id="customer-email"
                type="email"
                value={customerEmail}
                onChange={(event) => setCustomerEmail(event.target.value)}
                placeholder="you@example.com"
              />
            </div>

            <button type="submit" disabled={loading}>
              {loading ? "Confirming..." : "Confirm appointment"}
            </button>
          </form>
        </section>
      )}
    </main>
  );
}