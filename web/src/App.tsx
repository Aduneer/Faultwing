import { useCallback, useEffect, useState } from "react";
import { APIError, flytrap, type Session } from "./api/flytrap";
import { Dashboard } from "./pages/Dashboard";

const sessionKey = "flytrap.session";

function storedSession(): Session | null {
  try {
    const value = sessionStorage.getItem(sessionKey);
    if (!value) return null;

    const session = JSON.parse(value) as Session;
    const expiresAt = Date.parse(session.expires_at);
    if (!session.token || !Number.isFinite(expiresAt) || expiresAt <= Date.now()) {
      sessionStorage.removeItem(sessionKey);
      return null;
    }
    return session;
  } catch {
    sessionStorage.removeItem(sessionKey);
    return null;
  }
}

export default function App() {
  const [session, setSession] = useState<Session | null>(storedSession);
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [demo, setDemo] = useState(false);

  const sessionExpired = useCallback(() => {
    sessionStorage.removeItem(sessionKey);
    setSession(null);
    setError("Your session expired. Please log in again.");
  }, []);

  useEffect(() => {
    if (!session) return;

    const remaining = Date.parse(session.expires_at) - Date.now();
    if (remaining <= 0) {
      sessionExpired();
      return;
    }

    const timer = window.setTimeout(sessionExpired, Math.min(remaining, 2_147_483_647));
    return () => window.clearTimeout(timer);
  }, [session, sessionExpired]);

  function finishLogin(next: Session) {
    sessionStorage.setItem(sessionKey, JSON.stringify(next));
    setSession(next);
    setDemo(false);
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);

    try {
      if (mode === "register") {
        await flytrap.register(email, password);
        setMode("login");
        setError("Account created. Sign in to continue.");
      } else {
        finishLogin(await flytrap.login(email, password));
      }
    } catch (caught) {
      setError(caught instanceof APIError ? caught.message : "Unable to reach FlyTrap. Please try again.");
    } finally {
      setBusy(false);
    }
  }

  async function logout() {
    if (demo || !session) {
      setDemo(false);
      return;
    }

    const token = session.token;
    sessionStorage.removeItem(sessionKey);
    setSession(null);
    try {
      await flytrap.logout(token);
    } catch (caught) {
      setError(
        `You were signed out locally, but the server session could not be revoked: ${
          caught instanceof Error ? caught.message : "network error"
        }.`,
      );
    }
  }

  if (session || demo) {
    return (
      <Dashboard
        session={session}
        demo={demo}
        onLogout={logout}
        onSessionExpired={sessionExpired}
      />
    );
  }

  const accountCreated = error?.startsWith("Account created") ?? false;
  return (
    <main className="auth-page">
      <section className="auth-card">
        <div className="auth-brand">
          <span className="brand-leaf" aria-hidden="true" />
          FlyTrap
        </div>
        <p className="eyebrow">Error monitoring, naturally organized</p>
        <h1>{mode === "login" ? "Welcome back." : "Create your account."}</h1>
        <p className="auth-card__intro">
          Track grouped exceptions across your projects and see problems as they happen.
        </p>

        {error && (
          <p
            className={`state-message ${accountCreated ? "" : "state-message--error"}`}
            role={accountCreated ? "status" : "alert"}
          >
            {error}
          </p>
        )}

        <form onSubmit={submit}>
          <label>
            Email
            <input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </label>
          <label>
            Password
            <input
              type="password"
              minLength={8}
              maxLength={72}
              autoComplete={mode === "login" ? "current-password" : "new-password"}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </label>
          <button className="button button--primary auth-card__submit" disabled={busy}>
            {busy ? "Please wait…" : mode === "login" ? "Log in" : "Create account"}
          </button>
        </form>

        <button
          type="button"
          className="auth-card__switch"
          onClick={() => {
            setMode(mode === "login" ? "register" : "login");
            setError(null);
          }}
        >
          {mode === "login" ? "Need an account? Register" : "Already have an account? Log in"}
        </button>
        <button type="button" className="auth-card__demo" onClick={() => setDemo(true)}>
          Explore the interactive demo
        </button>
      </section>
    </main>
  );
}
