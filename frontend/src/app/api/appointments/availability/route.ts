import { NextRequest, NextResponse } from "next/server";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

export async function GET(request: NextRequest) {
    const date = request.nextUrl.searchParams.get("date");

    if (!date) {
        return NextResponse.json(
            { error: "date query parameter is required" },
            { status: 400 },
        );
    }

    try {
        const response = await fetch(
            `${API_URL}/appointments/availability?date=${encodeURIComponent(date)}`,
        );

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