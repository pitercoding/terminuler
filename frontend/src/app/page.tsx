"use client";

import { useState } from "react";
import {
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
        <p>
          Selected time: {selectedSlot.start_time} -{" "}
          {selectedSlot.end_time}
        </p>
      )}
    </main>
  );
}