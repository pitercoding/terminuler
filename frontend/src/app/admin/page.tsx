"use client";

import { SignOutButton } from "@clerk/nextjs";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { ErrorMessage } from "@/components/ErrorMessage";
import { formatDisplayDate } from "@/lib/date";
import {
    AppointmentApiError,
    cancelAdminAppointment,
    getAdminAppointments,
    type Appointment,
    type AppointmentStatus,
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

    // The appointment awaiting confirmation in the cancel dialog, if any.
    const [pendingCancel, setPendingCancel] = useState<Appointment | null>(
        null,
    );
    const [cancelling, setCancelling] = useState(false);
    const [cancelError, setCancelError] = useState<string | null>(null);

    function handleRetry() {
        setState({ status: "loading" });
        loadAppointments().then(setState);
    }

    /**
     * Marks the appointment as cancelled in place, so it stays in the
     * history. The API answers 204 without a body, so cancelled_at is the
     * browser's time until the next load brings the stored one.
     */
    function markCancelled(id: number) {
        const cancelledAt = new Date().toISOString();

        setState((current) =>
            current.status === "loaded"
                ? {
                    ...current,
                    appointments: current.appointments.map((appointment) =>
                        appointment.id === id
                            ? {
                                ...appointment,
                                status: "cancelled",
                                cancelled_at: cancelledAt,
                            }
                            : appointment,
                    ),
                }
                : current,
        );
    }

    function openCancelDialog(appointment: Appointment) {
        setCancelError(null);
        setPendingCancel(appointment);
    }

    function closeCancelDialog() {
        setPendingCancel(null);
        setCancelError(null);
    }

    async function handleConfirmCancel() {
        if (!pendingCancel || cancelling) {
            return;
        }

        setCancelling(true);
        setCancelError(null);

        try {
            await cancelAdminAppointment(pendingCancel.id);

            markCancelled(pendingCancel.id);
            closeCancelDialog();
        } catch (err) {
            // Already cancelled elsewhere: the outcome the admin wanted.
            if (err instanceof AppointmentApiError && err.status === 404) {
                markCancelled(pendingCancel.id);
                closeCancelDialog();
            } else {
                setCancelError(
                    "Unable to cancel the appointment. Please try again.",
                );
            }
        } finally {
            setCancelling(false);
        }
    }

    return (
        <main className="mx-auto w-full max-w-5xl px-4 py-10 sm:px-6 sm:py-14">
            <nav className="mb-8 flex items-center justify-between gap-4">
                <Link
                    href="/"
                    className="rounded-lg py-1.5 text-sm font-medium text-slate-600 transition-colors hover:text-teal-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-600"
                >
                    <span aria-hidden="true">←</span> Back to booking
                </Link>

                {/* Clerk ends the session; /admin then shows the sign-in again. */}
                <SignOutButton redirectUrl="/admin">
                    <button
                        type="button"
                        className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-800 shadow-sm transition-colors hover:bg-slate-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-600"
                    >
                        Logout
                    </button>
                </SignOutButton>
            </nav>

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
                            <AppointmentsTable
                                appointments={state.appointments}
                                onCancel={openCancelDialog}
                            />
                        ))}
                </div>
            </section>

            {pendingCancel && (
                <CancelAppointmentDialog
                    appointment={pendingCancel}
                    cancelling={cancelling}
                    error={cancelError}
                    onConfirm={handleConfirmCancel}
                    onClose={closeCancelDialog}
                />
            )}
        </main>
    );
}

interface AppointmentsTableProps {
    appointments: Appointment[];
    onCancel: (appointment: Appointment) => void;
}

