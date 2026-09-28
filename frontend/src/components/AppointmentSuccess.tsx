import type { Appointment } from "@/services/appointmentService";

interface AppointmentSuccessProps {
  appointment: Appointment;
}

export function AppointmentSuccess({ appointment }: AppointmentSuccessProps) {
  return (
    <main>
      <h1>Appointment confirmed!</h1>

      <p>Your appointment has been successfully booked.</p>

      <section>
        <h2>Appointment details</h2>

        <p>
          <strong>Date:</strong> {appointment.appointment_date}
        </p>

        <p>
          <strong>Time:</strong> {appointment.start_time} -{" "}
          {appointment.end_time}
        </p>

        <p>
          <strong>Name:</strong> {appointment.customer_name}
        </p>

        <p>
          <strong>Email:</strong> {appointment.customer_email}
        </p>

        <p>A confirmation email will be sent to this address.</p>
      </section>
    </main>
  );
}
