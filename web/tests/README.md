# Browser checks

The Playwright suite intentionally checks user-visible behavior and a small
set of network contracts, rather than mirroring every DOM node. Run it with
`npm run test:e2e`; install the cached browser with
`npx playwright install chromium` if the executable is not available.
