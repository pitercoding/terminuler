import { NextRequest, NextResponse } from "next/server";

import { proxyToAPI } from "@/lib/apiProxy";

export async function GET(request: NextRequest) {
    const date = request.nextUrl.searchParams.get("date");

    if (!date) {
        return NextResponse.json(
            { error: "date query parameter is required" },
            { status: 400 },
        );
    }

    return proxyToAPI(
        `/appointments/availability?date=${encodeURIComponent(date)}`,
    );
}
