import { NextResponse } from "next/server";

/**
 * The answer of the route handlers when the Go API cannot be reached at all:
 * fetch threw before any response arrived.
 */
export function apiUnavailable(): NextResponse {
    return NextResponse.json(
        { error: "failed to connect to appointment API" },
        { status: 502 },
    );
}

/**
 * Forwards a response of the Go API to the browser with the same status.
 * Bodies that are not JSON, such as the empty 401 of the Clerk middleware,
 * the plain-text 404 of an unknown route or an error page of the hosting
 * platform, are replaced by {"error": ...}, so the status the API answered
 * is never masked as a 502. Only a success without a JSON body, which is not
 * an answer of the Go API (API_URL pointing elsewhere), becomes a 502.
 */
export async function forwardJSON(response: Response): Promise<NextResponse> {
    const text = await response.text();

    try {
        return NextResponse.json(JSON.parse(text), {
            status: response.status,
        });
    } catch {
        return NextResponse.json(
            { error: `appointment API responded with status ${response.status}` },
            { status: response.ok ? 502 : response.status },
        );
    }
}
