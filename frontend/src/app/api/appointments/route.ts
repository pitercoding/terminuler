import { NextRequest, NextResponse } from "next/server";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

/**
 * Returns the IP of the client that called this proxy, or null when it is
 * unknown. The last X-Forwarded-For entry is the one added by the closest
 * hop (the hosting platform, a reverse proxy or Next.js itself), while
 * earlier entries can be sent by the client and are not trusted. The Go API
 * only accepts the result when the connection comes from TRUSTED_PROXIES.
 */
function clientIP(request: NextRequest): string | null {
    const forwardedFor = request.headers.get("x-forwarded-for");

    const ip = forwardedFor?.split(",").at(-1)?.trim();

    return ip || null;
}

export async function POST(request: NextRequest) {
    try {
        const body = await request.json();

        const headers = new Headers({ "Content-Type": "application/json" });

        const ip = clientIP(request);

        if (ip) {
            headers.set("X-Real-IP", ip);
        }

        const response = await fetch(`${API_URL}/appointments`, {
            method: "POST",
            headers,
            body: JSON.stringify(body),
        });

        const data = await response.json();

        const retryAfter = response.headers.get("Retry-After");

        return NextResponse.json(data, {
            status: response.status,
            headers: retryAfter ? { "Retry-After": retryAfter } : undefined,
        });
    } catch {
        return NextResponse.json(
            { error: "failed to connect to appointment API" },
            { status: 502 },
        );
    }
}
