import assert from 'node:assert/strict';
import test from 'node:test';

import { Faultwing } from '../dist/index.js';

test('sends an exception with monitoring context and a project key', async (t) => {
  let request;
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    request = { url, options };
    return { ok: true };
  });

  const client = new Faultwing('http://localhost:8080/', 'faultwing_test-key', {
    environment: 'development',
    release: '1.3.2',
    timeoutMs: 1500,
  });

  class DatabaseTimeoutError extends Error {}
  const delivered = await client.captureException(new DatabaseTimeoutError('connection timed out'));

  assert.equal(delivered, true);
  assert.equal(request.url, 'http://localhost:8080/api/v1/events');
  assert.equal(request.options.method, 'POST');
  assert.equal(request.options.headers.Authorization, 'Bearer faultwing_test-key');
  assert.ok(request.options.signal instanceof AbortSignal);
  const payload = JSON.parse(request.options.body);
  assert.match(payload.stacktrace, /Error: connection timed out/);
  assert.deepEqual({ ...payload, stacktrace: undefined }, {
    exception_type: 'DatabaseTimeoutError',
    message: 'connection timed out',
    stacktrace: undefined,
    environment: 'development',
    release: '1.3.2',
  });
});

test('returns false for HTTP failures', async (t) => {
  const client = new Faultwing('http://localhost:8080', 'faultwing_test-key');
  t.mock.method(globalThis, 'fetch', async () => ({ ok: false }));
  assert.equal(await client.captureException(new Error('broken')), false);
});

test('returns false when delivery fails', async (t) => {
  const client = new Faultwing('http://localhost:8080', 'faultwing_test-key');
  t.mock.method(globalThis, 'fetch', async () => { throw new Error('offline'); });
  assert.equal(await client.captureException(new Error('broken')), false);
});

test('rejects timeout values Node would not honor', () => {
  assert.throws(
    () => new Faultwing('http://localhost:8080', 'faultwing_test-key', { timeoutMs: 1.5 }),
    /timeoutMs/,
  );
  assert.throws(
    () => new Faultwing('http://localhost:8080', 'faultwing_test-key', { timeoutMs: 2_147_483_648 }),
    /timeoutMs/,
  );
});
