export interface AvailableSlot {
    start_time: string;
    end_time: string;
}

export interface AvailabilityResponse {
    date: string;
    available_slots: AvailableSlot[];
}

export interface CreateAppointmentRequest {
    appointment_date: string;
    start_time: string;
    end_time: string;
    customer_name: string;
    customer_phone: string;
    customer_email: string;
}

export interface Appointment {
    id: number;
    appointment_date: string;
    start_time: string;
    end_time: string;
    customer_name: string;
    customer_phone: string;
    customer_email: string;
    created_at: string;
}

interface ErrorResponse {
    error: string;
}

export class AppointmentApiError extends Error {
    status: number;

    constructor(message: string, status: number) {
        super(message);
        this.name = "AppointmentApiError";
        this.status = status;
    }
}

export async function getAvailableSlots(
    date: string,
): Promise<AvailabilityResponse> {
    const response = await fetch(
        `/api/appointments/availability?date=${encodeURIComponent(date)}`,
    );

    const data: AvailabilityResponse | ErrorResponse = await response.json();

    if (!response.ok) {
        const errorMessage =
            "error" in data ? data.error : "Failed to fetch availability";

        throw new AppointmentApiError(errorMessage, response.status);
    }

    return data as AvailabilityResponse;
}

export async function createAppointment(
    appointment: CreateAppointmentRequest,
): Promise<Appointment> {
    const response = await fetch("/api/appointments", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(appointment),
    });

    const data: Appointment | ErrorResponse = await response.json();

    if (!response.ok) {
        const errorMessage =
            "error" in data ? data.error : "Failed to create appointment";

        throw new AppointmentApiError(errorMessage, response.status);
    }

    return data as Appointment;
}