import { useCallback, useEffect, useRef, useState } from "react";
import {
  APIError,
  flytrap,
  type Issue,
  type IssueDetail,
  type IssueStatus,
  type Project,
  type Session,
} from "../api/flytrap";
import { IssueList } from "../components/IssueList";
import { IssuePanel } from "../components/IssuePanel";
import { Terrarium } from "../components/Terrarium";

const demoProject: Project = {
  id: 1,
  owner_id: 0,
  name: "Checkout API",
  created_at: "2026-09-18T08:00:00Z",
};

const demoNow = Date.now();
const ago = (milliseconds: number) => new Date(demoNow - milliseconds).toISOString();
const minute = 60_000;
const day = 86_400_000;

const demoIssues: Issue[] = [
  {
    id: 7,
    project_id: 1,
    fingerprint: "6bb1",
    exception_type: "DatabaseTimeoutError",
    message: "database connection timed out",
    stacktrace: "DatabaseTimeoutError: database connection timed out\n    at connect (db/client.go:42)",
    status: "open",
    event_count: 128,
    first_seen: ago(3 * day),
    last_seen: ago(2 * minute),
    created_at: ago(3 * day),
  },
  {
    id: 12,
    project_id: 1,
    fingerprint: "2a9f",
    exception_type: "UpstreamConnectionError",
    message: "payment service refused the connection",
    stacktrace: "UpstreamConnectionError: connection refused\n    at charge (payments/client.go:81)",
    status: "open",
    event_count: 42,
    first_seen: ago(2 * day),
    last_seen: ago(8 * minute),
    created_at: ago(2 * day),
  },
  {
    id: 18,
    project_id: 1,
    fingerprint: "d4e8",
    exception_type: "ValueError",
    message: "invalid order total",
    stacktrace: "ValueError: invalid order total\n    at validate (orders/validate.go:19)",
    status: "open",
    event_count: 9,
    first_seen: ago(day),
    last_seen: ago(24 * minute),
    created_at: ago(day),
  },
];

function fromHash() {
  const parameters = new URLSearchParams(location.hash.slice(1));
  return {
    project: Number(parameters.get("project")) || null,
    issue: Number(parameters.get("issue")) || null,
  };
}

function describe(error: unknown) {
  return error instanceof Error ? error.message : "Something went wrong. Please try again.";
}

function normalizedDetail(issue: IssueDetail): IssueDetail {
  return {
    ...issue,
    environments: issue.environments ?? [],
    releases: issue.releases ?? [],
  };
}

function NavGlyph({ type }: { type: "issues" | "terrarium" | "projects" }) {
  if (type === "issues") {
    return <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true"><ellipse cx="12" cy="13" rx="5" ry="6.5" /><path d="M9 7 7 4m8 3 2-3M7 11H3m18 0h-4M7 16l-3 3m13-3 3 3M9 11h6m-6 4h6" /></svg>;
  }
  if (type === "terrarium") {
    return <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 21V9m0 3C8 12 5 9 5 5c4 0 7 3 7 7Zm0 4c4 0 7-3 7-7-4 0-7 3-7 7Z" /></svg>;
  }
  return <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M3 7h7l2 2h9v10H3Z" /><path d="M3 7V5h7l2 2h9v2" /></svg>;
}

type DashboardProps = {
  session: Session | null;
  demo?: boolean;
  onLogout: () => Promise<void> | void;
  onSessionExpired: () => void;
};

type WorkspaceView = "issues" | "terrarium" | "projects";

