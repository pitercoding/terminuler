export interface AvailableSlot {
    start_time: string;
    end_time: string;
}

export interface AvailabilityResponse {
    date: string;
    available_slots: AvailableSlot[];
}

interface ErrorResponse {
    error: string;
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

        throw new Error(errorMessage);
    }

    return data as AvailabilityResponse;
}