function AppointmentsTable({ appointments, onCancel }: AppointmentsTableProps) {
    return (
        <div className="-mx-6 overflow-x-auto sm:-mx-8">
            <table className="w-full min-w-208 text-left text-sm">
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
                        <th scope="col" className="px-3 py-3 font-medium">
                            Email
                        </th>
                        <th scope="col" className="px-3 py-3 font-medium">
                            Status
                        </th>
                        <th
                            scope="col"
                            className="px-6 py-3 text-right font-medium sm:px-8"
                        >
                            Actions
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
                            <td className="px-3 py-3">
                                <a
                                    href={`mailto:${appointment.customer_email}`}
                                    className="break-all hover:text-teal-700 hover:underline"
                                >
                                    {appointment.customer_email}
                                </a>
                            </td>
                            <td className="whitespace-nowrap px-3 py-3">
                                <StatusBadge status={appointment.status} />
                            </td>
                            <td className="px-6 py-3 text-right sm:px-8">
                                {/* A cancelled appointment cannot be cancelled again. */}
                                {appointment.status === "confirmed" ? (
                                    <button
                                        type="button"
                                        onClick={() => onCancel(appointment)}
                                        aria-label={`Cancel appointment of ${appointment.customer_name} on ${formatDisplayDate(appointment.appointment_date)} at ${appointment.start_time}`}
                                        className="rounded-lg px-3 py-1.5 text-sm font-semibold text-red-700 hover:bg-red-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-red-600"
                                    >
                                        Cancel
                                    </button>
                                ) : (
                                    <span className="px-3 text-slate-400">
                                        <span aria-hidden="true">—</span>
                                        <span className="sr-only">No actions</span>
                                    </span>
                                )}
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

const statusBadges: Record<
    AppointmentStatus,
    { label: string; className: string }
> = {
    confirmed: {
        label: "Confirmed",
        className: "bg-teal-50 text-teal-700 ring-teal-600/20",
    },
    cancelled: {
        label: "Cancelled",
        className: "bg-slate-100 text-slate-600 ring-slate-500/20",
    },
};

// Shown for a status this page does not know, e.g. from an outdated API,
// instead of breaking the whole table.
const unknownStatusBadge = {
    label: "Unknown",
    className: "bg-amber-50 text-amber-800 ring-amber-600/20",
};

function StatusBadge({ status }: { status: AppointmentStatus }) {
    const badge = statusBadges[status] ?? unknownStatusBadge;

    return (
        <span
            className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset ${badge.className}`}
        >
            {badge.label}
        </span>
    );
}

interface CancelAppointmentDialogProps {
    appointment: Appointment;
    cancelling: boolean;
    error: string | null;
    onConfirm: () => void;
    onClose: () => void;
}

/**
 * Asks the admin to confirm a cancellation. A modal <dialog> traps focus,
 * makes the rest of the page inert and closes on Escape, except while the
 * request is in flight.
 */
function CancelAppointmentDialog({
    appointment,
    cancelling,
    error,
    onConfirm,
    onClose,
}: CancelAppointmentDialogProps) {
    const dialogRef = useRef<HTMLDialogElement>(null);

    useEffect(() => {
        dialogRef.current?.showModal();
    }, []);

    return (
        <dialog
            ref={dialogRef}
            aria-labelledby="cancel-dialog-title"
            aria-describedby="cancel-dialog-description"
            onCancel={(event) => {
                if (cancelling) {
                    event.preventDefault();
                }
            }}
            onClose={onClose}
            className="m-auto w-[calc(100%-2rem)] max-w-md rounded-2xl bg-white p-6 shadow-xl backdrop:bg-slate-900/40 sm:p-8"
        >
            <h2
                id="cancel-dialog-title"
                className="text-lg font-semibold text-slate-900"
            >
                Cancel appointment?
            </h2>

            <div className="mt-4 rounded-xl bg-slate-50 px-4 py-3 text-sm">
                <p className="font-medium text-slate-900">
                    {appointment.customer_name}
                </p>
                <p className="mt-0.5 tabular-nums text-slate-600">
                    {formatDisplayDate(appointment.appointment_date)} ·{" "}
                    {appointment.start_time}–{appointment.end_time}
                </p>
            </div>

            <p
                id="cancel-dialog-description"
                className="mt-4 text-sm text-slate-600"
            >
                The time slot becomes available for booking again. This action
                cannot be undone.
            </p>

            {error && (
                <div className="mt-4">
                    <ErrorMessage message={error} />
                </div>
            )}

            <div className="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
                <button
                    type="button"
                    autoFocus
                    disabled={cancelling}
                    onClick={() => dialogRef.current?.close()}
                    className="rounded-lg border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm transition-colors hover:bg-slate-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-600 disabled:cursor-not-allowed disabled:opacity-60"
                >
                    Keep appointment
                </button>
                <button
                    type="button"
                    disabled={cancelling}
                    onClick={onConfirm}
                    className="rounded-lg bg-red-700 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-red-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-red-600 disabled:cursor-not-allowed disabled:opacity-60"
                >
                    {cancelling ? "Cancelling…" : "Cancel appointment"}
                </button>
            </div>
        </dialog>
    );
}