export function Dashboard({
  session,
  demo = false,
  onLogout,
  onSessionExpired,
}: DashboardProps) {
  const initial = fromHash();
  const [projects, setProjects] = useState<Project[]>(demo ? [demoProject] : []);
  const [projectId, setProjectId] = useState<number | null>(initial.project ?? (demo ? demoProject.id : null));
  const [issues, setIssues] = useState<Issue[]>(demo ? demoIssues : []);
  const [habitatIssues, setHabitatIssues] = useState<Issue[]>(demo ? demoIssues : []);
  const [filter, setFilter] = useState<IssueStatus>("open");
  const [workspaceView, setWorkspaceView] = useState<WorkspaceView>("issues");
  const [selectedId, setSelectedId] = useState<number | null>(initial.issue ?? (demo ? demoIssues[0].id : null));
  const [detail, setDetail] = useState<IssueDetail | null>(null);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [loadingProjects, setLoadingProjects] = useState(!demo);
  const [loadingIssues, setLoadingIssues] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [newName, setNewName] = useState("");
  const [creating, setCreating] = useState(false);
  const [oneTimeKey, setOneTimeKey] = useState<string | null>(null);
  const [copyLabel, setCopyLabel] = useState("Copy key");
  const [updating, setUpdating] = useState(false);
  const [realtimeNotice, setRealtimeNotice] = useState<string | null>(null);
  const [realtimeState, setRealtimeState] = useState<"connecting" | "live" | "offline">(
    demo ? "live" : "connecting",
  );
  const [habitatOpen, setHabitatOpen] = useState(true);
  const requestVersion = useRef(0);
  const detailVersion = useRef(0);
  const issuesRef = useRef(issues);
  const filterRef = useRef(filter);
  const projectIdRef = useRef(projectId);
  const selectedIdRef = useRef(selectedId);

  issuesRef.current = issues;
  filterRef.current = filter;
  projectIdRef.current = projectId;
  selectedIdRef.current = selectedId;

  const expireIfNeeded = useCallback((value: unknown) => {
    if (value instanceof APIError && value.status === 401) {
      onSessionExpired();
      return true;
    }
    return false;
  }, [onSessionExpired]);

  const changeSelection = useCallback((
    project: number | null,
    issue: number | null,
    options: { clearProjectData?: boolean } = {},
  ) => {
    if (projectIdRef.current === project && selectedIdRef.current === issue) return;

    ++requestVersion.current;
    ++detailVersion.current;
    setDetail(null);
    setDetailError(null);
    setUpdating(false);
    setRealtimeNotice(null);
    setProjectId((current) => {
      if (current !== project && options.clearProjectData !== false) {
        setIssues([]);
        setHabitatIssues([]);
        setNextCursor(null);
      }
      return project;
    });
    setSelectedId(issue);

    const parameters = new URLSearchParams();
    if (project) parameters.set("project", String(project));
    if (issue) parameters.set("issue", String(issue));
    history.replaceState(null, "", `#${parameters}`);
  }, []);

  useEffect(() => {
    if (demo || !session) return;

    let active = true;
    flytrap.projects(session.token)
      .then((loaded) => {
        if (!active) return;
        setProjects(loaded);
        setProjectId((current) => (
          current && loaded.some((project) => project.id === current)
            ? current
            : (loaded[0]?.id ?? null)
        ));
      })
      .catch((caught) => {
        if (active && !expireIfNeeded(caught)) setError(describe(caught));
      })
      .finally(() => {
        if (active) setLoadingProjects(false);
      });

    return () => {
      active = false;
    };
  }, [demo, expireIfNeeded, session]);

  const loadIssues = useCallback(async (cursor?: string, append = false) => {
    if (demo || !session || !projectId) return;

    const version = ++requestVersion.current;
    append ? setLoadingMore(true) : setLoadingIssues(true);
    setError(null);

    try {
      const page = await flytrap.issues(session.token, projectId, filter, cursor);
      if (version !== requestVersion.current) return;

      const received = page.issues ?? [];
      setIssues((current) => (
        append
          ? [...current, ...received.filter((item) => !current.some((known) => known.id === item.id))]
          : received
      ));
      if (filter === "open") {
        setHabitatIssues((current) => (
          append
            ? [...current, ...received.filter((item) => !current.some((known) => known.id === item.id))]
            : received
        ));
      }
      setNextCursor(page.next_cursor ?? null);
    } catch (caught) {
      if (version === requestVersion.current && !expireIfNeeded(caught)) {
        setError(describe(caught));
      }
    } finally {
      if (version === requestVersion.current) {
        setLoadingIssues(false);
        setLoadingMore(false);
      }
    }
  }, [demo, expireIfNeeded, filter, projectId, session]);

  const loadIssuesRef = useRef(loadIssues);
  loadIssuesRef.current = loadIssues;

  useEffect(() => {
    setNextCursor(null);
    if (projectId) void loadIssues();
  }, [projectId, filter, loadIssues]);

  const loadDetail = useCallback(async () => {
    const version = ++detailVersion.current;
    if (!projectId || !selectedId) {
      setDetail(null);
      setLoadingDetail(false);
      return;
    }

    if (demo) {
      const issue = issues.find((item) => item.id === selectedId);
      setDetail(issue ? {
        ...issue,
        environments: ["production"],
        releases: ["v1.3.2"],
      } : null);
      return;
    }

    if (!session) return;
    setLoadingDetail(true);
    setDetailError(null);

    try {
      const loaded = normalizedDetail(await flytrap.issue(session.token, projectId, selectedId));
      if (version === detailVersion.current) setDetail(loaded);
    } catch (caught) {
      if (version === detailVersion.current && !expireIfNeeded(caught)) {
        setDetailError(describe(caught));
      }
    } finally {
      if (version === detailVersion.current) setLoadingDetail(false);
    }
  }, [demo, expireIfNeeded, issues, projectId, selectedId, session]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  useEffect(() => {
    if (demo || !session || !projectId) return;

    let socket: WebSocket | null = null;
    let retry: number | null = null;
    let cancelled = false;
    let retryDelay = 1_000;
    let connectedOnce = false;
    setRealtimeState("connecting");

    const refreshIssue = async (issueId: number) => {
      try {
        const loaded = normalizedDetail(await flytrap.issue(session.token, projectId, issueId));
        if (cancelled) return;

        const wasLoaded = issuesRef.current.some((issue) => issue.id === issueId);
        setIssues((current) => {
          if (!current.some((issue) => issue.id === issueId)) return current;
          if (loaded.status !== filterRef.current) return current.filter((issue) => issue.id !== issueId);
          return current.map((issue) => issue.id === issueId ? loaded : issue);
        });
        setHabitatIssues((current) => {
          const known = current.some((issue) => issue.id === issueId);
          if (loaded.status !== "open") return current.filter((issue) => issue.id !== issueId);
          if (known) return current.map((issue) => issue.id === issueId ? loaded : issue);
          return [loaded, ...current];
        });
        if (selectedIdRef.current === issueId) setDetail(loaded);
        if (!wasLoaded) setRealtimeNotice("New issue activity is available.");
      } catch (caught) {
        if (!cancelled && !expireIfNeeded(caught)) {
          setRealtimeNotice("An update arrived, but its issue could not be refreshed.");
        }
      }
    };

    const connect = () => {
      if (cancelled) return;
      setRealtimeState("connecting");
      socket = new WebSocket(flytrap.realtimeURL(projectId));
      socket.onopen = () => socket?.send(JSON.stringify({ token: session.token }));
      socket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as {
            type?: string;
            project_id?: number;
            issue_id?: number;
          };
          if (message.project_id !== projectId) return;
          if (message.type === "realtime.ready") {
            const reconnected = connectedOnce;
            connectedOnce = true;
            retryDelay = 1_000;
            setRealtimeState("live");
            if (reconnected) void loadIssuesRef.current();
          }
          if (message.type === "issue.updated" && message.issue_id) {
            void refreshIssue(message.issue_id);
          }
        } catch {
          // Ignore malformed messages and keep the connection alive.
        }
      };
      socket.onclose = (event) => {
        if (cancelled) return;
        setRealtimeState("offline");
        if (event.code === 1008) {
          setRealtimeNotice("Realtime updates are unavailable for this project.");
          return;
        }
        retry = window.setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, 15_000);
      };
    };

    connect();
    return () => {
      cancelled = true;
      if (retry) window.clearTimeout(retry);
      socket?.close();
    };
  }, [demo, expireIfNeeded, projectId, session]);

  async function createProject(event: React.FormEvent) {
    event.preventDefault();
    if (!session || !newName.trim()) return;
    setCreating(true);
    setError(null);

    try {
      const created = await flytrap.createProject(session.token, newName.trim());
      setProjects((current) => [...current, created.project]);
      setOneTimeKey(created.api_key);
      setCopyLabel("Copy key");
      setNewName("");
      setFilter("open");
      setWorkspaceView("issues");
      changeSelection(created.project.id, null);
    } catch (caught) {
      if (!expireIfNeeded(caught)) setError(describe(caught));
    } finally {
      setCreating(false);
    }
  }

  async function updateStatus(status: IssueStatus) {
    if (!projectId || !selectedId) return;
    const targetProject = projectId;
    const targetIssue = selectedId;
    const version = detailVersion.current;
    setUpdating(true);
    setDetailError(null);

    try {
      let updated: Issue;
      if (demo) {
        updated = {
          ...issues.find((issue) => issue.id === targetIssue)!,
          status,
          resolved_at: status === "resolved" ? new Date().toISOString() : undefined,
        };
      } else if (session) {
        updated = await flytrap.updateIssue(session.token, targetProject, targetIssue, status);
      } else {
        return;
      }

      if (
        version !== detailVersion.current
        || projectIdRef.current !== targetProject
        || selectedIdRef.current !== targetIssue
      ) return;
      setIssues((current) => {
        if (demo) {
          return current.map((issue) => issue.id === targetIssue ? { ...issue, ...updated } : issue);
        }
        return updated.status === filter
          ? current.map((issue) => issue.id === targetIssue ? { ...issue, ...updated } : issue)
          : current.filter((issue) => issue.id !== targetIssue);
      });
      setHabitatIssues((current) => (
        updated.status === "open"
          ? current.map((issue) => issue.id === targetIssue ? { ...issue, ...updated } : issue)
          : current.filter((issue) => issue.id !== targetIssue)
      ));
      setDetail((current) => current ? { ...current, ...updated } : current);
    } catch (caught) {
      if (version === detailVersion.current && !expireIfNeeded(caught)) {
        setDetailError(describe(caught));
      }
    } finally {
      if (version === detailVersion.current) setUpdating(false);
    }
  }

  async function copyKey() {
    if (!oneTimeKey) return;
    try {
      await navigator.clipboard.writeText(oneTimeKey);
      setCopyLabel("Copied");
    } catch {
      setCopyLabel("Select and copy manually");
    }
  }

  const visibleIssues = demo ? issues.filter((issue) => issue.status === filter) : issues;
  const currentProject = projects.find((project) => project.id === projectId);
  const userLabel = demo ? "Demo data" : session?.user.email;
  const viewTitle = workspaceView === "issues"
    ? "Issues"
    : workspaceView === "terrarium" ? "Terrarium" : "Projects";

  function showWorkspace(view: WorkspaceView) {
    setWorkspaceView(view);
    if (view === "terrarium") setHabitatOpen(true);
    if (view === "issues" && filter !== "open") {
      changeSelection(projectId, null);
      setFilter("open");
    }
  }

  function selectProject(nextProjectId: number | null) {
    if (demo && nextProjectId === demoProject.id) {
      setIssues(demoIssues);
      setHabitatIssues(demoIssues);
      setFilter("open");
      changeSelection(demoProject.id, demoIssues[0].id, { clearProjectData: false });
    } else {
      setFilter("open");
      changeSelection(nextProjectId, null);
    }
    setWorkspaceView("issues");
  }

  return (
    <main className="dashboard">
      <div className="flytrap-shell">
        <aside className="flytrap-sidebar" aria-label="FlyTrap navigation">
          <div className="flytrap-brand" aria-label="FlyTrap">
            <span className="brand-leaf" aria-hidden="true" />
            <strong>FlyTrap</strong>
          </div>

          <label className="sidebar-project-picker" htmlFor="project-picker">
            <span className="sr-only">Project</span>
            <select
              id="project-picker"
              value={projectId ?? ""}
              onChange={(event) => selectProject(Number(event.target.value) || null)}
              disabled={loadingProjects}
            >
              <option value="">{loadingProjects ? "Loading projects…" : "Choose a project"}</option>
              {projects.map((project) => (
                <option key={project.id} value={project.id}>{project.name}</option>
              ))}
            </select>
          </label>

          <nav className="flytrap-nav" aria-label="Workspace shortcuts">
            <button
              type="button"
              className={`flytrap-nav__item ${workspaceView === "issues" ? "flytrap-nav__item--active" : ""}`}
              aria-current={workspaceView === "issues" ? "page" : undefined}
              onClick={() => showWorkspace("issues")}
            >
              <NavGlyph type="issues" />
              Issues
            </button>
            <button
              type="button"
              className={`flytrap-nav__item ${workspaceView === "terrarium" ? "flytrap-nav__item--active" : ""}`}
              aria-current={workspaceView === "terrarium" ? "page" : undefined}
              onClick={() => showWorkspace("terrarium")}
            >
              <NavGlyph type="terrarium" />
              Terrarium
            </button>
            <button
              type="button"
              className={`flytrap-nav__item ${workspaceView === "projects" ? "flytrap-nav__item--active" : ""}`}
              aria-current={workspaceView === "projects" ? "page" : undefined}
              onClick={() => showWorkspace("projects")}
            >
              <NavGlyph type="projects" />
              Projects
            </button>
          </nav>

          <div className="flytrap-sidebar__spacer" />

          <div className="sidebar-account">
            <span className="sidebar-account__avatar" aria-hidden="true">
              {(userLabel?.[0] ?? "F").toUpperCase()}
            </span>
            <span title={userLabel}>{userLabel}</span>
          </div>
          <button type="button" className="sidebar-logout" onClick={() => void onLogout()}>
            {demo ? "Exit demo" : "Log out"}
          </button>
        </aside>

        <section className="flytrap-main" aria-label="Issues workspace">
          <header className="flytrap-main__header">
            <div>
              <p className="workspace-project">{currentProject?.name ?? "No project selected"}</p>
              <h1>{viewTitle}</h1>
            </div>
            <div className="workspace-actions">
              {demo && <span className="demo-pill">Demo · local data</span>}
              <span className={`live-pill live-pill--${realtimeState}`}>
                <i aria-hidden="true" />
                {realtimeState === "live" ? "Live" : realtimeState === "connecting" ? "Connecting" : "Offline"}
              </span>
            </div>
          </header>

          {error && <p role="alert" className="state-message state-message--error">{error}</p>}
          {realtimeNotice && (
            <p className="update-notice" role="status">
              <span>{realtimeNotice}</span>
              <button
                type="button"
                className="button button--small"
                onClick={() => {
                  setRealtimeNotice(null);
                  void loadIssues();
                }}
              >
                Refresh issues
              </button>
            </p>
          )}
          {oneTimeKey && (
            <section className="project-key" aria-live="polite">
              <strong>Copy this API key now. It will not be shown again.</strong>
              <code>{oneTimeKey}</code>
              <div>
                <button type="button" className="button" onClick={copyKey}>{copyLabel}</button>
                <button type="button" className="button" onClick={() => setOneTimeKey(null)}>I saved it</button>
              </div>
            </section>
          )}

          {workspaceView === "projects" ? (
            <section className="projects-workspace" aria-labelledby="projects-heading">
              <div className="projects-workspace__intro">
                <p className="eyebrow">Project workspace</p>
                <h2 id="projects-heading">Choose where errors should land.</h2>
                <p>Each project keeps its issues and ingestion key separate.</p>
              </div>

              <div className="project-cards">
                {projects.map((project) => (
                  <button
                    key={project.id}
                    type="button"
                    className={`project-card ${project.id === projectId ? "project-card--active" : ""}`}
                    onClick={() => selectProject(project.id)}
                  >
                    <span className="project-card__mark" aria-hidden="true"><NavGlyph type="projects" /></span>
                    <span>
                      <strong>{project.name}</strong>
                      <small>{project.id === projectId ? "Current project" : "Open project"}</small>
                    </span>
                    <span aria-hidden="true">›</span>
                  </button>
                ))}
              </div>

              {demo ? (
                <p className="projects-workspace__demo">
                  Checkout API is a local demonstration project. Exit the demo to create persistent projects.
                </p>
              ) : (
                <form className="project-create" onSubmit={createProject}>
                  <div>
                    <strong>Create a project</strong>
                    <span>You will receive its ingestion key once.</span>
                  </div>
                  <label>
                    <span>Project name</span>
                    <input
                      value={newName}
                      onChange={(event) => setNewName(event.target.value)}
                      maxLength={100}
                      placeholder="Payments API"
                      required
                    />
                  </label>
                  <button className="button button--primary" disabled={creating}>
                    {creating ? "Creating…" : "Create project"}
                  </button>
                </form>
              )}
            </section>
          ) : !projectId && !loadingProjects ? (
            <div className="workspace-empty">
              <strong>Create your first project</strong>
              <span>Open Projects in the sidebar to start receiving issues.</span>
              <button type="button" className="button" onClick={() => showWorkspace("projects")}>Open projects</button>
            </div>
          ) : (
            <div className={`flytrap-main__stack ${workspaceView === "terrarium" ? "flytrap-main__stack--terrarium" : ""}`}>
              <section id="terrarium" className="habitat-section">
                <div className="habitat-section__heading">
                  <span>{habitatIssues.length} loaded open issue{habitatIssues.length === 1 ? "" : "s"}</span>
                  <button
                    type="button"
                    className="quiet-button"
                    aria-expanded={habitatOpen}
                    onClick={() => setHabitatOpen((open) => !open)}
                  >
                    {habitatOpen ? "Hide habitat" : "Show habitat"}
                  </button>
                </div>
                {habitatOpen && (
                  <Terrarium
                    issues={habitatIssues}
                    selectedId={selectedId}
                    onSelect={(id) => changeSelection(projectId, id)}
                  />
                )}
              </section>

              {workspaceView === "issues" && <div id="issues" className="flytrap-issue-list-wrap">
                {loadingIssues ? (
                  <p className="state-message" aria-busy="true">Loading issues…</p>
                ) : (
                  <IssueList
                    issues={visibleIssues}
                    selectedId={selectedId}
                    filter={filter}
                    onFilter={(status) => {
                      changeSelection(projectId, null);
                      setFilter(status);
                    }}
                    onSelect={(id) => changeSelection(projectId, id)}
                    hasMore={Boolean(nextCursor)}
                    loadingMore={loadingMore}
                    onLoadMore={() => nextCursor && void loadIssues(nextCursor, true)}
                  />
                )}
              </div>}
            </div>
          )}
        </section>

        <IssuePanel
          issue={detail}
          loading={loadingDetail}
          error={detailError}
          updating={updating}
          onStatus={updateStatus}
        />
      </div>
    </main>
  );
}
