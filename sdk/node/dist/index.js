export class Faultwing {
    eventsUrl;
    apiKey;
    environment;
    release;
    timeoutMs;
    constructor(url, apiKey, options = {}) {
        if (!url.trim())
            throw new Error('url is required');
        if (!apiKey.trim())
            throw new Error('apiKey is required');
        if (!options.environment?.trim() && options.environment !== undefined) {
            throw new Error('environment is required');
        }
        const timeoutMs = options.timeoutMs ?? 2000;
        if (!Number.isSafeInteger(timeoutMs) || timeoutMs < 1 || timeoutMs > 2_147_483_647) {
            throw new Error('timeoutMs must be a whole number between 1 and 2147483647');
        }
        this.eventsUrl = `${url.trim().replace(/\/+$/, '')}/api/v1/events`;
        this.apiKey = apiKey.trim();
        this.environment = options.environment?.trim() ?? 'production';
        this.release = options.release?.trim() || undefined;
        this.timeoutMs = timeoutMs;
    }
    async captureException(error) {
        const exceptionType = error instanceof Error
            ? (error.name === 'Error' ? error.constructor.name : error.name)
            : 'Error';
        const message = error instanceof Error ? error.message : String(error);
        const payload = {
            exception_type: exceptionType,
            message: message || exceptionType,
            stacktrace: error instanceof Error ? (error.stack ?? '') : '',
            environment: this.environment,
            ...(this.release ? { release: this.release } : {}),
        };
        try {
            const response = await fetch(this.eventsUrl, {
                method: 'POST',
                headers: {
                    Authorization: `Bearer ${this.apiKey}`,
                    'Content-Type': 'application/json',
                    'User-Agent': 'faultwing-node/0.1.0',
                },
                body: JSON.stringify(payload),
                signal: AbortSignal.timeout(this.timeoutMs),
            });
            return response.ok;
        }
        catch {
            return false;
        }
    }
}
