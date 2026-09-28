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

/**
 * Reads a JSON response body. Errors that do not come from the API, such as
 * an HTML error page, have no JSON body and yield null instead of throwing.
 */
async function readJSON(response: Response): Promise<unknown> {
    try {
        return await response.json();
    } catch {
        return null;
    }
}

function errorMessageFrom(data: unknown, fallback: string): string {
    if (
        typeof data === "object" &&
        data !== null &&
        "error" in data &&
        typeof (data as ErrorResponse).error === "string"
    ) {
        return (data as ErrorResponse).error;
    }

    return fallback;
}

export async function getAvailableSlots(
    date: string,
): Promise<AvailabilityResponse> {
    const response = await fetch(
        `/api/appointments/availability?date=${encodeURIComponent(date)}`,
    );

    const data = await readJSON(response);

    if (!response.ok || data === null) {
        throw new AppointmentApiError(
            errorMessageFrom(data, "Failed to fetch availability"),
            response.status,
        );
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

    const data = await readJSON(response);

    if (!response.ok || data === null) {
        throw new AppointmentApiError(
            errorMessageFrom(data, "Failed to create appointment"),
            response.status,
        );
    }

    return data as Appointment;
}