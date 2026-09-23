# SDKs

Faultwing has small Python and Node.js clients for sending exceptions to
`POST /api/v1/events`. Create a project in the dashboard and copy its API key
when it appears; the plaintext key is shown only once. The API and worker must
both be running for an accepted event to appear as an issue.

Neither client is published to a package registry yet. Install them from a
local clone of this repository. Both clients return whether the request
succeeded; `true` means the API accepted the event into its durable queue, not
that the worker has already processed it. Delivery failures return `false`.

## Keep the API key in your application environment

Pass the key to the client when your application starts. The SDKs do not read
`.env` files themselves. During local development, your application or runtime
can load one; in a deployed application, use its normal secret configuration.
Never put a project API key in browser code or commit it to Git.

For example, a local `.env` file could contain:

```dotenv
FAULTWING_URL=http://localhost:8080
FAULTWING_API_KEY=faultwing_your_api_key
```

Use the key for the project that should receive these errors. If the key was
lost, there is currently no endpoint to reveal, rotate, or revoke it; see the
[API guide](api.md#projects) for the current local-development limitation.

## Python

Requires Python 3.10 or newer. From the Faultwing repository root:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -e sdk/python
```

In your server-side Python application:

```python
import os
from faultwing import Faultwing

client = Faultwing(
    os.environ.get("FAULTWING_URL", "http://localhost:8080"),
    os.environ["FAULTWING_API_KEY"],
    environment="development",
    release="0.1.0",
)

try:
    raise RuntimeError("the example service stopped responding")
except RuntimeError as error:
    queued = client.capture_exception(error)
```

Set `FAULTWING_API_KEY` in the process environment before starting your app.
Python does not load a `.env` file automatically. The call has a two-second
timeout by default; use the `timeout` constructor option to change it.

## Node.js / TypeScript

Requires Node.js 22 or newer. From your Node application's directory, install
the SDK using the path to your Faultwing clone:

```bash
npm install /path/to/Faultwing/sdk/node
```

The local install includes compiled JavaScript and TypeScript declarations.
In a server-side JavaScript or TypeScript application:

```js
import { Faultwing } from 'faultwing-node';

const apiKey = process.env.FAULTWING_API_KEY;
if (!apiKey) throw new Error('FAULTWING_API_KEY is required');

const client = new Faultwing(
  process.env.FAULTWING_URL ?? 'http://localhost:8080',
  apiKey,
  { environment: 'development', release: '0.1.0' },
);

try {
  throw new Error('the example service stopped responding');
} catch (error) {
  const queued = await client.captureException(error);
}
```

`captureException` is asynchronous, so await it before the process exits. The
default timeout is two seconds; set `timeoutMs` to change it. Node.js can load
the example `.env` file when starting your application with
`node --env-file=.env app.mjs`.

These clients are intended for server-side use. Their package versions start
at `0.1.0` independently of the repository's release tags; the Node SDK was
added after the repository's `v0.1.0` release.
