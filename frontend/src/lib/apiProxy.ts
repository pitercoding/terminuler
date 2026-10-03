import { NextResponse } from "next/server";

/**
 * How long the route handlers wait for the Go API. Long enough for a Render
 * free instance waking up from sleep, which can take close to a minute, but
 * bounded, so a hung API does not hold the request until the platform kills
 * the function.
 */
export const API_TIMEOUT_MS = 60_000;

const DEVELOPMENT_API_URL = "http://localhost:8080";

/**
 * Returns the base URL of the Go API (API_URL, server-only). Outside
 * production it defaults to the local API; in production a missing API_URL
 * returns null instead of silently pointing every request to localhost.
 */
export function apiURL(): string | null {
    if (process.env.API_URL) {
        return process.env.API_URL;
    }

    return process.env.NODE_ENV === "production" ? null : DEVELOPMENT_API_URL;
}

/**
 * The answer of the route handlers when the Go API cannot be reached at all:
 * fetch threw before any response arrived.
 */
export function apiUnavailable(): NextResponse {
    return NextResponse.json(
        { error: "failed to connect to appointment API" },
        { status: 502, headers: noStore() },
    );
}

/** The answer when the Go API accepted the request but did not respond in time. */
export function apiTimeout(): NextResponse {
    return NextResponse.json(
        { error: "appointment API did not respond in time" },
        { status: 504, headers: noStore() },
    );
}

/**
 * Forwards a response of the Go API to the browser with the same status.
 * Bodies that are not JSON, such as the empty 401 of the Clerk middleware,
 * the plain-text 404 of an unknown route or an error page of the hosting
 * platform, are replaced by {"error": ...}, so the status the API answered
 * is never masked as a 502 and no internal error page reaches the browser.
 * Only a success without a JSON body, which is not an answer of the Go API
 * (API_URL pointing elsewhere), becomes a 502. Retry-After is kept, so the
 * browser knows when a rate-limited booking may be retried.
 */
export async function forwardJSON(response: Response): Promise<NextResponse> {
    const headers = noStore();

    const retryAfter = response.headers.get("Retry-After");

    if (retryAfter) {
        headers["Retry-After"] = retryAfter;
    }

    const text = await response.text();

    try {
        return NextResponse.json(JSON.parse(text), {
            status: response.status,
            headers,
        });
    } catch {
        return NextResponse.json(
            { error: `appointment API responded with status ${response.status}` },
            { status: response.ok ? 502 : response.status, headers },
        );
    }
}

/**
 * Calls the Go API at path and turns the outcome into the response of the
 * route handler: the API's own answer (see forwardJSON), an empty 204, or a
 * 502, 503 or 504 when the API is unreachable, not configured or too slow.
 * Network and configuration details are logged, never sent to the browser.
 */
export async function proxyToAPI(
    path: string,
    init: RequestInit = {},
): Promise<NextResponse> {
    const baseURL = apiURL();

    if (!baseURL) {
        console.error("API_URL is not set: the appointment API cannot be reached");

        return NextResponse.json(
            { error: "appointment API is not configured" },
            { status: 503, headers: noStore() },
        );
    }

    let response: Response;

    try {
        response = await fetch(`${baseURL}${path}`, {
            ...init,
            cache: "no-store",
            signal: AbortSignal.timeout(API_TIMEOUT_MS),
        });
    } catch (err) {
        if (err instanceof DOMException && err.name === "TimeoutError") {
            console.error(`appointment API timed out: ${init.method ?? "GET"} ${pathWithoutQuery(path)}`);

            return apiTimeout();
        }

        console.error(`appointment API unreachable: ${init.method ?? "GET"} ${pathWithoutQuery(path)}`, err);

        return apiUnavailable();
    }

    // A successful cancellation has no body to forward.
    if (response.status === 204) {
        return new NextResponse(null, { status: 204, headers: noStore() });
    }

    return forwardJSON(response);
}

/**
 * Tells whether the request body is declared as JSON. Browsers can only send
 * a cross-site POST without a CORS preflight as text/plain, a form or with
 * no content type, so requiring JSON keeps other sites from making their
 * visitors' browsers call these routes.
 */
export function isJSONRequest(request: Request): boolean {
    const contentType = request.headers.get("content-type") ?? "";

    return contentType.split(";")[0].trim().toLowerCase() === "application/json";
}

/**
 * The answer of the admin route handlers when the request has no Clerk
 * session, so there is no token to forward to the Go API.
 */
export function unauthorized(): NextResponse {
    return NextResponse.json(
        { error: "unauthorized" },
        { status: 401, headers: noStore() },
    );
}

export function unsupportedMediaType(): NextResponse {
    return NextResponse.json(
        { error: "content type must be application/json" },
        { status: 415, headers: noStore() },
    );
}

/**
 * Responses carry customer data or change with every booking, so neither
 * the browser nor any CDN may cache them.
 */
function noStore(): Record<string, string> {
    return { "Cache-Control": "no-store" };
}

// Query strings can hold customer input, so they are left out of the logs.
function pathWithoutQuery(path: string): string {
    return path.split("?")[0];
}
