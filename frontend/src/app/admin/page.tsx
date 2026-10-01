"use client";

import { useEffect, useState } from "react";
import { ErrorMessage } from "@/components/ErrorMessage";
import { formatDisplayDate } from "@/lib/date";
import {
    AppointmentApiError,
    getAdminAppointments,
    type Appointment,
} from "@/services/appointmentService";

type AppointmentsState =
    | { status: "loading" }
    | { status: "error"; message: string; canRetry: boolean }
    | { status: "loaded"; appointments: Appointment[] };

/**
 * Returns the message shown when the appointments cannot be loaded. A 403
 * means the user is signed in but is not the admin, so retrying is useless.
 */
function loadFailure(err: unknown): { message: string; canRetry: boolean } {
    if (err instanceof AppointmentApiError && err.status === 403) {
        return {
            message: "Your account does not have access to the admin dashboard.",
            canRetry: false,
        };
    }

    return {
        message: "Unable to load appointments. Please try again.",
        canRetry: true,
    };
}

async function loadAppointments(): Promise<AppointmentsState> {
    try {
        const appointments = await getAdminAppointments();

        return { status: "loaded", appointments };
    } catch (err) {
        return { status: "error", ...loadFailure(err) };
    }
}

export default function AdminPage() {
    const [state, setState] = useState<AppointmentsState>({
        status: "loading",
    });

    useEffect(() => {
        let ignore = false;

        loadAppointments().then((result) => {
            if (!ignore) {
                setState(result);
            }
        });

        return () => {
            ignore = true;
        };
    }, []);

    function handleRetry() {
        setState({ status: "loading" });
        loadAppointments().then(setState);
    }

    return (
        <main className="mx-auto w-full max-w-5xl px-4 py-10 sm:px-6 sm:py-14">
            <header>
                <h1 className="text-2xl font-semibold tracking-tight text-slate-900 sm:text-3xl">
                    Admin Dashboard
                </h1>
                <p className="mt-2 text-slate-600">
                    Upcoming appointments, from today onwards.
                </p>
            </header>

            <section
                aria-labelledby="appointments-heading"
                aria-busy={state.status === "loading"}
                className="mt-8 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm sm:p-8"
            >
                <h2
                    id="appointments-heading"
                    className="text-lg font-semibold text-slate-900"
                >
                    Appointments
                    {state.status === "loaded" && (
                        <span className="ml-2 text-sm font-normal text-slate-500">
                            ({state.appointments.length})
                        </span>
                    )}
                </h2>

                <div className="mt-6">
                    {state.status === "loading" && (
                        <p className="text-sm text-slate-500">Loading appointments…</p>
                    )}

                    {state.status === "error" && (
                        <ErrorMessage
                            message={state.message}
                            onRetry={state.canRetry ? handleRetry : undefined}
                        />
                    )}

                    {state.status === "loaded" &&
                        (state.appointments.length === 0 ? (
                            <p className="text-sm text-slate-500">
                                No upcoming appointments.
                            </p>
                        ) : (
                            <AppointmentsTable appointments={state.appointments} />
                        ))}
                </div>
            </section>
        </main>
    );
}

function AppointmentsTable({ appointments }: { appointments: Appointment[] }) {
    return (
        <div className="-mx-6 overflow-x-auto sm:-mx-8">
            <table className="w-full min-w-[40rem] text-left text-sm">
                <thead className="border-b border-slate-200 text-slate-500">
                    <tr>
                        <th scope="col" className="px-6 py-3 font-medium sm:px-8">
                            Date
                        </th>
                        <th scope="col" className="px-3 py-3 font-medium">
                            Time
                        </th>
                        <th scope="col" className="px-3 py-3 font-medium">
                            Name
                        </th>
                        <th scope="col" className="px-3 py-3 font-medium">
                            Phone
                        </th>
                        <th scope="col" className="px-6 py-3 font-medium sm:px-8">
                            Email
                        </th>
                    </tr>
                </thead>

                <tbody className="divide-y divide-slate-100 text-slate-700">
                    {appointments.map((appointment) => (
                        <tr key={appointment.id}>
                            <td className="whitespace-nowrap px-6 py-3 font-medium text-slate-900 sm:px-8">
                                {formatDisplayDate(appointment.appointment_date)}
                            </td>
                            <td className="whitespace-nowrap px-3 py-3 tabular-nums">
                                {appointment.start_time}–{appointment.end_time}
                            </td>
                            <td className="px-3 py-3">{appointment.customer_name}</td>
                            <td className="whitespace-nowrap px-3 py-3">
                                <a
                                    href={`tel:${appointment.customer_phone}`}
                                    className="hover:text-teal-700 hover:underline"
                                >
                                    {appointment.customer_phone}
                                </a>
                            </td>
                            <td className="px-6 py-3 sm:px-8">
                                <a
                                    href={`mailto:${appointment.customer_email}`}
                                    className="break-all hover:text-teal-700 hover:underline"
                                >
                                    {appointment.customer_email}
                                </a>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
