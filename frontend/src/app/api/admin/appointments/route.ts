import { auth } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

/**
 * Forwards the request to the admin API with the Clerk session token of the
 * signed-in user. The Go API verifies the token and that the user is the
 * admin, so this route only has to reject requests without a session.
 */
export async function GET() {
    const { getToken } = await auth();

    const token = await getToken();

    if (!token) {
        return NextResponse.json(
            { error: "unauthorized" },
            { status: 401 },
        );
    }

    try {
        const response = await fetch(`${API_URL}/admin/appointments`, {
            headers: { Authorization: `Bearer ${token}` },
            cache: "no-store",
        });

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
