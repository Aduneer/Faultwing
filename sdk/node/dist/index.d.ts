export interface FaultwingOptions {
    environment?: string;
    release?: string;
    timeoutMs?: number;
}
export declare class Faultwing {
    private readonly eventsUrl;
    private readonly apiKey;
    private readonly environment;
    private readonly release?;
    private readonly timeoutMs;
    constructor(url: string, apiKey: string, options?: FaultwingOptions);
    captureException(error: unknown): Promise<boolean>;
}
