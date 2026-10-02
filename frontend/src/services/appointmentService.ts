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

/**
 * A confirmed appointment holds its slot; a cancelled one stays stored for
 * the history but frees its slot.
 */
export type AppointmentStatus = "confirmed" | "cancelled";

export interface Appointment {
    id: number;
    appointment_date: string;
    start_time: string;
    end_time: string;
    customer_name: string;
    customer_phone: string;
    customer_email: string;
    status: AppointmentStatus;
    created_at: string;
    // Null unless status is "cancelled".
    cancelled_at: string | null;
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
/**
 * Returns the upcoming appointments, from today onwards, both confirmed and
 * cancelled. Only the admin can list them: other users get a 401 or 403
 * AppointmentApiError.
 */
export async function getAdminAppointments(): Promise<Appointment[]> {
    const response = await fetch("/api/admin/appointments", {
        cache: "no-store",
    });

    const data = await readJSON(response);

    if (!response.ok || !Array.isArray(data)) {
        throw new AppointmentApiError(
            errorMessageFrom(data, "Failed to fetch appointments"),
            response.status,
        );
    }

    return data as Appointment[];
}

/**
 * Cancels the appointment with the given ID, freeing its slot while keeping
 * it in the history. Only the admin can cancel: other users get a 401 or 403
 * AppointmentApiError, and an appointment that does not exist or is already
 * cancelled gets a 404.
 */
export async function cancelAdminAppointment(id: number): Promise<void> {
    const response = await fetch(`/api/admin/appointments/${id}`, {
        method: "DELETE",
    });

    // 204 No Content: there is no body to read.
    if (response.ok) {
        return;
    }

    const data = await readJSON(response);

    throw new AppointmentApiError(
        errorMessageFrom(data, "Failed to cancel appointment"),
        response.status,
    );
}
