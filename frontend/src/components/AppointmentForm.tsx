import type { AvailableSlot } from "@/services/appointmentService";

export interface CustomerDetails {
  name: string;
  phone: string;
  email: string;
}

interface AppointmentFormProps {
  slot: AvailableSlot;
  customer: CustomerDetails;
  isSubmitting: boolean;
  onCustomerChange: (customer: CustomerDetails) => void;
  onSubmit: () => void;
}

export function AppointmentForm({
  slot,
  customer,
  isSubmitting,
  onCustomerChange,
  onSubmit,
}: AppointmentFormProps) {
  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSubmit();
  }

  return (
    <section>
      <h2>Appointment details</h2>

      <p>
        Selected time: {slot.start_time} - {slot.end_time}
      </p>

      <form onSubmit={handleSubmit}>
        <div>
          <label htmlFor="customer-name">Full name</label>
          <input
            id="customer-name"
            type="text"
            value={customer.name}
            onChange={(event) =>
              onCustomerChange({ ...customer, name: event.target.value })
            }
            placeholder="Your full name"
          />
        </div>

        <div>
          <label htmlFor="customer-phone">Phone</label>
          <input
            id="customer-phone"
            type="tel"
            value={customer.phone}
            onChange={(event) =>
              onCustomerChange({ ...customer, phone: event.target.value })
            }
            placeholder="Your phone number"
          />
        </div>

        <div>
          <label htmlFor="customer-email">Email</label>
          <input
            id="customer-email"
            type="email"
            value={customer.email}
            onChange={(event) =>
              onCustomerChange({ ...customer, email: event.target.value })
            }
            placeholder="you@example.com"
          />
        </div>

        <button type="submit" disabled={isSubmitting}>
          {isSubmitting ? "Confirming..." : "Confirm appointment"}
        </button>
      </form>
    </section>
  );
}
