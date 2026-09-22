import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import type { IssueDetail, IssueStatus } from "../api/flytrap";
import {
  buildEditorURL,
  editorPresets,
  editorTemplate,
  jetbrainsProducts,
  readEditorSettings,
  readJetBrainsProjectName,
  readProjectSourceRoot,
  resolveSourcePath,
  saveEditorSettings,
  saveJetBrainsProjectName,
  saveProjectSourceRoot,
  sourceLocationFromStacktrace,
  sourceLocationLabel,
  validateEditorTemplate,
  type EditorPreset,
  type EditorSettings,
} from "../editor";

function fullDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

type IssuePanelProps = {
  issue: IssueDetail | null;
  loading: boolean;
  error: string | null;
  updating: boolean;
  onStatus: (status: IssueStatus) => void;
};

const initialEditorSettings: EditorSettings = {
  preset: "vscode",
  customTemplate: "",
  jetbrainsProduct: "idea",
};

function folderName(path: string) {
  return path.trim().replace(/[\\/]+$/, "").split(/[\\/]/).pop() ?? "";
}

function EditorActions({ issue }: { issue: IssueDetail }) {
  const source = sourceLocationFromStacktrace(issue.stacktrace ?? "");
  const [settings, setSettings] = useState<EditorSettings | null>(readEditorSettings);
  const [sourceRoot, setSourceRoot] = useState(() => readProjectSourceRoot(issue.project_id));
  const [jetbrainsProjectName, setJetBrainsProjectName] = useState(() => readJetBrainsProjectName(issue.project_id));
  const [configuring, setConfiguring] = useState(false);
  const [draftSettings, setDraftSettings] = useState<EditorSettings>(settings ?? initialEditorSettings);
  const [draftRoot, setDraftRoot] = useState(sourceRoot);
  const [draftJetBrainsProjectName, setDraftJetBrainsProjectName] = useState(jetbrainsProjectName);
  const [configurationError, setConfigurationError] = useState<string | null>(null);
  const [copyLabel, setCopyLabel] = useState("Copy location");

  useEffect(() => {
    const nextRoot = readProjectSourceRoot(issue.project_id);
    const nextJetBrainsProjectName = readJetBrainsProjectName(issue.project_id);
    setSourceRoot(nextRoot);
    setDraftRoot(nextRoot);
    setJetBrainsProjectName(nextJetBrainsProjectName);
    setDraftJetBrainsProjectName(nextJetBrainsProjectName);
    setCopyLabel("Copy location");
  }, [issue.project_id, issue.id]);

  useEffect(() => {
    if (!configuring) return;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setConfiguring(false);
    };
    window.addEventListener("keydown", closeOnEscape);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", closeOnEscape);
    };
  }, [configuring]);

  if (!source) {
    return (
      <div className="editor-actions editor-actions--unavailable">
        <span>No file location was found in this stack trace.</span>
        <button type="button" className="button button--small" disabled>Open in Editor</button>
      </div>
    );
  }

  const sourceLocation = source;
  const fullPath = resolveSourcePath(sourceLocation.path, sourceRoot);
  const editorURL = settings && fullPath
    ? buildEditorURL(settings, fullPath, sourceLocation, jetbrainsProjectName)
    : null;

  function beginConfiguration() {
    setDraftSettings(settings ?? initialEditorSettings);
    setDraftRoot(sourceRoot);
    setDraftJetBrainsProjectName(jetbrainsProjectName || folderName(sourceRoot));
    setConfigurationError(null);
    setConfiguring(true);
  }

  function finishConfiguration(event: React.FormEvent) {
    event.preventDefault();
    const template = editorTemplate(draftSettings);
    const templateError = validateEditorTemplate(template);
    if (templateError) {
      setConfigurationError(templateError);
      return;
    }
    if (!resolveSourcePath(sourceLocation.path, draftRoot)) {
      setConfigurationError("Enter the absolute path to this project's source checkout.");
      return;
    }
    if (draftSettings.preset === "jetbrains" && !draftJetBrainsProjectName.trim()) {
      setConfigurationError("Enter the project name shown by your JetBrains IDE.");
      return;
    }

    if (
      !saveEditorSettings(draftSettings)
      || !saveProjectSourceRoot(issue.project_id, draftRoot.trim())
      || !saveJetBrainsProjectName(issue.project_id, draftJetBrainsProjectName.trim())
    ) {
      setConfigurationError("Your browser prevented FlyTrap from saving these local settings.");
      return;
    }
    setSettings(draftSettings);
    setSourceRoot(draftRoot.trim());
    setJetBrainsProjectName(draftJetBrainsProjectName.trim());
    setConfiguring(false);
  }

  async function copyLocation() {
    const value = `${fullPath ?? sourceLocation.path}:${sourceLocation.line}${sourceLocation.column > 1 ? `:${sourceLocation.column}` : ""}`;
    try {
      await navigator.clipboard.writeText(value);
      setCopyLabel("Copied");
    } catch {
      setCopyLabel("Copy failed");
    }
  }

  return (
    <>
      <div className="editor-actions">
        <code title={sourceLocationLabel(sourceLocation)}>{sourceLocationLabel(sourceLocation)}</code>
        <div>
          {editorURL ? (
            <a className="button button--small editor-open" href={editorURL}>Open in Editor</a>
          ) : (
            <button type="button" className="button button--small editor-open" onClick={beginConfiguration}>
              Open in Editor
            </button>
          )}
          <button type="button" className="button button--small" onClick={copyLocation}>{copyLabel}</button>
          <button type="button" className="editor-settings-button" onClick={beginConfiguration}>Editor settings</button>
        </div>
      </div>

      {configuring && createPortal((
        <div className="editor-dialog-backdrop" role="presentation" onMouseDown={(event) => {
          if (event.currentTarget === event.target) setConfiguring(false);
        }}>
          <section className="editor-dialog" role="dialog" aria-modal="true" aria-labelledby="editor-dialog-title">
            <header>
              <div>
                <p className="eyebrow">Local integration</p>
                <h3 id="editor-dialog-title">Open source in your editor</h3>
              </div>
              <button type="button" className="editor-dialog__close" aria-label="Close editor settings" onClick={() => setConfiguring(false)}>×</button>
            </header>

            <p className="editor-dialog__intro">
              These settings stay in this browser. FlyTrap never sends your local source path to the server.
            </p>

            <form onSubmit={finishConfiguration}>
              <label>
                <span>Editor</span>
                <select
                  aria-label="Editor"
                  autoFocus
                  value={draftSettings.preset}
                  onChange={(event) => {
                    const preset = event.target.value as EditorPreset;
                    setDraftSettings((current) => ({ ...current, preset }));
                    if (preset === "jetbrains" && !draftJetBrainsProjectName) {
                      setDraftJetBrainsProjectName(folderName(draftRoot));
                    }
                  }}
                >
                  {editorPresets.map((preset) => <option key={preset.value} value={preset.value}>{preset.label}</option>)}
                </select>
              </label>

              {draftSettings.preset === "jetbrains" && (
                <div className="editor-dialog__jetbrains">
                  <label>
                    <span>JetBrains IDE</span>
                    <select
                      aria-label="JetBrains IDE"
                      value={draftSettings.jetbrainsProduct}
                      onChange={(event) => setDraftSettings((current) => ({
                        ...current,
                        jetbrainsProduct: event.target.value,
                      }))}
                    >
                      {jetbrainsProducts.map((product) => <option key={product.value} value={product.value}>{product.label}</option>)}
                    </select>
                  </label>
                  <label>
                    <span>IDE project name</span>
                    <input
                      aria-label="IDE project name"
                      value={draftJetBrainsProjectName}
                      onChange={(event) => setDraftJetBrainsProjectName(event.target.value)}
                      placeholder="checkout-api"
                      required
                    />
                    <small>This is usually the checkout folder name. JetBrains Toolbox must be installed.</small>
                  </label>
                </div>
              )}

              {draftSettings.preset === "custom" && (
                <label>
                  <span>Editor URL template</span>
                  <input
                    value={draftSettings.customTemplate}
                    onChange={(event) => setDraftSettings((current) => ({
                      ...current,
                      customTemplate: event.target.value,
                    }))}
                    placeholder="my-editor://open?file={pathEncoded}&line={line}"
                    required
                  />
                  <small>Available placeholders: {"{path}"}, {"{pathEncoded}"}, {"{pathNoLeadingSlash}"}, {"{line}"}, and {"{column}"}.</small>
                </label>
              )}

              <label>
                <span>Local source root for this project</span>
                <input
                  value={draftRoot}
                  onChange={(event) => setDraftRoot(event.target.value)}
                  placeholder="/home/you/code/checkout-api"
                  required={!resolveSourcePath(sourceLocation.path, "")}
                />
                <small>Maps {sourceLocation.path} to the copy checked out on this computer.</small>
              </label>

              {configurationError && <p className="editor-dialog__error" role="alert">{configurationError}</p>}

              <div className="editor-dialog__actions">
                <button type="button" className="button" onClick={() => setConfiguring(false)}>Cancel</button>
                <button type="submit" className="button button--primary">Save settings</button>
              </div>
            </form>
          </section>
        </div>
      ), document.body)}
    </>
  );
}

