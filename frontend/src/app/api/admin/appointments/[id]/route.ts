import { auth } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

/**
 * Cancels an appointment through the admin API with the Clerk session token
 * of the signed-in user. The Go API verifies the token and that the user is
 * the admin, so this route only has to reject requests without a session.
 */
export async function DELETE(
    _request: Request,
    { params }: { params: Promise<{ id: string }> },
) {
    const { getToken } = await auth();

    const token = await getToken();

    if (!token) {
        return NextResponse.json(
            { error: "unauthorized" },
            { status: 401 },
        );
    }

    const { id } = await params;

    try {
        const response = await fetch(
            `${API_URL}/admin/appointments/${encodeURIComponent(id)}`,
            {
                method: "DELETE",
                headers: { Authorization: `Bearer ${token}` },
                cache: "no-store",
            },
        );

        // A successful cancellation has no body to forward.
        if (response.status === 204) {
            return new NextResponse(null, { status: 204 });
        }

        const data = await response.json();

        return NextResponse.json(data, {
            status: response.status,
        });
    } catch {
        return NextResponse.json(
            { error: "failed to connect to appointment API" },
            { status: 502 },
        );
    }
}