export function IssuePanel({ issue, loading, error, updating, onStatus }: IssuePanelProps) {
  if (loading) {
    return (
      <aside className="issue-panel issue-panel--state" aria-busy="true">
        <span className="panel-loader" aria-hidden="true" />
        <p>Loading issue…</p>
      </aside>
    );
  }

  if (error) {
    return (
      <aside className="issue-panel issue-panel--state">
        <p className="state-message state-message--error">{error}</p>
      </aside>
    );
  }

  if (!issue) {
    return (
      <aside className="issue-panel issue-panel--state">
        <span className="empty-specimen" aria-hidden="true">⌁</span>
        <strong>Select an issue</strong>
        <p>Choose an insect or an issue row to inspect its details.</p>
      </aside>
    );
  }

  const actions: Array<[IssueStatus, string]> = issue.status === "open"
    ? [["resolved", "Resolve"], ["ignored", "Ignore"]]
    : [["open", "Reopen"]];

  return (
    <aside className="issue-panel" aria-label="Issue detail">
      <header className="issue-panel__header">
        <h2>{issue.exception_type}</h2>
        <p>{issue.message}</p>
      </header>

      <div className="issue-panel__pills">
        <span className={`status-pill status-pill--${issue.status}`}>
          <i aria-hidden="true" />
          {issue.status}
        </span>
        {issue.environments.map((environment) => <span key={environment}>{environment}</span>)}
        {issue.releases.map((release) => <span key={release}>{release}</span>)}
      </div>

      <dl className="detail-meta">
        <div className="detail-meta__events">
          <dt>Events</dt>
          <dd>{issue.event_count}</dd>
        </div>
        <div>
          <dt>First seen</dt>
          <dd><time dateTime={issue.first_seen}>{fullDate(issue.first_seen)}</time></dd>
        </div>
        <div>
          <dt>Last seen</dt>
          <dd><time dateTime={issue.last_seen}>{fullDate(issue.last_seen)}</time></dd>
        </div>
      </dl>

      <section className="stacktrace-wrap">
        <h3>Stack trace</h3>
        <pre className="stacktrace">{issue.stacktrace || "No stack trace received."}</pre>
      </section>

      <EditorActions issue={issue} />

      <div className="issue-panel__actions">
        {actions.map(([status, label]) => (
          <button
            key={status}
            type="button"
            className={`button ${status === "resolved" || status === "open" ? "button--primary" : ""}`}
            disabled={updating}
            onClick={() => onStatus(status)}
          >
            {updating ? "Saving…" : label}
          </button>
        ))}
      </div>
    </aside>
  );
}
